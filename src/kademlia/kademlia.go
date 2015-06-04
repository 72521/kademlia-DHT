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
	"sync"
)

const (
	ALPHA = 3
	b     = 8 * IDBytes
	K     = 20
)

// Kademlia type. You can put whatever state you need in this.
type Kademlia struct {
	NodeID           ID
	Routes           *RoutingTable
	contactChan      chan *Contact
	keyChan          chan *KeySet
	searchChan       chan *KeySet
	hashtable        map[ID][]byte
	bucketChan       chan int
	bucketResultChan chan []Contact
	VDOmap           VDOmap
}

type VDOmap struct {
	sync.RWMutex
	m map[ID]VanashingDataObject
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
	k.bucketChan = make(chan int)
	k.bucketResultChan = make(chan []Contact)
	k.VDOmap.m = make(map[ID]VanashingDataObject)

	// Set up RPC server
	// NOTE: KademliaCore is just a wrapper around Kademlia. This type includes
	// the RPC functions.
	server := rpc.NewServer()
	server.Register(&KademliaCore{k})
	_, port, _ := net.SplitHostPort(laddr)
	server.HandleHTTP(rpc.DefaultRPCPath+port, rpc.DefaultDebugPath+port)

	l, err := net.Listen("tcp", laddr)
	if err != nil {
		log.Fatal("Listen: ", err)
	}
	// Run RPC server forever.
	go http.Serve(l, nil)

	// Add self contact
	hostname, port, _ := net.SplitHostPort(l.Addr().String())
	port_int, _ := strconv.Atoi(port)
	ipAddrStrings, _ := net.LookupHost(hostname)
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
			k.Routes.Lock()
			k.Routes.Update(contact)
			k.Routes.Unlock()

		case prefix_length := <-k.bucketChan:
			k.bucketResultChan <- k.Routes.buckets[prefix_length]
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

func (k *Kademlia) ReadFromBuckets(prefix_length int) []Contact {
	k.bucketChan <- prefix_length
	ret := <-k.bucketResultChan
	return ret
}

func (k *Kademlia) FindContact(nodeId ID) (*Contact, error) {
	if nodeId == k.Routes.SelfContact.NodeID {
		return &k.Routes.SelfContact, nil
	}
	prefix_length := nodeId.Xor(k.Routes.SelfContact.NodeID).PrefixLen()
	bucket := k.ReadFromBuckets(prefix_length)
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

	port_str := strconv.Itoa(int(port))
	client, err := rpc.DialHTTPPath("tcp", Dest(host, port), rpc.DefaultRPCPath+port_str)
	if err != nil {
		log.Fatal("DialHTTP: ", err)
	}
	defer func() {
		client.Close()
	}()
	err = client.Call("KademliaCore.Ping", ping, &pong)
	if err != nil {
		log.Fatal("Call: ", err)
		return "ERR: " + err.Error()
	}
	k.contactChan <- &(&pong).Sender

	return "OK: " + pong.MsgID.AsString()
}

func (k *Kademlia) DoStore(contact *Contact, key ID, value []byte) string {
	req := StoreRequest{k.Routes.SelfContact, NewRandomID(), key, value}
	var res StoreResult

	port_str := strconv.Itoa(int(contact.Port))
	client, err := rpc.DialHTTPPath("tcp", Dest(contact.Host, contact.Port), rpc.DefaultRPCPath+port_str)
	if err != nil {
		log.Fatal("DialHTTP: ", err)
	}
	defer client.Close()
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

	port_str := strconv.Itoa(int(contact.Port))
	client, err := rpc.DialHTTPPath("tcp", Dest(contact.Host, contact.Port), rpc.DefaultRPCPath+port_str)
	if err != nil {
		log.Fatal("DialHTTP: ", err)
	}
	defer client.Close()
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

	port_str := strconv.Itoa(int(contact.Port))
	client, err := rpc.DialHTTPPath("tcp", Dest(contact.Host, contact.Port), rpc.DefaultRPCPath+port_str)
	if err != nil {
		log.Fatal("DialHTTP: ", err)
	}
	defer client.Close()
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
	ret := k.IterativeFindNode(id, false)
	if len(ret.contacts) > 0 {
		return "Success itertativefindnode"
	} else {
		return "Failed to iterativefindnode"
	}
}
func (k *Kademlia) DoIterativeStore(key ID, value []byte) string {
	// For project 2!
	ret := k.IterativeFindNode(key, false)
	for _, c := range ret.contacts {
		new_c := c
		go k.DoStore(&new_c, key, value)
	}

	return "Success store!"

}
func (k *Kademlia) DoIterativeFindValue(key ID) string {
	// For project 2!
	ret := k.IterativeFindNode(key, true)
	if ret.value != nil {
		str := "Key: " + ret.key.AsString() + " --> Value: " + string(ret.value)
		return str
	} else {
		return "Cannot find value"
	}
}

////////////////////////for project 3/////////////////////////////
func (k *Kademlia) DoIterativeFindValue_UsedInVanish(key ID) string {
	// For project 2!
	ret := k.IterativeFindNode(key, true)
	if ret.value != nil {
		str := string(ret.value)
		return str
	} else {
		return ""
	}
}

func (k *Kademlia) DoGetVDO(nodeid ID, vdoid ID) string {
	//find the right contact using FindClosest
	var right_contact Contact
	if nodeid == k.NodeID {
		right_contact = k.Routes.SelfContact
	} else {
		contacts := k.Routes.FindClosest(nodeid, 20)
		right_contact = contacts[0]
	}

	//using GetVDO to retrieve the right VDO
	req := GetVDORequest{right_contact, NewRandomID(), vdoid}
	res := new(GetVDOResult)

	port_str := strconv.Itoa(int(right_contact.Port))
	client, err := rpc.DialHTTPPath("tcp", Dest(right_contact.Host, right_contact.Port), rpc.DefaultRPCPath+port_str)
	if err != nil {
		log.Fatal("DialHTTP: ", err)
	}
	defer client.Close()
	err = client.Call("KademliaCore.GetVDO", req, &res)

	if err != nil {
		log.Fatal("Call: ", err)
		return "ERR: " + err.Error()
	}

	data := string(UnvanishData(*k, res.VDO))

	return data

}

////////////////////////for project 3/////////////////////////////

func Dest(host net.IP, port uint16) string {
	return host.String() + ":" + strconv.FormatInt(int64(port), 10)
}
