// Package credstore stores and resolves admin passwords for configured servers.
package credstore

import "errors"

// Store kinds, as referenced by config.Server.CredentialStore.
const (
	KindKeychain = "keychain"
	KindFile     = "file"
	KindEnv      = "env"
)

var (
	ErrNotFound = errors.New("credential not found")
	ErrReadOnly = errors.New("credential store is read-only")
)

// CredentialStore persists one admin password per server name.
type CredentialStore interface {
	Get(server string) (string, error)
	Set(server, password string) error
	Delete(server string) error
}
