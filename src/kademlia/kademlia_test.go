package kademlia

import (
	"math"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

func StringToIpPort(laddr string) (ip net.IP, port uint16, err error) {
	hostString, portString, err := net.SplitHostPort(laddr)
	if err != nil {
		return
	}
	ipStr, err := net.LookupHost(hostString)
	if err != nil {
		return
	}
	for i := 0; i < len(ipStr); i++ {
		ip = net.ParseIP(ipStr[i])
		if ip.To4() != nil {
			break
		}
	}
	portInt, err := strconv.Atoi(portString)
	port = uint16(portInt)
	return
}

func TestPing(t *testing.T) {
	instance1 := NewKademlia("localhost:7890")
	instance2 := NewKademlia("localhost:7891")
	host2, port2, _ := StringToIpPort("localhost:7891")
	//	host1, port1, _ := StringToIpPort("localhost:7890")
	instance1.DoPing(host2, port2)
	//	instance2.DoPing(host1, port1)
	time.Sleep(1 * time.Second)
	contact2, err := instance1.FindContact(instance2.NodeID)
	if err != nil {
		t.Error("Instance 2's contact not found in Instance 1's contact list")
		t.Error("instance2: NodeID: ", instance2.NodeID.AsString())
		t.Error("instance1: NodeID: ", instance1.NodeID.AsString())
		t.Error("instance1: routers: ", instance1.Routes)
		t.Error("instance2: routers: ", instance2.Routes)
		t.Error("Error: ", err)
		return
	}
	contact1, err := instance2.FindContact(instance1.NodeID)
	if err != nil {
		t.Error("Instance 1's contact not found in Instance 2's contact list")
		return
	}
	if contact1.NodeID != instance1.NodeID {
		t.Error("Instance 1 ID incorrectly stored in Instance 2's contact list")
	}
	if contact2.NodeID != instance2.NodeID {
		t.Error("Instance 2 ID incorrectly stored in Instance 1's contact list")
	}
	return
}

func TestIterative(t *testing.T) {
	instanceList := make([]*Kademlia, 0)

	for i := 0; i < 200; i++ {
		instanceList = append(instanceList, NewKademlia("127.0.0.1:"+strconv.Itoa(10000+i)))
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
			tmp_host, tmp_port, _ := StringToIpPort("127.0.0.1:" + strconv.Itoa(10000+j))
			go instanceList[i].DoPing(tmp_host, tmp_port)
			counter++
		}
	}

	key := NewRandomID()
	value := "answer"
	instanceList[0].DoIterativeStore(key, []byte(value))
	result := instanceList[0].DoIterativeFindValue(key)
	if !strings.Contains(result, value) {
		t.Error("Expected value: ", value)
		t.Error("Return value: ", result)
	}
}
