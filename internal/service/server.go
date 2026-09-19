// Package service holds the use cases the CLI and TUI call into.
package service

import (
	"context"
	"errors"
	"fmt"

	"launch-pg/internal/config"
	"launch-pg/internal/credstore"
	"launch-pg/internal/introspect"
	"launch-pg/internal/model"
	"launch-pg/internal/pg"
)

// ServerService manages registered servers and their admin credentials.
type ServerService struct {
	configs      config.Repository
	stores       *credstore.Registry
	connector    pg.Connector
	introspector introspect.Introspector
}

func NewServerService(
	configs config.Repository,
	stores *credstore.Registry,
	connector pg.Connector,
	introspector introspect.Introspector,
) *ServerService {
	return &ServerService{
		configs:      configs,
		stores:       stores,
		connector:    connector,
		introspector: introspector,
	}
}

// Add registers a server and stores its admin password. password may be
// empty only for the env store, which is read-only.
func (s *ServerService) Add(server config.Server, password string, makeDefault bool) error {
	cfg, err := s.configs.Load()
	if err != nil {
		return err
	}
	if err := cfg.AddServer(server); err != nil {
		return err
	}
	if makeDefault {
		if err := cfg.SetDefaultServer(server.Name); err != nil {
			return err
		}
	}

	store, err := s.stores.Store(server.CredentialStore)
	if err != nil {
		return err
	}
	switch {
	case server.CredentialStore == credstore.KindEnv && password != "":
		return fmt.Errorf("the env store is read-only; set $%s instead", credstore.VarName(server.Name))
	case server.CredentialStore != credstore.KindEnv && password == "":
		return errors.New("an admin password is required")
	case password != "":
		if err := store.Set(server.Name, password); err != nil {
			return err
		}
	}

	if err := s.configs.Save(cfg); err != nil {
		// Don't leave an orphaned credential behind.
		return errors.Join(err, store.Delete(server.Name))
	}
	return nil
}

// List returns all servers and the name of the default one.
func (s *ServerService) List() ([]config.Server, string, error) {
	cfg, err := s.configs.Load()
	if err != nil {
		return nil, "", err
	}
	servers := make([]config.Server, len(cfg.Servers))
	for i, srv := range cfg.Servers {
		servers[i] = srv.WithDefaults()
	}
	return servers, cfg.DefaultServer, nil
}

// Remove unregisters a server and deletes its stored credential.
func (s *ServerService) Remove(name string) error {
	cfg, err := s.configs.Load()
	if err != nil {
		return err
	}
	server, err := cfg.Server(name)
	if err != nil {
		return err
	}
	if err := cfg.RemoveServer(name); err != nil {
		return err
	}
	if err := s.configs.Save(cfg); err != nil {
		return err
	}

	store, err := s.stores.Store(server.CredentialStore)
	if err != nil {
		return fmt.Errorf("server removed, but credential not cleaned up: %w", err)
	}
	if err := store.Delete(server.Name); err != nil {
		return fmt.Errorf("server removed, but credential not cleaned up: %w", err)
	}
	return nil
}

// SetDefault makes name the server used when --server is omitted.
func (s *ServerService) SetDefault(name string) error {
	cfg, err := s.configs.Load()
	if err != nil {
		return err
	}
	if err := cfg.SetDefaultServer(name); err != nil {
		return err
	}
	return s.configs.Save(cfg)
}

// Test connects to the server (default when name is empty) and reports
// its version and the admin role's capabilities.
func (s *ServerService) Test(ctx context.Context, name string) (model.ServerInfo, error) {
	cfg, err := s.configs.Load()
	if err != nil {
		return model.ServerInfo{}, err
	}
	server, err := cfg.Server(name)
	if err != nil {
		return model.ServerInfo{}, err
	}

	session, err := s.connector.Connect(ctx, server, "")
	if err != nil {
		return model.ServerInfo{}, err
	}
	defer session.Close(ctx)

	return s.introspector.ServerInfo(ctx, session)
}
