package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"mychain/internal/chain"
)

const baseURL = "http://localhost:8080"

func doGet(path string) {
	resp, err := http.Get(baseURL + path)
	if err != nil {
		fmt.Println("request failed:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("server returned:", resp.StatusCode)
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

func cmdMine() {
	doPost("/mine", nil)
}

func cmdSend(from string, to string, amountstr string) {
	amount, err := strconv.Atoi(amountstr)
	if err != nil {
		fmt.Println("not number:", err)
		os.Exit(1)
	}
	tx := chain.Tx{From: from, To: to, Amount: amount}

	body, err := json.Marshal(tx)
	if err != nil {
		fmt.Println("wrong json:", err)
		os.Exit(1)
	}

	doPost("/transaction", body)
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
		cmdMine()
	case "send":
		if len(os.Args) < 5 {
			fmt.Println("usage: mychain send <from> <to> <amount>")
			os.Exit(1)
		}
		cmdSend(os.Args[2], os.Args[3], os.Args[4])
	default:
		fmt.Println("unknown command:", os.Args[1])
		os.Exit(1)
	}
}
