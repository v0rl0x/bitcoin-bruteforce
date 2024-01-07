package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
  "time"
  "encoding/json"

	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/ripemd160"
	"crypto/sha256"
)

type BlockCypherResponse struct {
    Balance int `json:"balance"`
}

func generateKeyAndAddress() (string, string, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}

	publicKey := privateKey.PublicKey
	address, err := publicKeyToAddress(publicKey)
	if err != nil {
		return "", "", err
	}

	return hex.EncodeToString(privateKey.D.Bytes()), address, nil
}

func publicKeyToAddress(publicKey ecdsa.PublicKey) (string, error) {
	pubKeyBytes := append(publicKey.X.Bytes(), publicKey.Y.Bytes()...)

	sha256Hash := sha256.New()
	sha256Hash.Write(pubKeyBytes)
	sha256Result := sha256Hash.Sum(nil)

	ripemd160Hash := ripemd160.New()
	ripemd160Hash.Write(sha256Result)
	ripemd160Result := ripemd160Hash.Sum(nil)

	networkVersion := byte(0x00)
	addressBytes := append([]byte{networkVersion}, ripemd160Result...)
	checksum := sha256Checksum(addressBytes)
	fullAddress := append(addressBytes, checksum...)

	return base58.Encode(fullAddress), nil
}

func sha256Checksum(input []byte) []byte {
	firstSHA := sha256.New()
	firstSHA.Write(input)
	result := firstSHA.Sum(nil)

	secondSHA := sha256.New()
	secondSHA.Write(result)
	finalResult := secondSHA.Sum(nil)

	return finalResult[:4]
}

func checkBalance(address string) (int, error) {
    time.Sleep(3 * time.Second)
    url := fmt.Sprintf("https://api.blockcypher.com/v1/btc/main/addrs/%s/balance", address)
    resp, err := http.Get(url)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()

    var response BlockCypherResponse
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return 0, err
    }

    return response.Balance, nil
}

func worker(id int, wg *sync.WaitGroup, mutex *sync.Mutex, outputFile string) {
	defer wg.Done()

	for {
		privateKey, publicAddress, err := generateKeyAndAddress()
		if err != nil {
			log.Printf("Worker %d: Failed to generate key and address: %s", id, err)
			continue
		}

		balance, err := checkBalance(publicAddress)
		if err != nil {
			log.Printf("Worker %d: Failed to check balance for %s: %s", id, publicAddress, err)
			continue
		}

		fmt.Printf("Privatekey: %s Publicaddress: %s Balance: %d\n", privateKey, publicAddress, balance)

		if balance > 0 {
			mutex.Lock()
			file, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Printf("Worker %d: Failed to open file: %s", id, err)
				mutex.Unlock()
				continue
			}

			if _, err := file.WriteString(fmt.Sprintf("%s:%s:%d\n", privateKey, publicAddress, balance)); err != nil {
				log.Printf("Worker %d: Failed to write to file: %s", id, err)
			}
			file.Close()
			mutex.Unlock()
		}
	}
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: ./golangscript <threads> <output-file.txt>")
		os.Exit(1)
	}

	numThreads, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalf("Invalid number of threads: %s", err)
	}

	outputFile := os.Args[2]
	var wg sync.WaitGroup
	var mutex sync.Mutex

	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go worker(i, &wg, &mutex, outputFile)
	}

	wg.Wait()
}