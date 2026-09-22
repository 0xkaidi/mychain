package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"mychain/internal/api"
	"mychain/internal/chain"
)

const (
	baseURL        = "http://localhost:8080"
	defaultKeyPath = "./key.pem"
)

func doGet(path string) {
	resp, err := http.Get(baseURL + path)
	if err != nil {
		fmt.Println("request failed:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("server returned:", resp.StatusCode, string(body))
		os.Exit(1)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("read failed:", err)
		os.Exit(1)
	}
	fmt.Println(string(body))
}

func doPost(path string, body []byte) {
	resp, err := http.Post(baseURL+path, "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Println("request failed:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("server returned:", resp.StatusCode, string(body))
		os.Exit(1)
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("read failed:", err)
		os.Exit(1)
	}
	fmt.Println(string(body))
}

func cmdBlocks() {
	doGet("/blocks")
}

func cmdValid() {
	doGet("/valid")
}

func cmdBalance(addr string) {
	doGet("/balance/" + addr)
}

func cmdMine(args []string) {
	fs := flag.NewFlagSet("mine", flag.ExitOnError)
	keyPath := fs.String("key", defaultKeyPath, "path to private key")
	fs.Parse(args)

	data, err := os.ReadFile(*keyPath)
	if err != nil {
		fmt.Println("read failed:", err)
		os.Exit(1)
	}

	w, err := chain.WalletFromPem(data)
	if err != nil {
		fmt.Println("parse pem failed:", err)
		os.Exit(1)
	}

	addr := w.Address()

	miner := api.MineRequest{Miner: addr}

	body, err := json.Marshal(miner)
	if err != nil {
		fmt.Println("invalid json:", err)
		os.Exit(1)
	}

	doPost("/mine", body)
}

func cmdSend(args []string) {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	keyPath := fs.String("key", defaultKeyPath, "path to private key")
	fs.Parse(args)
	rest := fs.Args()

	if len(rest) < 2 {
		fmt.Println("usage: mychain send [-key key.pem] <to> <amount>")
		os.Exit(1)
	}

	to := rest[0]
	amount, err := strconv.Atoi(rest[1])
	if err != nil {
		fmt.Println("not number:", err)
		os.Exit(1)
	}

	data, err := os.ReadFile(*keyPath)
	if err != nil {
		fmt.Println("read failed:", err)
		os.Exit(1)
	}

	w, err := chain.WalletFromPem(data)
	if err != nil {
		fmt.Println("parse pem failed:", err)
		os.Exit(1)
	}

	tx := chain.Tx{From: w.Address(), To: to, Amount: amount, PubKey: w.PubKeyHex()}

	tx.Signature, err = chain.Sign(w.PrivKey, tx.SigningBytes())
	if err != nil {
		fmt.Println("sign failed:", err)
		os.Exit(1)
	}

	body, err := json.Marshal(tx)
	if err != nil {
		fmt.Println("wrong json:", err)
		os.Exit(1)
	}

	doPost("/transaction", body)
}

func cmdKeygen(args []string) {
	fs := flag.NewFlagSet("keygen", flag.ExitOnError)
	out := fs.String("out", defaultKeyPath, "path to save private key")
	fs.Parse(args)

	w, err := chain.NewWallet()
	if err != nil {
		fmt.Println("keygen failed:", err)
		os.Exit(1)
	}
	block, err := w.ToPem()
	if err != nil {
		fmt.Println("cant pem:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, block, 0o600); err != nil {
		fmt.Println("write key:", err)
		os.Exit(1)
	}
	fmt.Println("address:", w.Address())
	fmt.Println("save to:", *out)
}

func cmdInspect(args []string) {
	fs := flag.NewFlagSet("inspect", flag.ExitOnError)
	keypath := fs.String("key", defaultKeyPath, "path to save private key")
	fs.Parse(args)

	data, err := os.ReadFile(*keypath)
	if err != nil {
		fmt.Println("invalid file:", err)
		os.Exit(1)
	}

	w, err := chain.WalletFromPem(data)
	if err != nil {
		fmt.Println("invalid pem:", err)
		os.Exit(1)
	}
	fmt.Println("address:", w.Address())
	fmt.Println("pubkey:", w.PubKeyHex())
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: mychain <command>")
		fmt.Println("commands: blocks")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "blocks":
		cmdBlocks()
	case "valid":
		cmdValid()
	case "balance":
		if len(os.Args) < 3 {
			fmt.Println("usage: mychain balance <address>")
			os.Exit(1)
		}
		cmdBalance(os.Args[2])
	case "mine":
		cmdMine(os.Args[2:])
	case "send":
		if len(os.Args) < 4 {
			fmt.Println("usage: mychain send [from] <to> <amount>")
			os.Exit(1)
		}
		cmdSend(os.Args[2:])
	case "keygen":
		cmdKeygen(os.Args[2:])
	case "inspect":
		cmdInspect(os.Args[2:])
	default:
		fmt.Println("unknown command:", os.Args[1])
		os.Exit(1)
	}
}
