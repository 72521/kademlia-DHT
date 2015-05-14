package kademlia

import (
	"fmt"
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

func (k *Kademlia) IterativeFindNode(target ID) []Contact {
	tempShortlist := k.Routes.FindClosest(target, K)
	shortlist := make([]ContactDistance, 0)
	//	var closestNode Contact
	visited := make(map[ID]int)
	active := new(ConcurrMap)
	active.m = make(map[ID]int)
	nodeChan := make(chan Contact)

	//assert shortlist is length of k-->20
	fmt.Println("shortlist length: ", len(shortlist))

	for _, node := range tempShortlist {
		shortlist = append(shortlist, ContactDistance{node, node.NodeID.Xor(target).ToInt()})
	}

	go func() {
		for {
			select {
			case node := <-nodeChan:
				shortlist = append(shortlist, ContactDistance{node, node.NodeID.Xor(target).ToInt()})
			}
		}
	}()

	waitChan := make(chan int, ALPHA)

	for terminated(shortlist, active) {
		count := 0
		for count < ALPHA {
			for _, c := range shortlist {
				if visited[c.contact.NodeID] == 0 {
					go sendQuery(c.contact, active, waitChan, nodeChan)
					visited[c.contact.NodeID] = 1
					count++
				}
			}
		}

		for i := 0; i < ALPHA; i++ {
			<-waitChan
		}
	}

	ret := make([]Contact, 0)
	sort.Sort(ByDist(shortlist))
	if len(shortlist) > K {
		shortlist = shortlist[:K]
	}
	for _, value := range shortlist {
		ret = append(ret, value.contact)
	}

	return ret
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

func terminated(shortlist []ContactDistance, active *ConcurrMap) bool {
	sort.Sort(ByDist(shortlist))
	if len(shortlist) < K {
		fmt.Println("shortlist length: ", len(shortlist))
	}

	for i := 0; i < len(shortlist); i++ {
		active.RLock()
		a := active.m[shortlist[i].contact.NodeID]
		active.RUnlock()
		if a == 0 {
			return false
		}
	}
	return true
}
