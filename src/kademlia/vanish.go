package kademlia

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	mathrand "math/rand"
	"sss"
	"time"
)

type VanashingDataObject struct {
	AccessKey  int64
	Ciphertext []byte
	NumberKeys byte
	Threshold  byte
	//add VDOID to be used in getting VDO
	VDOID ID
}

func GenerateRandomCryptoKey() (ret []byte) {
	for i := 0; i < 32; i++ {
		ret = append(ret, uint8(mathrand.Intn(256)))
	}
	return
}

func GenerateRandomAccessKey(epoch int64) (accessKey int64) {
	r := mathrand.New(mathrand.NewSource(epoch))
	accessKey = r.Int63()
	return
}

func CalculateSharedKeyLocations(accessKey int64, count int64) (ids []ID) {
	r := mathrand.New(mathrand.NewSource(accessKey))
	ids = make([]ID, count)
	for i := int64(0); i < count; i++ {
		for j := 0; j < IDBytes; j++ {
			ids[i][j] = uint8(r.Intn(256))
		}
	}
	return
}

func encrypt(key []byte, text []byte) (ciphertext []byte) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	ciphertext = make([]byte, aes.BlockSize+len(text))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], text)
	return
}

func decrypt(key []byte, ciphertext []byte) (text []byte) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext is not long enough")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	return ciphertext
}

func CalculateEpochNumber() (number int64) {
	number = int64((time.Now().UTC().Year()-1970)*365*3 + (int(time.Now().UTC().Month())-1)*30*3 + (time.Now().UTC().Day()-1)*3 + time.Now().UTC().Hour()/8)
	return
}

func VanishData(kadem *Kademlia, VDOID ID, data []byte, numberKeys byte,
	threshold byte) (vdo VanashingDataObject) {
	vdo.NumberKeys = numberKeys
	vdo.Threshold = threshold
	//vdo.VDOID = NewRandomID()
	//Use the provided "GenerateRandomCryptoKey" function to create a random cryptographic key (K)
	random_K := GenerateRandomCryptoKey()

	//Use the provided "encrypt" function by giving it the random key(K) and text
	cyphertext := encrypt(random_K, data)

	vdo.Ciphertext = cyphertext

	map_K, err := sss.Split(numberKeys, threshold, random_K)
	if err != nil {
		log.Fatal("Split", err)
		fmt.Println("ERROR: Split: ", err)
		return
	} else {

		//Use provided GenerateRandomAccessKey to create an access key
		vdo.AccessKey = GenerateRandomAccessKey(CalculateEpochNumber())

		//Use "CalculateSharedKeyLocations" function to find the right []ID to send RPC
		keysLocation := CalculateSharedKeyLocations(vdo.AccessKey, int64(numberKeys))
		for i := 0; i < len(keysLocation); i++ {
			k := byte(i + 1)
			v := map_K[k]
			all := []byte{k}
			for x := 0; x < len(v); x++ {
				all = append(all, v[x])
			}

			kadem.DoIterativeStore(keysLocation[i], all)

		}
	}

	vdo.VDOID = VDOID

	return
}

func Refresh(kadem *Kademlia, vdoid ID, timeout int) {
	// the unit of timeout is second
	for {

		time.Sleep(time.Duration(timeout) * time.Second)
		kadem.VDOmap.Lock()

		temp_vdo := kadem.VDOmap.m[vdoid]

		//----------------------------------------
		//retrieve the key
		keysLocation := CalculateSharedKeyLocations(temp_vdo.AccessKey, int64(temp_vdo.NumberKeys))

		number_valid_location := 0
		map_value := make(map[byte][]byte)
		if len(keysLocation) < int(temp_vdo.Threshold) {
			fmt.Println("ERR: Could not obtain a sufficient number of shared keys")
			return
		}

		//Use sss.Combine to recreate the key, K
		for i := 0; i < len(keysLocation); i++ {
			value := kadem.DoIterativeFindValue_UsedInVanish(keysLocation[i])
			if len(value) != 0 {
				number_valid_location += 1
			}
			fmt.Println("Value is ", value)
			if len(value) == 0 {
				continue
			} else {
				all := []byte(value)
				k := all[0]
				v := all[1:]
				map_value[k] = v

				//fmt.Println(k, "'s value is ", v)

			}
		}

		if number_valid_location < int(temp_vdo.Threshold) {
			fmt.Println("ERR: Could not obtain a sufficient number of valid nodes")
			return
		}

		//Use sss.Combine to recreate the key, K
		real_key := sss.Combine(map_value)

		//---------------------------------------
		//resplit real_key

		map_K, err := sss.Split(temp_vdo.NumberKeys, temp_vdo.Threshold, real_key)
		if err != nil {
			return
		} else {

			//Use provided GenerateRandomAccessKey to create an access key
			temp_vdo.AccessKey = GenerateRandomAccessKey(CalculateEpochNumber())

			//Use "CalculateSharedKeyLocations" function to find the right []ID to send RPC
			keysLocation = CalculateSharedKeyLocations(temp_vdo.AccessKey, int64(temp_vdo.NumberKeys))

			//fmt.Println("InVanish: len of keyslocation", len(keysLocation))

			for i := 0; i < len(keysLocation); i++ {
				k := byte(i + 1)
				v := map_K[k]
				all := []byte{k}
				for x := 0; x < len(v); x++ {
					all = append(all, v[x])
				}

				kadem.DoIterativeStore(keysLocation[i], all)

			}
		}
		kadem.VDOmap.m[vdoid] = temp_vdo

		kadem.VDOmap.Unlock()
	}
}

func UnvanishData(kadem *Kademlia, vdo VanashingDataObject) (data []byte) {
	current_epoch_number := CalculateEpochNumber()

	for i := 0; i < 3; i++ {
		data = UnvanishData_acc(kadem, vdo, current_epoch_number)
		if len(data) == 0 {
			current_epoch_number -= 1
		} else {
			return
		}
	}

	return
}

func UnvanishData_acc(kadem *Kademlia, vdo VanashingDataObject, epoch int64) (data []byte) {

	map_value := make(map[byte][]byte)
	//used to see if the number of valid nodes is enough where we can find value in it.
	number_valid_location := 0
	//Use AccessKey and CalculateSharedKeyLocations to search for at least vdo.Threshold keys in the DHT.

	AccessKey := GenerateRandomAccessKey(epoch)

	keysLocation := CalculateSharedKeyLocations(AccessKey, int64(vdo.NumberKeys))

	if len(keysLocation) < int(vdo.Threshold) {
		fmt.Println("ERR: Could not obtain a sufficient number of shared keys")
		return
	}

	//Use sss.Combine to recreate the key, K
	for i := 0; i < len(keysLocation); i++ {
		value := kadem.DoIterativeFindValue_UsedInVanish(keysLocation[i])
		if len(value) != 0 {
			number_valid_location += 1
		}

		if len(value) == 0 {
			continue
		} else {
			all := []byte(value)
			k := all[0]
			v := all[1:]
			map_value[k] = v

		}
	}

	if number_valid_location < int(vdo.Threshold) {
		fmt.Println("ERR: Could not obtain a sufficient number of valid nodes")
		return
	}

	//Use sss.Combine to recreate the key, K
	real_key := sss.Combine(map_value)

	//use decrypt to unencrypt vdo.Ciphertext.
	data = decrypt(real_key, vdo.Ciphertext)

	return
}
