package chain

import (
	"encoding/hex"
	"errors"
	"log"
	"strings"
	"sync"
)

const (
	genesisPrevHash = "0000000000000000000000000000000000000000000000000000000000000000"
	MiningReward    = 50
	Difficulty      = 5
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

func (bc *Blockchain) isValidGenesisLocked() bool {
	if bc.Blocks[0].PrevBlockHash != genesisPrevHash {
		return false
	}
	if bc.Blocks[0].Hash != bc.Blocks[0].CalculateHash() {
		return false
	}
	if bc.Difficulty != Difficulty {
		return false
	}
	target := strings.Repeat("0", bc.Difficulty)
	if !strings.HasPrefix(bc.Blocks[0].Hash, target) {
		return false
	}
	return true
}

func (bc *Blockchain) isValidHashesLocked() bool {
	target := strings.Repeat("0", bc.Difficulty)
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

func (bc *Blockchain) isValidTransactionsLocked() bool {
	balances := make(map[string]int)
	for i, b := range bc.Blocks {
		coinbaseCount := 0
		for _, tx := range b.Transactions {
			if tx.From == "" {
				// coinbase
				coinbaseCount++
				if i > 0 && tx.Amount != MiningReward {
					return false
				}
				if tx.To == "" {
					return false
				}
				balances[tx.To] += tx.Amount
			} else {
				// 普通交易
				if tx.PubKey == "" || tx.Signature == "" {
					return false
				}
				pubKeyBytes, err := hex.DecodeString(tx.PubKey)
				if err != nil {
					return false
				}
				if Address(pubKeyBytes) != tx.From {
					return false
				}
				if err := tx.VerifySignature(); err != nil {
					return false
				}
				if balances[tx.From] < tx.Amount {
					return false
				}
				balances[tx.From] -= tx.Amount
				balances[tx.To] += tx.Amount
			}
		}
		if i > 0 && coinbaseCount != 1 {
			return false
		}
	}
	return true
}

func (bc *Blockchain) isValidLocked() bool {
	if len(bc.Blocks) == 0 {
		return false
	}
	if !bc.isValidGenesisLocked() {
		return false
	}
	if !bc.isValidHashesLocked() {
		return false
	}
	if !bc.isValidTransactionsLocked() {
		return false
	}
	return true
}

func (bc *Blockchain) IsValid() bool {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	return bc.isValidLocked()
}

func (bc *Blockchain) BalanceWithPending(addr string) int {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	return bc.balanceWithPendingLocked(addr)
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
	if tx.PubKey == "" {
		return errors.New("pub key cant be empty")
	}
	if tx.Signature == "" {
		return errors.New("signature cant be empty")
	}
	pubKeyBytes, err := hex.DecodeString(tx.PubKey)
	if err != nil {
		return err
	}
	if Address(pubKeyBytes) != tx.From {
		return errors.New("pub key and address does not match")
	}
	if err := tx.VerifySignature(); err != nil {
		return err
	}
	if bc.balanceWithPendingLocked(tx.From) < tx.Amount {
		return errors.New("insufficient balance")
	}
	bc.Pending = append(bc.Pending, tx)
	return nil
}

func (bc *Blockchain) Mine(miner string) (*Block, error) {
	bc.mu.Lock()
	tx := append([]Tx(nil), bc.Pending...)
	bc.Pending = nil
	bc.mu.Unlock()

	coinbase := Tx{From: "", To: miner, Amount: MiningReward}
	allTx := append(tx, coinbase)

	b, err := bc.addBlock(allTx)
	if err != nil {
		bc.mu.Lock()
		bc.Pending = append(bc.Pending, tx...)
		bc.mu.Unlock()
		return nil, err
	}
	return b, nil
}
