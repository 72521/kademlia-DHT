package kademlia

import (
	"fmt"
	"net/rpc"
	"sort"
	"strconv"
)

type RoutingTable struct {
	SelfContact Contact
	buckets     [][]Contact
}

func NewRoutingTable(node Contact) (ret *RoutingTable) {
	ret = new(RoutingTable)
	ret.buckets = make([][]Contact, IDBits)
	ret.SelfContact = node
	return
}

func (table *RoutingTable) Update(contact *Contact) {
	prefix_length := contact.NodeID.Xor(table.SelfContact.NodeID).PrefixLen()
	if prefix_length == 160 {
		fmt.Println("Error: Prefix_length: ", prefix_length)
		return
	}

	bucket := &table.buckets[prefix_length]
	var element Contact
	found := 0
	index := -1
	for x, value := range *bucket {
		if value.NodeID.Equals(contact.NodeID) {
			element = value
			index = x
			found = 1
			break
		}
	}
	if found == 0 {
		if len(*bucket) <= K {
			*bucket = append(*bucket, *contact)
		} else {
			pingToRemove(bucket, contact, &table.SelfContact)
		}

	} else {
		*bucket = append((*bucket)[:index], (*bucket)[index+1:]...)
		*bucket = append(*bucket, element)
	}
}

/***
 * if the least recently used contact has response, then
 * ignore the new element. If the least recently used contact
 * doesn't have response, then delete it and add the new contact.
 */
func pingToRemove(bucket *[]Contact, contact *Contact, self *Contact) {
	ping := PingMessage{*self, NewRandomID()}
	var pong PongMessage
	remove := 0

	port_str := strconv.Itoa(int((*bucket)[0].Port))
	client, err := rpc.DialHTTPPath("tcp", Dest((*bucket)[0].Host, (*bucket)[0].Port), rpc.DefaultRPCPath+port_str)
	if err != nil {
		remove = 1
	}
	defer client.Close()
	err = client.Call("KademliaCore.Ping", ping, &pong)
	if err != nil {
		remove = 1
	}

	if remove == 1 {
		*bucket = (*bucket)[1:]
		*bucket = append(*bucket, *contact)
	} else {
		tmp := (*bucket)[0]
		*bucket = (*bucket)[1:]
		*bucket = append(*bucket, tmp)
	}
}

type ContactDistance struct {
	contact Contact
	Dist    int
}

type ByDist []ContactDistance

func (d ByDist) Len() int           { return len(d) }
func (d ByDist) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }
func (d ByDist) Less(i, j int) bool { return d[i].Dist < d[j].Dist }

func calcDist(target ID, bucket []Contact, tempList *[]ContactDistance) {
	for _, value := range bucket {
		distID := value.NodeID.Xor(target)
		dist := distID.ToInt()
		cd := &ContactDistance{value, dist}
		*tempList = append(*tempList, *cd)
	}
}

func (table *RoutingTable) FindClosest(target ID, count int) (ret []Contact) {
	ret = make([]Contact, 0)
	tempList := make([]ContactDistance, 0)
	prefix_len := target.Xor(table.SelfContact.NodeID).PrefixLen()
	fmt.Println("prefix_len: ", prefix_len)
	for i := 0; (prefix_len-i >= 0 || prefix_len+i < IDBits) && len(tempList) < count; i++ {
		if prefix_len == IDBits && prefix_len-i == IDBits {
			tempList = append(tempList, ContactDistance{table.SelfContact, 0})
			continue
		}
		if prefix_len-i >= 0 {
			bucket := table.buckets[prefix_len-i]
			calcDist(target, bucket, &tempList)
		}
		if prefix_len+i < IDBits {
			bucket := table.buckets[prefix_len+i]
			calcDist(target, bucket, &tempList)
		}
	}

	sort.Sort(ByDist(tempList))
	if len(tempList) > count {
		tempList = tempList[:count]
	}

	for _, value := range tempList {
		ret = append(ret, value.contact)
	}

	return
}
