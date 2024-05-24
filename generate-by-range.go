package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/ripemd160"
)

const botToken = "your_bot_token_here"
const chatID = "your_chat_id_here"

type BlockCypherResponse struct {
	Balance int `json:"balance"`
}

func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("no non-loopback address found")
}

func sendMessage(botToken, chatID, message string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?chat_id=%s&text=%s", botToken, chatID, url.QueryEscape(message))
	resp, err := http.Get(apiURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func generateKeyInRange(start, end *big.Int) (*big.Int, error) {
	diff := new(big.Int).Sub(end, start)
	randInt, err := rand.Int(rand.Reader, diff)
	if err != nil {
		return nil, err
	}
	return new(big.Int).Add(randInt, start), nil
}

func generateKeyAndAddressInRange(start, end *big.Int) (string, string, error) {
	privateKeyInt, err := generateKeyInRange(start, end)
	if err != nil {
		return "", "", err
	}

	privateKey := new(ecdsa.PrivateKey)
	privateKey.PublicKey.Curve = elliptic.P256()
	privateKey.D = privateKeyInt
	privateKey.PublicKey.X, privateKey.PublicKey.Y = elliptic.P256().ScalarBaseMult(privateKey.D.Bytes())

	publicKey := privateKey.PublicKey
	address, err := publicKeyToAddress(publicKey)
	if err != nil {
		return "", "", err
	}

	return fmt.Sprintf("%064x", privateKey.D), address, nil
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

func worker(id int, wg *sync.WaitGroup, mutex *sync.Mutex, outputFile string, start, end *big.Int) {
	defer wg.Done()

	for {
		privateKey, publicAddress, err := generateKeyAndAddressInRange(start, end)
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

			message := fmt.Sprintf("Privatekey: %s Publicaddress: %s Balance: %d", privateKey, publicAddress, balance)
			if err := sendMessage(botToken, chatID, message); err != nil {
				log.Printf("Worker %d: Failed to send Telegram message: %s", id, err)
			}
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

	ip, err := getLocalIP()
	if err != nil {
		log.Fatalf("Failed to get local IP address: %s", err)
	}

	if err := sendMessage(botToken, chatID, fmt.Sprintf("Started BTC Finder on: %s", ip)); err != nil {
		log.Fatalf("Failed to send startup message: %s", err)
	}

	startRange := new(big.Int)
	startRange.SetString("20000000000000000", 16)
	endRange := new(big.Int)
	endRange.SetString("3ffffffffffffffff", 16)

	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go worker(i, &wg, &mutex, outputFile, startRange, endRange)
	}

	wg.Wait()
}
