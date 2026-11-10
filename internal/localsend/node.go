package localsend

import (
	"context"
	"errors"
	"net"
	"strconv"
	"sync/atomic"
)

var errNoIdentity = errors.New("localsend: no identity")

// NodeConfig configures a running LocalSend node: its receiver and its
// discovery.
type NodeConfig struct {
	Identity *Identity
	// Alias is the name peers see; it may change while the node runs.
	Alias func() string
	// Port is the server's port, DefaultPort when 0.
	Port int
	// Addr overrides the server's listen address, for tests.
	Addr     string
	Protocol Protocol
	// DeviceModel and DeviceType describe this device to peers.
	DeviceModel string
	DeviceType  DeviceType
	// PIN protects receiving when it returns a nonempty string.
	PIN             func() string
	VerifyChecksums bool

	OnRegister   func(Device)
	OnPrepare    func(*IncomingRequest)
	OnProgress   func(Progress)
	OnSessionEnd func(SessionResult)
	OnDevice     func(Device)
	OnError      func(error)
}

// Node is a receiver and its discovery, the LocalSend side of an app.
type Node struct {
	cfg       NodeConfig
	identity  *Identity
	client    *Client
	server    *Server
	discovery *Discovery
	port      atomic.Int64
	// DiscoveryErr is why discovery did not start, when it did not; the
	// receiver still works then.
	DiscoveryErr error
}

// StartNode listens and begins discovery.
func StartNode(cfg NodeConfig) (*Node, error) {
	if cfg.Identity == nil {
		return nil, errNoIdentity
	}
	if cfg.Port == 0 {
		cfg.Port = DefaultPort
	}
	if cfg.Protocol != ProtocolHTTP && cfg.Protocol != ProtocolHTTPS {
		cfg.Protocol = ProtocolHTTPS
	}
	n := &Node{cfg: cfg, identity: cfg.Identity}
	n.port.Store(int64(cfg.Port))
	n.client = NewClient(cfg.Identity)
	addr := cfg.Addr
	if addr == "" {
		addr = ":" + strconv.Itoa(cfg.Port)
	}
	n.server = NewServer(ServerConfig{
		Identity:        cfg.Identity,
		Addr:            addr,
		Info:            n.Self,
		PIN:             cfg.PIN,
		VerifyChecksums: cfg.VerifyChecksums,
		OnRegister:      cfg.OnRegister,
		OnPrepare:       cfg.OnPrepare,
		OnProgress:      cfg.OnProgress,
		OnSessionEnd:    cfg.OnSessionEnd,
		OnError:         cfg.OnError,
	})
	if err := n.server.Start(); err != nil {
		n.client.Close()
		return nil, err
	}
	n.port.Store(int64(n.server.Port()))
	n.discovery = NewDiscovery(DiscoveryConfig{
		Client:   n.client,
		Info:     n.Self,
		Port:     cfg.Port,
		OnDevice: cfg.OnDevice,
		OnError:  cfg.OnError,
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := n.discovery.Start(ctx); err != nil {
		n.DiscoveryErr = err
	}
	return n, nil
}

// Self is this device as peers see it.
func (n *Node) Self() Device {
	alias := "LocalSend"
	if n.cfg.Alias != nil {
		if a := n.cfg.Alias(); a != "" {
			alias = a
		}
	}
	return Device{
		Alias:       alias,
		Version:     Version,
		DeviceModel: n.cfg.DeviceModel,
		DeviceType:  n.cfg.DeviceType,
		Fingerprint: n.identity.ClaimedFingerprint(n.cfg.Protocol),
		Port:        int(n.port.Load()),
		Protocol:    n.cfg.Protocol,
	}
}

// Port is the port the receiver listens on.
func (n *Node) Port() int { return int(n.port.Load()) }

// Identity is the node's identity.
func (n *Node) Identity() *Identity { return n.identity }

// Client talks to peers.
func (n *Node) Client() *Client { return n.client }

// Announce sends an announcement burst over multicast.
func (n *Node) Announce(ctx context.Context) {
	if n.discovery != nil {
		n.discovery.AnnounceBurst(ctx)
	}
}

// Scan probes the local networks for devices that miss multicast.
func (n *Node) Scan(ctx context.Context) {
	if n.discovery != nil {
		n.discovery.Scan(ctx)
	}
}

// Probe registers with a known device, such as a favorite.
func (n *Node) Probe(ctx context.Context, ip string, port int, protocol Protocol) (Device, bool) {
	if n.discovery == nil {
		return Device{}, false
	}
	return n.discovery.Probe(ctx, ip, port, protocol)
}

// LocalAddresses lists this device's non-loopback IPv4 addresses.
func (n *Node) LocalAddresses() []string {
	var out []string
	seen := map[string]bool{}
	ifis, err := net.Interfaces()
	if err != nil {
		return out
	}
	for _, ifi := range ifis {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifi.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ip4 := ipnet.IP.To4(); ip4 != nil && !ip4.IsLinkLocalUnicast() && !seen[ip4.String()] {
					seen[ip4.String()] = true
					out = append(out, ip4.String())
				}
			}
		}
	}
	return out
}

// Stop ends discovery and the receiver.
func (n *Node) Stop(ctx context.Context) {
	if n.discovery != nil {
		n.discovery.Stop()
	}
	if n.server != nil {
		_ = n.server.Stop(ctx)
	}
	if n.client != nil {
		n.client.Close()
	}
}

// CancelReceive ends the receive session sessionID, as the app's cancel
// button does.
func (n *Node) CancelReceive(sessionID string) bool {
	if n.server == nil {
		return false
	}
	return n.server.CancelSession(sessionID)
}
