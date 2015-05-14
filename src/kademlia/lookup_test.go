package kademlia

import (
	"strconv"
	"testing"
)

func TestIterativeFindNode(t *testing.T) {
	instanceList := make([]*Kademlia, 0)

	for i := 0; i < 1000; i++ {
		instanceList = append(instanceList, NewKademlia("127.0.0.1:"+strconv.Itoa(8000+i)))
	}

	for i := 0; i < len(instanceList); i++ {
		for j := 0; j < len(instanceList); j++ {
			if i == j {
				continue
			}
			tmp_host, tmp_port, _ := StringToIpPort("127.0.0.1:" + strconv.Itoa(8000+j))
			instanceList[i].DoPing(tmp_host, tmp_port)
		}
	}

	//	instanceList[0].IterativeFindNode(NewRandomID())

	/*
		for i := 0; i < len(instanceList); i++ {
			for j := 0; j < len(instanceList); j++ {
				if i == j {
					continue
				}
				tmp_contact, err := instanceList[i].FindContact(instanceList[j].NodeID)
				if err != nil {
					t.Errorf("Contact[%d] cannot find contact[%d]\n", i, j)
					return
				}
				if tmp_contact.NodeID != instanceList[j].NodeID {
					t.Errorf("Found contact[%d] ID from contact[%d] is not correct\n", j, i)
					return
				}
			}

		}
	*/
}
