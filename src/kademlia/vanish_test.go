package kademlia

import (
	"math"
	"strconv"
	"testing"
	"time"
)

func TestGenerateRandomCryptoKey(t *testing.T) {
	r := GenerateRandomCryptoKey()
	if len(r) != 32 {
		t.Error("the size of randomCryptoKey is not 32")
	}
}

func TestCalculateSharedKeyLocations(t *testing.T) {
}

func TestVanishData(t *testing.T) {
	instanceList := make([]*Kademlia, 0)
	for i := 0; i < 50; i++ {
		instanceList = append(instanceList, NewKademlia("127.0.0.1:"+strconv.Itoa(11000+i)))
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
			tmp_host, tmp_port, _ := StringToIpPort("127.0.0.1:" + strconv.Itoa(11000+j))
			go instanceList[i].DoPing(tmp_host, tmp_port)
			counter++
		}
	}

	VDOID := NewRandomID()
	n := byte(30)
	k := byte(2)
	data := []byte("Hello World")

	vdo_result := VanishData(*instanceList[0], VDOID, data, n, k)

	if vdo_result.NumberKeys != n || vdo_result.VDOID != VDOID || vdo_result.Threshold != k {
		t.Error(vdo_result)
	}

	unvanish_result := UnvanishData(*instanceList[0], vdo_result)

	t.Log("unvanish_result:", string(unvanish_result))

	if string(unvanish_result) != string(data) {
		t.Error("unvanish_result: ", unvanish_result)
	}

}
