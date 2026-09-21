package chain

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

func Verify(pubKeyHex string, data []byte, signatureHex string) error {
	pubKeyBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return fmt.Errorf("decode pubkey: %w", err)
	}

	pubKey, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), pubKeyBytes)
	if err != nil {
		return fmt.Errorf("parse pub key: %w", err)
	}

	signature, err := hex.DecodeString(signatureHex)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	hash := sha256.Sum256(data)

	if !ecdsa.VerifyASN1(pubKey, hash[:], signature) {
		return errors.New("invalid signature")
	}

	return nil
}

func Sign(privKey *ecdsa.PrivateKey, data []byte) (string, error) {
	hash := sha256.Sum256(data)
	signature, err := ecdsa.SignASN1(rand.Reader, privKey, hash[:])
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(signature), nil
}
