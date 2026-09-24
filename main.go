package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"mychain/internal/api"
	"mychain/internal/chain"
)

func parsePeers(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	peers := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			peers = append(peers, p)
		}
	}
	return peers
}

func main() {
	port := flag.String("port", "8080", "port on listen on")
	peersFlag := flag.String("peers", "", "comma-separated peer address")
	flag.Parse()

	peers := parsePeers(*peersFlag)
	chainFilePath := fmt.Sprintf("mychain-%s.json", *port)

	bc, err := chain.LoadFromFile(chainFilePath)
	if err != nil {
		log.Fatal(err)
	}
	if bc == nil {
		log.Println("no chain file, creating genesis")
		bc = chain.NewBlockchain(chain.Difficulty, chainFilePath)
		if err := chain.SaveToFile(bc); err != nil {
			log.Fatal(err)
		}
	}

	mux := api.NewMux(bc)

	log.Printf("listening on: %s, peers: %v", *port, peers)
	log.Fatal(http.ListenAndServe(":"+*port, mux))
}
