package main

import (
	"log"
	"net/http"

	"mychain/internal/api"
	"mychain/internal/chain"
)

func main() {
	bc, err := chain.LoadFromFile(chain.ChainFilePath)
	if err != nil {
		log.Fatal(err)
	}
	if bc == nil {
		log.Println("no chain file, creating genesis")
		bc = chain.NewBlockchain(5, chain.ChainFilePath)
		if err := chain.SaveToFile(bc); err != nil {
			log.Fatal(err)
		}
	}

	mux := api.NewMux(bc)

	log.Println("listening on: 8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
