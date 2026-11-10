package localsend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"golang.org/x/net/ipv4"
)

// DiscoveryConfig configures multicast discovery.
type DiscoveryConfig struct {
	Client *Client
	// Info is this device as it announces itself. It must carry the
	// server's port and protocol.
	Info func() Device
	// Port is the multicast port, DefaultPort when 0.
	Port int
	// Group is the IPv4 multicast group, DefaultMulticastGroup when nil.
	Group net.IP
	// OnDevice hears a peer that answered.
	OnDevice func(Device)
	// OnError hears failures that do not belong to one probe.
	OnError func(error)
}

// Discovery announces this device over IPv4 multicast and answers other
// announcements with the HTTP register request. It keeps no device list;
// the app merges what it hears.
type Discovery struct {
	cfg   DiscoveryConfig
	conn  *ipv4.PacketConn
	pc    net.PacketConn
	ifis  []net.Interface
	group net.IP
	port  int

	mu      sync.Mutex
	closed  bool
	probing map[string]bool
	once    sync.Once
}

// NewDiscovery makes a discovery; Start begins listening.
func NewDiscovery(cfg DiscoveryConfig) *Discovery {
	group := cfg.Group
	if group == nil {
		group = net.ParseIP(DefaultMulticastGroup).To4()
	}
	port := cfg.Port
	if port == 0 {
		port = DefaultPort
	}
	return &Discovery{cfg: cfg, group: group, port: port, probing: map[string]bool{}}
}

// announcement is the multicast message, with the legacy announce flag.
type announcement struct {
	Device
	Announce bool `json:"announce"`
}

// Start binds the multicast socket and begins hearing announcements. It
// binds LocalSend's port when it is free, and an ephemeral one else: this
// device still announces, and peers answer by registering with its
// server, which is how it discovers them.
func (d *Discovery) Start(ctx context.Context) error {
	pc, err := listenMulticast(d.port)
	if err != nil {
		if d.cfg.OnError != nil {
			d.cfg.OnError(fmt.Errorf("localsend: multicast port %d is busy, using an ephemeral one: %w", d.port, err))
		}
		pc, err = listenMulticast(0)
		if err != nil {
			return fmt.Errorf("localsend: opening multicast socket: %w", err)
		}
	}
	p := ipv4.NewPacketConn(pc)
	_ = p.SetMulticastTTL(1)
	_ = p.SetMulticastLoopback(true)
	_ = p.SetControlMessage(ipv4.FlagInterface, true)
	d.conn, d.pc = p, pc
	d.ifis = multicastInterfaces()
	joined := 0
	for _, ifi := range d.ifis {
		if err := p.JoinGroup(&ifi, &net.UDPAddr{IP: d.group}); err == nil {
			joined++
		}
	}
	// A machine whose interfaces say nothing still tries the default one.
	if joined == 0 {
		_ = p.JoinGroup(nil, &net.UDPAddr{IP: d.group})
	}
	go d.readLoop()
	return nil
}

// Stop ends discovery and its socket.
func (d *Discovery) Stop() {
	d.once.Do(func() {
		d.mu.Lock()
		d.closed = true
		d.mu.Unlock()
		if d.pc != nil {
			d.pc.Close()
		}
	})
}

// Self is the device as it announces itself.
func (d *Discovery) Self() Device {
	var dev Device
	if d.cfg.Info != nil {
		dev = d.cfg.Info()
	}
	if dev.Version == "" {
		dev.Version = Version
	}
	if dev.Protocol == "" {
		dev.Protocol = ProtocolHTTPS
	}
	return dev
}

// Announce sends one announcement on every multicast interface.
func (d *Discovery) Announce(ctx context.Context) error {
	if d.conn == nil {
		return errors.New("localsend: discovery is not running")
	}
	payload, err := json.Marshal(announcement{Device: d.Self(), Announce: true})
	if err != nil {
		return err
	}
	target := &net.UDPAddr{IP: d.group, Port: d.port}
	if len(d.ifis) == 0 {
		_, _ = d.conn.WriteTo(payload, nil, target)
		return nil
	}
	for i := range d.ifis {
		if err := ctx.Err(); err != nil {
			return err
		}
		_ = d.conn.SetMulticastInterface(&d.ifis[i])
		_, _ = d.conn.WriteTo(payload, nil, target)
	}
	return nil
}

// AnnounceBurst announces three times, as LocalSend does, so a lost
// datagram or a peer that just started does not hide this device.
func (d *Discovery) AnnounceBurst(ctx context.Context) {
	for _, delay := range []time.Duration{0, 500 * time.Millisecond, 2 * time.Second} {
		if delay > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
		}
		_ = d.Announce(ctx)
	}
}

// Scan probes the local IPv4 networks, for devices that do not answer
// multicast, as LocalSend's legacy scan does.
func (d *Discovery) Scan(ctx context.Context) {
	self := d.Self()
	protocol := self.Protocol
	if protocol == "" {
		protocol = ProtocolHTTPS
	}
	port := self.Port
	if port == 0 {
		port = DefaultPort
	}
	hosts := scanHosts()
	if len(hosts) == 0 {
		return
	}
	workers := 32
	jobs := make(chan net.IP)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range jobs {
				if ctx.Err() != nil {
					return
				}
				probeCtx, cancel := context.WithTimeout(ctx, 1200*time.Millisecond)
				_, _ = d.register(probeCtx, ip.String(), port, protocol)
				cancel()
			}
		}()
	}
	for _, ip := range hosts {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return
		case jobs <- ip:
		}
	}
	close(jobs)
	wg.Wait()
}

