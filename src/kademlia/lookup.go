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
	var closestNode Contact
	visited := make(map[ID]int)
	active := new(ConcurrMap)
	active.m = make(map[ID]int)
	nodeChan := make(chan Contact)

	for _, node := range tempShortlist {
		shortlist = append(shortlist, ContactDistance{node, node.NodeID.Xor(target).ToInt()})
	}

	fmt.Println("tempShortlist: ", tempShortlist)
	//assert shortlist is length of k-->20
	fmt.Println("shortlist length: ", len(shortlist))

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
			}
		}
	}()

	waitChan := make(chan int, ALPHA)

	for !terminated(shortlist, active, closestNode) {
		count := 0
		for count < ALPHA {
			for _, c := range shortlist {
				if visited[c.contact.NodeID] == 0 {
					if count >= ALPHA {
						break
					}
					go sendQuery(c.contact, active, waitChan, nodeChan)
					visited[c.contact.NodeID] = 1
					count++
				}
			}
		}

		fmt.Println("waiting")
		for i := 0; i < ALPHA; i++ {
			<-waitChan
		}
		fmt.Println("counter: ", count)
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

func terminated(shortlist []ContactDistance, active *ConcurrMap, closestnode Contact) bool {
	sort.Sort(ByDist(shortlist))
	if len(shortlist) < K {
		fmt.Println("shortlist length: ", len(shortlist))
	}

	if shortlist[0].contact.NodeID.Equals(closestnode.NodeID) {
		return true
	}
	closestnode = shortlist[0].contact

	for i := 0; i < len(shortlist) && i < K; i++ {
		active.RLock()
		a := active.m[shortlist[i].contact.NodeID]
		active.RUnlock()
		fmt.Println("a: ", a)
		if a == 0 {
			return false
		}
	}
	return true
}
