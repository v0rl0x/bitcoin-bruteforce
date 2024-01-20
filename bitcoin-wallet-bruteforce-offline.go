//offline version, use any database u want to achieve this, I used http://alladdresses.loyce.club/

package main

import (
        "bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/ripemd160"
	"crypto/sha256"
)

func readAddresses(filePath string) (map[string]bool, error) {
    addresses := make(map[string]bool)

    file, err := os.Open(filePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        addresses[scanner.Text()] = true
    }

    if err := scanner.Err(); err != nil {
        return nil, err
    }

    return addresses, nil
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

func worker(id int, wg *sync.WaitGroup, mutex *sync.Mutex, outputFile string, btcAddresses map[string]bool) {
    defer wg.Done()

    for {
        privateKey, publicAddress, err := generateKeyAndAddress()
        if err != nil {
            log.Printf("Worker %d: Failed to generate key and address: %s", id, err)
            continue
        }

        if _, exists := btcAddresses[publicAddress]; exists {
            fmt.Printf("Match Found! Privatekey: %s Publicaddress: %s\n", privateKey, publicAddress)

            mutex.Lock()
            file, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
            if err != nil {
                log.Printf("Worker %d: Failed to open file: %s", id, err)
                mutex.Unlock()
                continue
            }

            if _, err := file.WriteString(fmt.Sprintf("%s:%s\n", privateKey, publicAddress)); err != nil {
                log.Printf("Worker %d: Failed to write to file: %s", id, err)
            }
            file.Close()
            mutex.Unlock()
        }
    }
}

func main() {
    if len(os.Args) != 4 {
        fmt.Println("Usage: ./golangscript <threads> <output-file.txt> <btc-address-file.txt>")
        os.Exit(1)
    }

    numThreads, err := strconv.Atoi(os.Args[1])
    if err != nil {
        log.Fatalf("Invalid number of threads: %s", err)
    }

    outputFile := os.Args[2]
    btcAddressesFile := os.Args[3]

    btcAddresses, err := readAddresses(btcAddressesFile)
    if err != nil {
        log.Fatalf("Failed to read BTC addresses: %s", err)
    }

    var wg sync.WaitGroup
    var mutex sync.Mutex

    for i := 0; i < numThreads; i++ {
        wg.Add(1)
        go worker(i, &wg, &mutex, outputFile, btcAddresses)
    }

    wg.Wait()
}
