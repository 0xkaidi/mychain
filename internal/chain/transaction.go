package chain

import (
	"fmt"
)

type Tx struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    int    `json:"amount"`
	PubKey    string `json:"pub_key"`
	Signature string `json:"signature"`
}

func (tx Tx) SigningBytes() []byte {
	record := fmt.Sprintf("%s|%s|%d|%s", tx.From, tx.To, tx.Amount, tx.PubKey)
	return []byte(record)
}

func (tx Tx) VerifySignature() error {
	err := Verify(tx.PubKey, tx.SigningBytes(), tx.Signature)
	if err != nil {
		return fmt.Errorf("verify failed: %w", err)
	}
	return nil
}
