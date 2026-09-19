package service

import (
	"context"

	"launch-pg/internal/config"
	"launch-pg/internal/introspect"
	"launch-pg/internal/model"
	"launch-pg/internal/pg"
)

// StateLoader resolves a server and snapshots its current state.
type StateLoader struct {
	configs      config.Repository
	connector    pg.Connector
	introspector introspect.Introspector
}

func NewStateLoader(configs config.Repository, connector pg.Connector, introspector introspect.Introspector) *StateLoader {
	return &StateLoader{configs: configs, connector: connector, introspector: introspector}
}

// Load returns the named server (default if empty) and its state.
func (l *StateLoader) Load(ctx context.Context, serverName string) (config.Server, model.ServerState, error) {
	cfg, err := l.configs.Load()
	if err != nil {
		return config.Server{}, model.ServerState{}, err
	}
	server, err := cfg.Server(serverName)
	if err != nil {
		return config.Server{}, model.ServerState{}, err
	}

	session, err := l.connector.Connect(ctx, server, "")
	if err != nil {
		return config.Server{}, model.ServerState{}, err
	}
	defer session.Close(ctx)

	var state model.ServerState
	if state.Info, err = l.introspector.ServerInfo(ctx, session); err != nil {
		return config.Server{}, model.ServerState{}, err
	}
	if state.Databases, err = l.introspector.Databases(ctx, session); err != nil {
		return config.Server{}, model.ServerState{}, err
	}
	if state.Roles, err = l.introspector.Roles(ctx, session); err != nil {
		return config.Server{}, model.ServerState{}, err
	}
	return server, state, nil
}
