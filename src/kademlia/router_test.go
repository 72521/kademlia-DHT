package kademlia

import (
	"net"
	"testing"
)

func TestRoutingTable(t *testing.T) {
	Nodes := make([]ID, 0)

	for i := 0; i < 20; i++ {
		Nodes = append(Nodes, NewRandomID())
	}
	rt := NewRoutingTable(Contact{Nodes[0], net.ParseIP("127.0.0.1"), uint16(7890)})

	for i := 1; i < len(Nodes); i++ {
		rt.Update(&Contact{Nodes[i], net.ParseIP("127.0.0.1"), uint16(7890 + i)})
	}

	//	t.Log(rt)
	for i := 1; i < 160; i++ {
		if len(rt.buckets[i]) > 20 {
			t.Errorf("buckets[%d] size: %d\n", i, len(rt.buckets[i]))
		}
	}

	//	for i := 0; i < len(Nodes); i++ {
	//		t.Log(Nodes[i].AsString())
	//	}
}
