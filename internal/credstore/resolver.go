package credstore

import (
	"errors"

	"launch-pg/internal/config"
)

// Resolver finds the admin password for a server. An environment variable
// always wins; otherwise the server's configured store is used.
type Resolver struct {
	env      CredentialStore
	registry *Registry
}

func NewResolver(env CredentialStore, registry *Registry) *Resolver {
	return &Resolver{env: env, registry: registry}
}

func (r *Resolver) Password(server config.Server) (string, error) {
	pw, err := r.env.Get(server.Name)
	if err == nil {
		return pw, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return "", err
	}
	store, err := r.registry.Store(server.CredentialStore)
	if err != nil {
		return "", err
	}
	return store.Get(server.Name)
}
