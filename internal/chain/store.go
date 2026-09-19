package chain

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	ChainFilePath = "./mychain.json"
)

type ChainFile struct {
	Blocks     []*Block `json:"blocks"`
	Difficulty int      `json:"difficulty"`
}

func SaveToFile(bc *Blockchain) error {
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

func saveToFileLocked(bc *Blockchain) error {
	cf := ChainFile{bc.Blocks, bc.Difficulty}
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
	bc := &Blockchain{Blocks: cf.Blocks, Difficulty: cf.Difficulty, path: path}
	if !bc.IsValid() {
		return nil, fmt.Errorf("chain file %s is invalid", path)
	}
	return bc, nil
}
