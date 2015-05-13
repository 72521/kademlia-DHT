package kademlia

// Contains the core kademlia type. In addition to core state, this type serves
// as a receiver for the RPC methods, which is required by that package.

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
)

const (
	ALPHA = 3
	b     = 8 * IDBytes
	K     = 20
)

// Kademlia type. You can put whatever state you need in this.
type Kademlia struct {
	NodeID      ID
	Routes      *RoutingTable
	contactChan chan *Contact
	keyChan     chan *KeySet
	searchChan  chan *KeySet
	hashtable   map[ID][]byte
}

type KeySet struct {
	Key        ID
	Value      []byte
	resultChan chan int
}

func NewKademlia(laddr string) *Kademlia {
	// TODO: Initialize other state here as you add functionality.
	k := new(Kademlia)
	k.NodeID = NewRandomID()
	k.contactChan = make(chan *Contact)
	k.keyChan = make(chan *KeySet)
	k.searchChan = make(chan *KeySet)
	k.hashtable = make(map[ID][]byte)

	// Set up RPC server
	// NOTE: KademliaCore is just a wrapper around Kademlia. This type includes
	// the RPC functions.
	rpc.Register(&KademliaCore{k})
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", laddr)
	if err != nil {
		log.Fatal("Listen: ", err)
	}
	// Run RPC server forever.
	go http.Serve(l, nil)

	// Add self contact
	hostname, port, _ := net.SplitHostPort(l.Addr().String())
	port_int, _ := strconv.Atoi(port)
	ipAddrStrings, err := net.LookupHost(hostname)
	var host net.IP
	for i := 0; i < len(ipAddrStrings); i++ {
		host = net.ParseIP(ipAddrStrings[i])
		if host.To4() != nil {
			break
		}
	}
	SelfContact := Contact{k.NodeID, host, uint16(port_int)}
	k.Routes = NewRoutingTable(SelfContact)

	go handleChan(k)

	return k
}

type NotFoundError struct {
	id  ID
	msg string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%x %s", e.id, e.msg)
}

func handleChan(k *Kademlia) {
	for {
		select {
		case contact := <-k.contactChan:
			fmt.Printf("add new contact: %s\n", contact.NodeID.AsString())
			k.Routes.Update(contact)

		case set := <-k.keyChan:
			k.hashtable[set.Key] = set.Value
		case set := <-k.searchChan:
			set.Value = k.hashtable[set.Key]
			if set.Value == nil {
				set.resultChan <- 0
			} else {
				set.resultChan <- 1
			}
		}
	}
}

func (k *Kademlia) FindContact(nodeId ID) (*Contact, error) {
	if nodeId == k.Routes.SelfContact.NodeID {
		return &k.Routes.SelfContact, nil
	}
	prefix_length := nodeId.Xor(k.Routes.SelfContact.NodeID).PrefixLen()
	bucket := k.Routes.buckets[prefix_length]
	fmt.Println(bucket)
	for _, value := range bucket {
		if value.NodeID.Equals(nodeId) {
			k.contactChan <- &value
			return &value, nil
		}
	}

	return nil, &NotFoundError{nodeId, "Not found"}
}

// This is the function to perform the RPC
func (k *Kademlia) DoPing(host net.IP, port uint16) string {
	ping := PingMessage{k.Routes.SelfContact, NewRandomID()}
	var pong PongMessage

	client, err := rpc.DialHTTP("tcp", Dest(host, port))
	if err != nil {
		log.Fatal("DialHTTP: ", err)
	}
	err = client.Call("KademliaCore.Ping", ping, &pong)
	if err != nil {
		log.Fatal("Call: ", err)
		return "ERR: " + err.Error()
	}

	return "OK: " + pong.MsgID.AsString()
}

func (k *Kademlia) DoStore(contact *Contact, key ID, value []byte) string {
	req := StoreRequest{k.Routes.SelfContact, NewRandomID(), key, value}
	var res StoreResult

	client, err := rpc.DialHTTP("tcp", Dest(contact.Host, contact.Port))
	if err != nil {
		log.Fatal("DialHTTP: ", err)
	}
	err = client.Call("KademliaCore.Store", req, &res)
	if err != nil {
		log.Fatal("Call: ", err)
		return "ERR: " + err.Error()
	}
	return "OK: " + res.MsgID.AsString()
}

func (k *Kademlia) DoFindNode(contact *Contact, searchKey ID) string {
	req := FindNodeRequest{k.Routes.SelfContact, NewRandomID(), searchKey}
	var res FindNodeResult

	client, err := rpc.DialHTTP("tcp", Dest(contact.Host, contact.Port))
	if err != nil {
		log.Fatal("DialHTTP: ", err)
	}
	err = client.Call("KademliaCore.FindNode", req, &res)
	if err != nil {
		log.Fatal("Call: ", err)
		return "ERR: " + err.Error()
	}
	return "OK: " + res.MsgID.AsString()
}

func (k *Kademlia) DoFindValue(contact *Contact, searchKey ID) string {
	req := FindValueRequest{*contact, NewRandomID(), searchKey}
	res := new(FindValueResult)

	client, err := rpc.DialHTTP("tcp", Dest(contact.Host, contact.Port))
	if err != nil {
		log.Fatal("DialHTTP: ", err)
	}
	err = client.Call("KademliaCore.FindValue", req, &res)
	if err != nil {
		log.Fatal("Call: ", err)
		return "ERR: " + err.Error()
	}
	return "OK: value --> " + string(res.Value)
}

func (k *Kademlia) LocalFindValue(searchKey ID) string {
	// TODO: Implement
	// If all goes well, return "OK: <output>", otherwise print "ERR: <messsage>"
	keys, found := k.LocalFindValueHelper(searchKey)
	if found == 1 {
		return "OK: value --> " + string(keys.Value)
	}
	return "Err: cannot find key"
}

func (k *Kademlia) LocalFindValueHelper(searchKey ID) (ret *KeySet, found int) {
	ret = new(KeySet)
	ret.Key = searchKey
	ret.resultChan = make(chan int)
	k.searchChan <- ret
	found = <-ret.resultChan

	return
}

func (k *Kademlia) DoIterativeFindNode(id ID) string {
	// For project 2!
	return "ERR: Not implemented"
}
func (k *Kademlia) DoIterativeStore(key ID, value []byte) string {
	// For project 2!
	return "ERR: Not implemented"
}
func (k *Kademlia) DoIterativeFindValue(key ID) string {
	// For project 2!
	return "ERR: Not implemented"
}

func Dest(host net.IP, port uint16) string {
	return host.String() + ":" + strconv.FormatInt(int64(port), 10)
}
