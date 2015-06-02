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

func GenerateRandomAccessKey() (accessKey int64) {
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
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

func VanishData(kadem Kademlia, VDOID ID, data []byte, numberKeys byte,
	threshold byte) (vdo VanashingDataObject) {
	var newVDO VanashingDataObject
	newVDO.NumberKeys = numberKeys
	newVDO.Threshold = threshold
	//newVDO.VDOID = NewRandomID()
	//Use the provided "GenerateRandomCryptoKey" function to create a random cryptographic key (K)
	random_K := GenerateRandomCryptoKey()

	//Use the provided "encrypt" function by giving it the random key(K) and text
	cyphertext := encrypt(random_K, data)

	var kadem_pointer *Kademlia
	kadem_pointer = &kadem

	newVDO.Ciphertext = cyphertext

	map_K, err := sss.Split(numberKeys, threshold, random_K)
	if err != nil {
		log.Fatal("Split", err)
		fmt.Println("ERROR: Split: ", err)
		return newVDO
	} else {

		//kademPointer := &kadem

		//Use provided GenerateRandomAccessKey to create an access key
		newVDO.AccessKey = GenerateRandomAccessKey()

		//Use "CalculateSharedKeyLocations" function to find the right []ID to send RPC
		keysLocation := CalculateSharedKeyLocations(newVDO.AccessKey, int64(numberKeys))
		for i := 0; i < len(keysLocation); i++ {
			k := byte(i + 1)
			v := map_K[k]
			all := []byte{k}
			for x := 0; x < len(v); x++ {
				all = append(all, v[x])
			}

			kadem_pointer.DoIterativeStore(keysLocation[i], all)

		}
	}

	newVDO.VDOID = VDOID
	return newVDO
}

func UnvanishData(kadem Kademlia, vdo VanashingDataObject) (data []byte) {

	var kadem_pointer *Kademlia
	kadem_pointer = &kadem

	map_value := make(map[byte][]byte)
	//Use vdo.AccessKey and CalculateSharedKeyLocations to search for at least vdo.Threshold keys in the DHT.
	keysLocation := CalculateSharedKeyLocations(vdo.AccessKey, int64(vdo.Threshold))

	if len(keysLocation) < int(vdo.Threshold) {
		fmt.Println("ERR: Could not obtain a sufficient number of shared keys")
		return
	}

	//Use sss.Combine to recreate the key, K
	for i := 0; i < len(keysLocation); i++ {
		value := kadem_pointer.DoIterativeFindValue_UsedInVanish(keysLocation[i])
		if len(value) == 0 {
			continue
		} else {
			k := value[0]
			v := value[1:]
			map_value[k] = []byte(v)

		}
	}

	//Use sss.Combine to recreate the key, K
	real_key := sss.Combine(map_value)
	//use decrypt to unencrypt vdo.Ciphertext.
	data = decrypt(real_key, vdo.Ciphertext)

	return
}
