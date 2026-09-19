package credstore

import (
	"fmt"
	"strings"
)

// LookupEnv matches os.LookupEnv; injected so tests don't touch the real environment.
type LookupEnv func(key string) (string, bool)

// EnvStore reads LAUNCHPG_<SERVER>_PASSWORD. It is read-only.
type EnvStore struct {
	lookup LookupEnv
}

func NewEnvStore(lookup LookupEnv) *EnvStore {
	return &EnvStore{lookup: lookup}
}

// VarName returns the environment variable holding a server's password.
func VarName(server string) string {
	name := strings.ToUpper(strings.ReplaceAll(server, "-", "_"))
	return "LAUNCHPG_" + name + "_PASSWORD"
}

func (e *EnvStore) Get(server string) (string, error) {
	key := VarName(server)
	if pw, ok := e.lookup(key); ok {
		return pw, nil
	}
	return "", fmt.Errorf("%w: $%s is not set", ErrNotFound, key)
}

func (e *EnvStore) Set(string, string) error { return ErrReadOnly }
func (e *EnvStore) Delete(string) error      { return nil }
