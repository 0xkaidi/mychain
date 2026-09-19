package chain

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
)

const (
	genesisPrevHash = "0000000000000000000000000000000000000000000000000000000000000000"
)

type Blockchain struct {
	Blocks     []*Block
	Difficulty int
	Pending    []Tx
	path       string
	mu         sync.Mutex
}

func NewBlockchain(difficulty int, path string) *Blockchain {
	fb := NewBlock([]Tx{{From: "", To: "genesis", Amount: 1000}}, genesisPrevHash, difficulty)
	bc := &Blockchain{Blocks: []*Block{fb}, Difficulty: difficulty, path: path}
	return bc
}

func (bc *Blockchain) GetBlocks() []*Block {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	blocks := append([]*Block(nil), bc.Blocks...)
	return blocks
}

func (bc *Blockchain) addBlock(tx []Tx) (*Block, error) {
	for {
		bc.mu.Lock()
		lbh := bc.Blocks[len(bc.Blocks)-1].Hash
		bc.mu.Unlock()

		nb := NewBlock(tx, lbh, bc.Difficulty)

		bc.mu.Lock()
		if lbh != bc.Blocks[len(bc.Blocks)-1].Hash {
			bc.mu.Unlock()
			continue
		}
		bc.Blocks = append(bc.Blocks, nb)
		if err := saveToFileLocked(bc); err != nil {
			bc.Blocks = bc.Blocks[:len(bc.Blocks)-1]
			bc.mu.Unlock()
			log.Println("save failed:", err)
			return nb, err
		}
		bc.mu.Unlock()
		return nb, nil
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

func (bc *Blockchain) Balance(addr string) int {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	balance := 0
	for _, b := range bc.Blocks {
		for _, tx := range b.Transactions {
			if tx.To == addr {
				balance += tx.Amount
			}
			if tx.From != "" && tx.From == addr {
				balance -= tx.Amount
			}
		}
	}
	return balance
}

func (bc *Blockchain) BalanceWithPending(addr string) int {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	balance := 0
	for _, b := range bc.Blocks {
		for _, tx := range b.Transactions {
			if tx.To == addr {
				balance += tx.Amount
			}
			if tx.From != "" && tx.From == addr {
				balance -= tx.Amount
			}
		}
	}
	for _, tx := range bc.Pending {
		if tx.From != "" && tx.From == addr {
			balance -= tx.Amount
		}
		if tx.To == addr {
			balance += tx.Amount
		}
	}
	return balance
}

func (bc *Blockchain) balanceWithPendingLocked(addr string) int {
	balance := 0
	for _, b := range bc.Blocks {
		for _, tx := range b.Transactions {
			if tx.To == addr {
				balance += tx.Amount
			}
			if tx.From != "" && tx.From == addr {
				balance -= tx.Amount
			}
		}
	}
	for _, tx := range bc.Pending {
		if tx.From != "" && tx.From == addr {
			balance -= tx.Amount
		}
		if tx.To == addr {
			balance += tx.Amount
		}
	}
	return balance
}

func (bc *Blockchain) SubmitTx(tx Tx) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if tx.From == "" {
		return errors.New("from cant be empty")
	}
	if bc.balanceWithPendingLocked(tx.From) < tx.Amount {
		return errors.New("insufficient balance")
	}
	bc.Pending = append(bc.Pending, tx)
	return nil
}

func (bc *Blockchain) MinePending() (*Block, error) {
	bc.mu.Lock()
	if len(bc.Pending) == 0 {
		bc.mu.Unlock()
		return nil, fmt.Errorf("no pending")
	}
	tx := append([]Tx(nil), bc.Pending...)
	bc.Pending = nil
	bc.mu.Unlock()
	b, err := bc.addBlock(tx)
	if err != nil {
		bc.mu.Lock()
		bc.Pending = append(bc.Pending, tx...)
		bc.mu.Unlock()
		return nil, err
	}
	return b, nil
}
