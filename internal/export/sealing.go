package export

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
)

const sessionKeyBytes = 32

// hybridEncrypt matches sealed-secrets' crypto.HybridEncrypt:
//
//	uint16(len(rsaCiphertext)) || RSA-OAEP-SHA256(sessionKey, label) || AES-256-GCM(plaintext)
//
// The GCM nonce is all zeros, which is safe because every session key is
// random and used exactly once.
func hybridEncrypt(rnd io.Reader, pub *rsa.PublicKey, plaintext, label []byte) ([]byte, error) {
	sessionKey := make([]byte, sessionKeyBytes)
	if _, err := io.ReadFull(rnd, sessionKey); err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	rsaCiphertext, err := rsa.EncryptOAEP(sha256.New(), rnd, pub, sessionKey, label)
	if err != nil {
		return nil, err
	}

	out := make([]byte, 2, 2+len(rsaCiphertext)+len(plaintext)+gcm.Overhead())
	binary.BigEndian.PutUint16(out, uint16(len(rsaCiphertext)))
	out = append(out, rsaCiphertext...)
	return gcm.Seal(out, make([]byte, gcm.NonceSize()), plaintext, nil), nil
}

// loadSealingKey reads the controller's public certificate
// (from `kubeseal --fetch-cert`).
func loadSealingKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read sealing cert: %w", err)
	}
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			return nil, fmt.Errorf("%s: no PEM certificate found", path)
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		pub, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("sealing certificate does not hold an RSA key")
		}
		return pub, nil
	}
}
