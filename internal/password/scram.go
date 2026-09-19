package password

import (
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// Hasher turns a plaintext password into what is sent to the server.
type Hasher interface {
	Hash(plain string) (string, error)
}

const scramIterations = 4096 // Postgres' default scram_iterations

// SCRAM builds a SCRAM-SHA-256 verifier in the format Postgres stores in
// pg_authid, so CREATE/ALTER ROLE ... PASSWORD never carries the plaintext.
//
// Postgres applies SASLprep to passwords; generated passwords are plain
// ASCII, for which SASLprep is the identity.
type SCRAM struct {
	salt io.Reader
}

// NewSCRAM takes the salt source (crypto/rand.Reader in production).
func NewSCRAM(salt io.Reader) *SCRAM {
	return &SCRAM{salt: salt}
}

func (s *SCRAM) Hash(plain string) (string, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(s.salt, salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	return scramVerifier(plain, salt, scramIterations)
}

func scramVerifier(plain string, salt []byte, iterations int) (string, error) {
	salted, err := pbkdf2.Key(sha256.New, plain, salt, iterations, sha256.Size)
	if err != nil {
		return "", fmt.Errorf("derive key: %w", err)
	}
	clientKey := hmacSHA256(salted, "Client Key")
	storedKey := sha256.Sum256(clientKey)
	serverKey := hmacSHA256(salted, "Server Key")

	b64 := base64.StdEncoding.EncodeToString
	return fmt.Sprintf("SCRAM-SHA-256$%d:%s$%s:%s",
		iterations, b64(salt), b64(storedKey[:]), b64(serverKey)), nil
}

func hmacSHA256(key []byte, msg string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(msg))
	return mac.Sum(nil)
}
