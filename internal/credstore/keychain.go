package credstore

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

// KeychainStore keeps passwords in the OS keychain (macOS Keychain,
// Linux Secret Service, Windows Credential Manager).
type KeychainStore struct {
	service string
}

func NewKeychainStore(service string) *KeychainStore {
	return &KeychainStore{service: service}
}

func (k *KeychainStore) Get(server string) (string, error) {
	pw, err := keyring.Get(k.service, server)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", fmt.Errorf("%w: keychain entry for %q", ErrNotFound, server)
	}
	if err != nil {
		return "", fmt.Errorf("read keychain: %w", err)
	}
	return pw, nil
}

func (k *KeychainStore) Set(server, password string) error {
	if err := keyring.Set(k.service, server, password); err != nil {
		return fmt.Errorf("write keychain: %w", err)
	}
	return nil
}

func (k *KeychainStore) Delete(server string) error {
	err := keyring.Delete(k.service, server)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("delete keychain entry: %w", err)
	}
	return nil
}
