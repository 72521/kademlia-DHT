package kademlia

import (
	"log"
	"net/rpc"
	"sort"
	"strconv"
	"sync"
)

type ConcurrMap struct {
	sync.RWMutex
	m map[ID]int
}

type IterativeResult struct {
	contacts []Contact
	key      ID
	value    []byte
}

func (k *Kademlia) IterativeFindNode(target ID, findvalue bool) (ret *IterativeResult) {
	tempShortlist := k.Routes.FindClosest(target, K)
	shortlist := make([]ContactDistance, 0)
	var closestNode Contact
	visited := make(map[ID]int)
	active := new(ConcurrMap)
	active.m = make(map[ID]int)
	nodeChan := make(chan Contact)
	keyChan := make(chan []byte)
	ret = new(IterativeResult)
	if findvalue == true {
		ret.key = target
	}
	ret.value = nil

	for _, node := range tempShortlist {
		shortlist = append(shortlist, ContactDistance{node, node.NodeID.Xor(target).ToInt()})
	}

	go func() {
		for {
			select {
			case node := <-nodeChan:
				found := 0
				for _, value := range shortlist {
					if value.contact.NodeID == node.NodeID {
						found = 1
						break
					}
				}
				if found == 0 {
					shortlist = append(shortlist, ContactDistance{node, node.NodeID.Xor(target).ToInt()})
				}
				sort.Sort(ByDist(shortlist))
			case value := <-keyChan:
				ret.value = value
				fakeNode := new(Contact)
				fakeNode.NodeID = ret.key
				shortlist = append(shortlist, ContactDistance{*fakeNode, (*fakeNode).NodeID.Xor(target).ToInt()})

			}
		}
	}()

	waitChan := make(chan int, ALPHA)

	for !terminated(shortlist, active, closestNode, ret.value) {
		//		startTime := time.Now()
		count := 0
		//TODO: fix infinity loop
		for _, c := range shortlist {

			if visited[c.contact.NodeID] == 0 {
				if count >= ALPHA {
					break
				}
				if findvalue == false {
					go sendQuery(c.contact, active, waitChan, nodeChan)
				} else {
					go sendFindValueQuery(c.contact, active, waitChan, nodeChan, keyChan, target)
				}
				visited[c.contact.NodeID] = 1
				count++
			}
		}

		for ; count > 0; count-- {
			<-waitChan
		}
	}

	ret.contacts = make([]Contact, 0)
	sort.Sort(ByDist(shortlist))
	if len(shortlist) > K {
		shortlist = shortlist[:K]
	}
	for _, value := range shortlist {
		ret.contacts = append(ret.contacts, value.contact)
	}

	return
}

func sendFindValueQuery(c Contact, active *ConcurrMap, waitChan chan int, nodeChan chan Contact, keyChan chan []byte, target ID) {
	args := FindValueRequest{c, NewRandomID(), target}
	var reply FindValueResult
	active.Lock()
	active.m[c.NodeID] = 1
	active.Unlock()

	port_str := strconv.Itoa(int(c.Port))
	client, err := rpc.DialHTTPPath("tcp", Dest(c.Host, c.Port), rpc.DefaultRPCPath+port_str)
	if err != nil {
		log.Fatal("DialHTTP", err)
		active.Lock()
		active.m[c.NodeID] = 0
		active.Unlock()
	}
	defer client.Close()
	err = client.Call("KademliaCore.FindValue", args, &reply)
	if err != nil {
		log.Fatal("Call: ", err)
		active.Lock()
		active.m[c.NodeID] = 0
		active.Unlock()
	}

	active.RLock()
	a := active.m[c.NodeID]
	active.RUnlock()
	if reply.Value == nil {
		if a == 1 {
			for _, node := range reply.Nodes {
				nodeChan <- node
			}
		}
	} else {
		keyChan <- reply.Value
	}
	waitChan <- 1
}

func sendQuery(c Contact, active *ConcurrMap, waitChan chan int, nodeChan chan Contact) {
	args := FindNodeRequest{c, NewRandomID(), c.NodeID}
	var reply FindNodeResult
	active.Lock()
	active.m[c.NodeID] = 1
	active.Unlock()

	port_str := strconv.Itoa(int(c.Port))
	client, err := rpc.DialHTTPPath("tcp", Dest(c.Host, c.Port), rpc.DefaultRPCPath+port_str)
	if err != nil {
		log.Fatal("DialHTTP", err)
		active.Lock()
		active.m[c.NodeID] = 0
		active.Unlock()
	}
	defer client.Close()
	err = client.Call("KademliaCore.FindNode", args, &reply)
	if err != nil {
		log.Fatal("Call: ", err)
		active.Lock()
		active.m[c.NodeID] = 0
		active.Unlock()
	}

	active.RLock()
	a := active.m[c.NodeID]
	active.RUnlock()
	if a == 1 {
		for _, node := range reply.Nodes {
			nodeChan <- node
		}
	}
	waitChan <- 1
}

func terminated(shortlist []ContactDistance, active *ConcurrMap, closestnode Contact, found_value []byte) bool {
	if found_value != nil {
		return true
	}
	if len(shortlist) < K {
	}

	if shortlist[0].contact.NodeID.Equals(closestnode.NodeID) {
		return true
	}
	closestnode = shortlist[0].contact

	for i := 0; i < len(shortlist) && i < K; i++ {
		active.RLock()
		a := active.m[shortlist[i].contact.NodeID]
		active.RUnlock()
		if a == 0 {
			return false
		}
	}
	return true
}
