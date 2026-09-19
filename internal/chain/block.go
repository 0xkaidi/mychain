package chain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Tx struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Amount int    `json:"amount"`
}

type Block struct {
	PrevBlockHash string `json:"prev_block_hash"`
	Timestamp     int64  `json:"timestamp"`
	Transactions  []Tx   `json:"transactions"`
	Nonce         int    `json:"nonce"`
	Hash          string `json:"hash"`
}

func (b *Block) CalculateHash() string {
	t, _ := json.Marshal(b.Transactions)
	record := fmt.Sprintf("%s|%d|%s|%d", b.PrevBlockHash, b.Timestamp, string(t), b.Nonce)
	hashBytes := sha256.Sum256([]byte(record))
	return hex.EncodeToString(hashBytes[:])
}

func NewBlock(tx []Tx, prevHash string, Difficulty int) *Block {
	nb := Block{PrevBlockHash: prevHash, Timestamp: time.Now().Unix(), Transactions: tx, Nonce: 0, Hash: ""}
	nb.Mine(Difficulty)
	return &nb
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
