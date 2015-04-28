package kademlia

// Contains definitions mirroring the Kademlia spec. You will need to stick
// strictly to these to be compatible with the reference implementation and
// other groups' code.

import (
	"net"
)

type KademliaCore struct {
	kademlia *Kademlia
}

// Host identification.
type Contact struct {
	NodeID ID
	Host   net.IP
	Port   uint16
}

///////////////////////////////////////////////////////////////////////////////
// PING
///////////////////////////////////////////////////////////////////////////////
type PingMessage struct {
	Sender Contact
	MsgID  ID
}

type PongMessage struct {
	MsgID  ID
	Sender Contact
}

func (kc *KademliaCore) Ping(ping PingMessage, pong *PongMessage) error {
	// TODO: Finish implementation
	pong.MsgID = CopyID(ping.MsgID)
	// Specify the sender
	pong.Sender = kc.kademlia.routes.SelfContact
	// Update contact, etc
	kc.kademlia.contactChan <- &ping.Sender
	return nil
}

///////////////////////////////////////////////////////////////////////////////
// STORE
///////////////////////////////////////////////////////////////////////////////
type StoreRequest struct {
	Sender Contact
	MsgID  ID
	Key    ID
	Value  []byte
}

type StoreResult struct {
	MsgID ID
	Err   error
}

func (kc *KademliaCore) Store(req StoreRequest, res *StoreResult) error {
	set := &KeySet{req.Key, req.Value, make(chan int)}
	res.MsgID = CopyID(req.MsgID)
	kc.kademlia.contactChan <- &req.Sender
	kc.kademlia.keyChan <- set
	return nil
}

///////////////////////////////////////////////////////////////////////////////
// FIND_NODE
///////////////////////////////////////////////////////////////////////////////
type FindNodeRequest struct {
	Sender Contact
	MsgID  ID
	NodeID ID
}

type FindNodeResult struct {
	MsgID ID
	Nodes []Contact
	Err   error
}

func (kc *KademliaCore) FindNode(req FindNodeRequest, res *FindNodeResult) error {
	contacts := kc.kademlia.routes.FindClosest(req.NodeID, K)
	res.MsgID = CopyID(req.MsgID)
	res.Nodes = make([]Contact, len(contacts))
	copy(res.Nodes, contacts)
	return nil
}

///////////////////////////////////////////////////////////////////////////////
// FIND_VALUE
///////////////////////////////////////////////////////////////////////////////
type FindValueRequest struct {
	Sender Contact
	MsgID  ID
	Key    ID
}

// If Value is nil, it should be ignored, and Nodes means the same as in a
// FindNodeResult.
type FindValueResult struct {
	MsgID ID
	Value []byte
	Nodes []Contact
	Err   error
}

func (kc *KademliaCore) FindValue(req FindValueRequest, res *FindValueResult) error {
	res.MsgID = CopyID(req.MsgID)
	keys, found := kc.kademlia.LocalFindValueHelper(req.Key)
	if found == 1 {
		copy(res.Value, keys.Value)
		return nil
	}

	res.Value = nil

	reqNode := FindNodeRequest{req.Sender, req.MsgID, kc.kademlia.routes.SelfContact.NodeID}
	var resNode FindNodeResult

	kc.FindNode(reqNode, &resNode)

	copy(res.Nodes, resNode.Nodes)

	return nil
}