// Probe registers with one known address, such as a favorite.
func (d *Discovery) Probe(ctx context.Context, ip string, port int, protocol Protocol) (Device, bool) {
	if port == 0 {
		port = DefaultPort
	}
	if protocol == "" {
		protocol = ProtocolHTTPS
	}
	return d.register(ctx, ip, port, protocol)
}

// register answers an announcement and delivers a verified device.
func (d *Discovery) register(ctx context.Context, ip string, port int, protocol Protocol) (Device, bool) {
	if d.cfg.Client == nil {
		return Device{}, false
	}
	key := fmt.Sprintf("%s/%s/%d/%s", ip, protocol, port, d.Self().Fingerprint)
	d.mu.Lock()
	if d.probing[key] {
		d.mu.Unlock()
		return Device{}, false
	}
	d.probing[key] = true
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		delete(d.probing, key)
		d.mu.Unlock()
	}()
	at := Device{IP: ip, Port: port, Protocol: protocol}
	dev, err := d.cfg.Client.Register(ctx, at, d.Self())
	if err != nil {
		return Device{}, false
	}
	if dev.Fingerprint == "" || sameFingerprint(dev.Fingerprint, d.Self().Fingerprint) {
		return Device{}, false
	}
	dev.IP, dev.Port, dev.Protocol = ip, port, protocol
	dev.LastSeen = time.Now()
	if d.cfg.OnDevice != nil {
		d.cfg.OnDevice(dev)
	}
	return dev, true
}

// readLoop receives announcements until Stop.
func (d *Discovery) readLoop() {
	buf := make([]byte, 64<<10)
	for {
		n, _, src, err := d.conn.ReadFrom(buf)
		if err != nil {
			d.mu.Lock()
			closed := d.closed
			d.mu.Unlock()
			if closed {
				return
			}
			continue
		}
		udp, ok := src.(*net.UDPAddr)
		if !ok {
			continue
		}
		var msg announcement
		if err := json.Unmarshal(buf[:n], &msg); err != nil {
			continue
		}
		if msg.Fingerprint == "" || d.isSelf(msg.Device) {
			continue
		}
		ip := udp.IP.String()
		port := msg.Port
		if port == 0 {
			port = DefaultPort
		}
		protocol := msg.Protocol
		if protocol != ProtocolHTTP && protocol != ProtocolHTTPS {
			protocol = ProtocolHTTPS
		}
		if msg.Announce {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				_, _ = d.register(ctx, ip, port, protocol)
			}()
			continue
		}
		// A protocol 2.0 peer answered over multicast instead of
		// registering; it is already verified enough to show.
		dev := msg.Device
		dev.IP, dev.Port, dev.Protocol, dev.LastSeen = ip, port, protocol, time.Now()
		if d.cfg.OnDevice != nil {
			d.cfg.OnDevice(dev)
		}
	}
}

// isSelf reports whether dev is this device's own announcement.
func (d *Discovery) isSelf(dev Device) bool {
	self := d.Self()
	return sameFingerprint(dev.Fingerprint, self.Fingerprint) ||
		(d.cfg.Client != nil && sameFingerprint(dev.Fingerprint, d.cfg.Client.id.HTTPFingerprint))
}

// sameFingerprint compares identities case-insensitively.
func sameFingerprint(a, b string) bool {
	return a != "" && b != "" && len(a) == len(b) &&
		((a == b) || equalFoldASCII(a, b))
}

// equalFoldASCII compares ASCII case-insensitively without allocation.
func equalFoldASCII(a, b string) bool {
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// multicastInterfaces are the up, non-loopback IPv4 interfaces that can
// multicast.
func multicastInterfaces() []net.Interface {
	all, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []net.Interface
	for _, ifi := range all {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 || ifi.Flags&net.FlagMulticast == 0 {
			continue
		}
		addrs, err := ifi.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				out = append(out, ifi)
				break
			}
		}
	}
	return out
}

// scanHosts lists the IPv4 hosts on this machine's local networks, at
// most 512 in all.
func scanHosts() []net.IP {
	ifis, err := net.Interfaces()
	if err != nil {
		return nil
	}
	own := map[string]bool{}
	var hosts []net.IP
	seen := map[string]bool{}
	for _, ifi := range ifis {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifi.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipnet.IP.To4()
			if ip4 == nil || ip4.IsLoopback() || ip4.IsLinkLocalUnicast() {
				continue
			}
			own[ip4.String()] = true
			mask := ipnet.Mask
			if len(mask) == 16 {
				mask = mask[12:]
			}
			ones, bits := mask.Size()
			if bits != 32 || ones < 8 {
				continue
			}
			// Probe a /24 around this address for wide networks, so a
			// /16 does not take minutes.
			base := ip4.Mask(net.CIDRMask(24, 32))
			for i := 1; i < 255; i++ {
				next := make(net.IP, 4)
				copy(next, base)
				next[3] = byte(i)
				if !ipnet.Contains(next) && ones >= 24 {
					// Outside a network narrower than /24 only when its
					// mask was odd; skip it.
					continue
				}
				if seen[next.String()] {
					continue
				}
				seen[next.String()] = true
				hosts = append(hosts, next)
				if len(hosts) >= 512 {
					break
				}
			}
		}
	}
	sort.Slice(hosts, func(i, j int) bool { return hosts[i].String() < hosts[j].String() })
	return hosts
}

// listenMulticast binds the IPv4 UDP port, sharing it with another
// LocalSend app where the platform allows.
func listenMulticast(port int) (net.PacketConn, error) {
	lc := net.ListenConfig{Control: reuseControl}
	return lc.ListenPacket(context.Background(), "udp4", fmt.Sprintf(":%d", port))
}
