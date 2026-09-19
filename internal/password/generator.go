// Package password generates role passwords and hashes them client-side.
package password

import "crypto/rand"

// Generator produces new plaintext passwords.
type Generator interface {
	Generate() (string, error)
}

// Random yields 26-character base32 passwords (~130 bits). The alphabet is
// A-Z and 2-7, so values are safe in URLs, env files and YAML unquoted.
type Random struct{}

func (Random) Generate() (string, error) {
	return rand.Text(), nil
}
