package kademlia

import (
	"math"
	//"math/rand"
	"strconv"
	"testing"
	"time"
)

func TestIterativeFindNode(t *testing.T) {
	//r := rand.New(rand.NewSource(time.Now().Unix()))
	instanceList := make([]*Kademlia, 0)

	for i := 0; i < 200; i++ {
		instanceList = append(instanceList, NewKademlia("127.0.0.1:"+strconv.Itoa(8000+i)))
	}

	counter := 0
	for i := 0; i < len(instanceList); i++ {
		for j := 0; j < i; j++ {
			if i == j || math.Abs(float64(i-j)) > 10 {
				continue
			}
			if counter == 2 {
				counter = 0
				time.Sleep(10 * time.Millisecond)
			}
			tmp_host, tmp_port, _ := StringToIpPort("127.0.0.1:" + strconv.Itoa(8000+j))
			go instanceList[i].DoPing(tmp_host, tmp_port)
			counter++
		}
	}

	target0 := NewRandomID()
	target1 := instanceList[100].NodeID
	//tmp_host, tmp_port, _ := StringToIpPort("127.0.0.1:" + strconv.Itoa(8000+3))
	//instanceList[0].DoPing(tmp_host, tmp_port)

	result0 := instanceList[0].IterativeFindNode(target0)
	result1 := instanceList[0].IterativeFindNode(target1)

	found0 := false
	for _, value := range result0 {
		if value.NodeID == target0 {
			found0 = true
		}
	}
	found1 := false
	for _, value := range result1 {
		if value.NodeID == target1 {
			found1 = true
		}
	}
	if found1 == false {
		t.Error("Cannot find target")
	}
	t.Log("target0: ", target0)
	t.Log("result0: ", result0)
	t.Log("found0: ", found0)
	t.Log("target1: ", target1)
	t.Log("result1: ", result1)
	t.Log("found1: ", found1)

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
