# bitcoin-bruteforce
A Go program designed to create private keys, derive corresponding public keys from the private keys, and then check that the generated wallet addresses have funds. This is the most recent up to date FREE bruteforcer.

# how to use

go build bitcoin-wallet-bruteforce.go

(To use on all systems to ensure proper GCLIB: go build -ldflags '-extldflags "-static"' -o bitcoin-wallet-bruteforce

./bitcoin-wallet-bruteforce threads out-file.txt

Example: ./bitcoin-wallet-bruteforce 1000 wallets.txt

Offline Version: ./bitcoin-wallet-bruteforce threads out-file.txt btc-data-file.txt

Example: ./bitcoin-wallet-bruteforce 1000000 out.txt btc_aa.txt

# Information

All bitcoin addresses with funds in them will be recorded to the out-file.txt you choose. You can also rename this to anything you want. I advise you to run this in a screen and leave it for running for days on end. This is an efficient method of trying to obtain free funds.

Make sure Golang 1.2.1 is installed or latest version.

![LMAO](https://github.com/v0rl0x/bitcoin-bruteforce/assets/148959415/9f5cc5e5-0161-4554-ba45-f17a85324543)

Bitcoin bech32 addresses are generated with the bech32 version of the script.

The scripts come with the option to use telegram bots to save any bitcoin wallets automatically. If you do not whish to use this feature then put 123 as both values for the chat id and bot token.
