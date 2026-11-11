package localsend

import (
	"context"
	"net"
	"testing"
	"time"

	"golang.org/x/net/ipv4"
)

// skipWithoutMulticast skips a test where the machine cannot deliver a
// multicast datagram to a socket of its own, as GitHub's macOS runners
// cannot. LocalSend itself falls back to scanning such networks.
func skipWithoutMulticast(t *testing.T) {
	t.Helper()
	group := net.ParseIP(DefaultMulticastGroup).To4()
	ifis := multicastInterfaces()
	if len(ifis) == 0 {
		t.Skip("no multicast interface")
	}
	recv, err := listenMulticast(DefaultPort)
	if err != nil {
		recv, err = listenMulticast(0)
	}
	if err != nil {
		t.Skipf("no multicast socket: %v", err)
	}
	defer recv.Close()
	port := recv.LocalAddr().(*net.UDPAddr).Port
	rp := ipv4.NewPacketConn(recv)
	joined := 0
	for i := range ifis {
		if rp.JoinGroup(&ifis[i], &net.UDPAddr{IP: group}) == nil {
			joined++
		}
	}
	if joined == 0 {
		t.Skip("cannot join the multicast group")
	}
	send, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		t.Skipf("no send socket: %v", err)
	}
	defer send.Close()
	sp := ipv4.NewPacketConn(send)
	_ = sp.SetMulticastTTL(1)
	sent := 0
	for i := range ifis {
		if sp.SetMulticastInterface(&ifis[i]) != nil {
			continue
		}
		if _, err := sp.WriteTo([]byte("probe"), nil, &net.UDPAddr{IP: group, Port: port}); err == nil {
			sent++
		}
	}
	if sent == 0 {
		t.Skip("cannot send to the multicast group")
	}
	_ = recv.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := recv.ReadFrom(make([]byte, 64)); err != nil {
		t.Skip("multicast does not reach this machine's own sockets")
	}
}

func TestDiscoveryBetweenNodes(t *testing.T) {
	skipWithoutMulticast(t)
	idA, _ := NewIdentity()
	idB, _ := NewIdentity()
	seenA := make(chan Device, 8)
	seenB := make(chan Device, 8)
	a, err := StartNode(NodeConfig{Identity: idA, Addr: "0.0.0.0:0", Alias: func() string { return "Node A" }, Protocol: ProtocolHTTP,
		OnDevice: func(d Device) { seenA <- d }})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Stop(context.Background())
	b, err := StartNode(NodeConfig{Identity: idB, Addr: "0.0.0.0:0", Alias: func() string { return "Node B" }, Protocol: ProtocolHTTP,
		OnDevice: func(d Device) { seenB <- d }})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Stop(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	go a.Announce(ctx)
	go b.Announce(ctx)
	select {
	case d := <-seenA:
		t.Logf("A saw %s %s", d.Alias, d.Fingerprint[:8])
	case <-time.After(10 * time.Second):
		t.Fatal("A did not see B")
	}
	select {
	case d := <-seenB:
		t.Logf("B saw %s %s", d.Alias, d.Fingerprint[:8])
	case <-time.After(5 * time.Second):
		t.Fatal("B did not see A")
	}
}
