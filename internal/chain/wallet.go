package chain

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
)

type Wallet struct {
	PrivKey *ecdsa.PrivateKey
}

func NewWallet() (*Wallet, error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate private key: %w", err)
	}

	return &Wallet{
		PrivKey: privKey,
	}, nil
}

func (w *Wallet) PubKeyHex() string {
	pubKeyBytes, _ := w.PrivKey.PublicKey.Bytes()
	return hex.EncodeToString(pubKeyBytes)
}

func Address(pubKeyHex []byte) string {
	hash := sha256.Sum256(pubKeyHex)
	return hex.EncodeToString(hash[:])
}

func (w *Wallet) ToPem() ([]byte, error) {
	derBytes, err := x509.MarshalECPrivateKey(w.PrivKey)
	if err != nil {
		return nil, err
	}
	block := &pem.Block{Type: "EC PRIVATE KEY", Bytes: derBytes}
	return pem.EncodeToMemory(block), nil
}

func WalletFromPem(data []byte) (*Wallet, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("invalid pem")
	}
	derBytes := block.Bytes
	priv, err := x509.ParseECPrivateKey(derBytes)
	if err != nil {
		return nil, err
	}
	return &Wallet{priv}, nil
}
