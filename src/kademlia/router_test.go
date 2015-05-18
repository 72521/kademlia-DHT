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

}

func TestFindClosest(t *testing.T) {
	Nodes := make([]ID, 0)
	for i := 0; i < 25; i++ {
		Nodes = append(Nodes, NewRandomID())
	}
	rt := NewRoutingTable(Contact{Nodes[0], net.ParseIP("127.0.0.1"), uint16(7890)})

	for i := 1; i < len(Nodes); i++ {
		rt.Update(&Contact{Nodes[i], net.ParseIP("127.0.0.1"), uint16(7890 + i)})
	}

	target := Nodes[1]

	result := rt.FindClosest(target, 20)

	t.Log("result length: ", len(result))
	if result[0].NodeID != target {
		t.Errorf("FindClosest: %s\nBut target: %s\nNot equal\n",
			result[0].NodeID, target)
	}
}

/*
func TestUpdate(t testing.T) {
	Nodes := make([]ID, 0)
	Nodes = append(Nodes, IDFromString("0000000000000000"))
	prefix := NewRandomID()[0]
	prefix = 255
	for i := 0; i < 20; i++ {
		nodeid := NewRandomID().AsString()
		nodeid = append("0", nodeid[1:])
		Nodes = append(Nodes, nodeid.IDFromString)
	}
	rt := NewRoutingTable(Contact{Nodes[0], net.ParseIP("127.0.0.1"), uint16(7890)})
	for i := 1; i < len(Nodes); i++ {
		rt.Update(&Contact{Nodes[i], net.ParseIP("127.0.0.1"), uint16(7890 + i)})
	}

	if len(rt.buckets[0]) != 20 {
		t.Errorf("rt.buckets[0] length: %d\n", len(rt.buckets[0]))
	}
}
*/
