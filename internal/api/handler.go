package api

import (
	"encoding/json"
	"net/http"

	"mychain/internal/chain"
	"mychain/internal/p2p"
)

type ValidResponse struct {
	Valid bool `json:"valid"`
}

type MineRequest struct {
	Miner string `json:"miner"`
}

type SyncRequest struct {
	Blocks     []*chain.Block `json:"blocks"`
	Difficulty int            `json:"difficulty"`
}

func handleBlocks(bc *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		blocks := bc.GetBlocks()
		json.NewEncoder(w).Encode(blocks)
	}
}

func handleMine(bc *chain.Blockchain, node *p2p.Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()
		var req MineRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		b, err := bc.Mine(req.Miner)
		if err != nil {
			http.Error(w, "mine failed", http.StatusBadRequest)
			return
		}
		go node.Broadcast()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(b)
	}
}

func handleTransaction(bc *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()
		var req chain.Tx
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		err := bc.SubmitTx(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
	}
}

func handleBalance(bc *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		addr := r.PathValue("address")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"balance": bc.BalanceWithPending(addr)})
	}
}

func handleValid(bc *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var v ValidResponse
		v.Valid = bc.IsValid()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(v)
	}
}

func handleSync(bc *chain.Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()
		var req SyncRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := bc.ReplaceChain(req.Blocks, req.Difficulty); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "rejected", "reason": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
	}
}

func NewMux(bc *chain.Blockchain, node *p2p.Node) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/blocks", handleBlocks(bc))
	mux.HandleFunc("/mine", handleMine(bc, node))
	mux.HandleFunc("/transaction", handleTransaction(bc))
	mux.HandleFunc("GET /balance/{address}", handleBalance(bc))
	mux.HandleFunc("/valid", handleValid(bc))
	mux.HandleFunc("/sync", handleSync(bc))

	return mux
}
