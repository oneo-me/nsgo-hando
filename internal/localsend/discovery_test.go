package localsend

import (
	"context"
	"testing"
	"time"
)

func TestDiscoveryBetweenNodes(t *testing.T) {
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
