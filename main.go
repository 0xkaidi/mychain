package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Block struct {
	PrevBlockHash string   `json:"prev_block_hash"`
	Timestamp     int64    `json:"timestamp"`
	Data          []string `json:"data"`
	Nonce         int      `json:"nonce"`
	Hash          string   `json:"hash"`
}

const (
	genesisPrevHash = "0000000000000000000000000000000000000000000000000000000000000000"
	chainFilePath   = "./mychain.json"
)

type Blockchain struct {
	Blocks     []*Block
	Difficulty int
	path       string
	mu         sync.Mutex
}

type ChainFile struct {
	Blocks     []*Block `json:"blocks"`
	Difficulty int      `json:"difficulty"`
}

func (b *Block) CalculateHash() string {
	record := fmt.Sprintf("%s|%d|%s|%d", b.PrevBlockHash, b.Timestamp, strings.Join(b.Data, ","), b.Nonce)
	hashBytes := sha256.Sum256([]byte(record))
	return hex.EncodeToString(hashBytes[:])
}

func NewBlock(data []string, prevHash string, Difficulty int) *Block {
	nb := Block{PrevBlockHash: prevHash, Timestamp: time.Now().Unix(), Data: data, Nonce: 0, Hash: ""}
	nb.Mine(Difficulty)
	return &nb
}

func NewBlockchain(difficulty int, path string) *Blockchain {
	fb := NewBlock([]string{"Genesis Block"}, genesisPrevHash, difficulty)
	bc := &Blockchain{Blocks: []*Block{fb}, Difficulty: difficulty, path: path}
	return bc
}

func (bc *Blockchain) AddBlock(data []string) *Block {
	for {
		bc.mu.Lock()
		lbh := bc.Blocks[len(bc.Blocks)-1].Hash
		bc.mu.Unlock()

		nb := NewBlock(data, lbh, bc.Difficulty)

		bc.mu.Lock()
		if lbh != bc.Blocks[len(bc.Blocks)-1].Hash {
			bc.mu.Unlock()
			continue
		}
		bc.Blocks = append(bc.Blocks, nb)
		bc.mu.Unlock()
		if err := bc.SaveToFile(); err != nil {
			log.Println("save failed:", err)
		}
		return nb
	}
}

func (bc *Blockchain) IsValid() bool {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if len(bc.Blocks) == 0 {
		return false
	}
	if bc.Blocks[0].PrevBlockHash != genesisPrevHash {
		return false
	}
	if bc.Blocks[0].Hash != bc.Blocks[0].CalculateHash() {
		return false
	}
	target := strings.Repeat("0", bc.Difficulty)
	if !strings.HasPrefix(bc.Blocks[0].Hash, target) {
		return false
	}
	for i := 1; i < len(bc.Blocks); i++ {
		cur := bc.Blocks[i]
		prev := bc.Blocks[i-1]
		if cur.PrevBlockHash != prev.Hash {
			return false
		}
		if cur.Hash != cur.CalculateHash() {
			return false
		}
		if !strings.HasPrefix(cur.Hash, target) {
			return false
		}
	}
	return true
}

func (b *Block) Mine(difficulty int) {
	target := strings.Repeat("0", difficulty)
	for {
		hash := b.CalculateHash()
		if strings.HasPrefix(hash, target) {
			b.Hash = hash
			return
		}
		b.Nonce++
	}
}

type MineRequest struct {
	Data []string `json:"data"`
}

type ValidResponse struct {
	Valid bool `json:"valid"`
}

func (bc *Blockchain) SaveToFile() error {
	bc.mu.Lock()
	bs := append([]*Block(nil), bc.Blocks...)
	d := bc.Difficulty
	bc.mu.Unlock()
	cf := ChainFile{bs, d}
	b, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(bc.path, b, 0o644)
	return err
}

func LoadFromFile(path string) (*Blockchain, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cf ChainFile
	err = json.Unmarshal(b, &cf)
	if err != nil {
		return nil, err
	}
	bc := &Blockchain{Blocks: cf.Blocks, Difficulty: cf.Difficulty}
	if !bc.IsValid() {
		return nil, fmt.Errorf("chain file %s is invalid", path)
	}
	return bc, err
}

func main() {
	bc, err := LoadFromFile(chainFilePath)
	if err != nil {
		log.Fatal(err)
	}
	if bc == nil {
		log.Println("no chain file, creating genesis")
		bc = NewBlockchain(5, chainFilePath)
		if err := bc.SaveToFile(); err != nil {
			log.Fatal(err)
		}
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/blocks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		bc.mu.Lock()
		blocks := append([]*Block(nil), bc.Blocks...)
		bc.mu.Unlock()
		json.NewEncoder(w).Encode(blocks)
	})

	mux.HandleFunc("/mine", func(w http.ResponseWriter, r *http.Request) {
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
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bc.AddBlock(req.Data))
	})

	mux.HandleFunc("/valid", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var v ValidResponse
		v.Valid = bc.IsValid()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(v)
	})

	log.Println("listening on: 8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
