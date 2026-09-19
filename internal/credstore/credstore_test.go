package credstore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"launch-pg/internal/config"
)

func TestFileStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pgpass")
	store := NewFileStore(path)

	if _, err := store.Get("home"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get on empty store: err = %v, want ErrNotFound", err)
	}
	if err := store.Set("home", "pa:ss#word"); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get("home")
	if err != nil || got != "pa:ss#word" {
		t.Fatalf("Get = %q, %v", got, err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file mode = %o, want 600", perm)
	}

	if err := store.Delete("home"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get("home"); !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete: err = %v, want ErrNotFound", err)
	}
}

func TestFileStoreRejectsNewlines(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "pgpass"))
	if err := store.Set("home", "a\nb"); err == nil {
		t.Error("expected error for newline in password")
	}
}

func TestVarName(t *testing.T) {
	if got := VarName("my-server"); got != "LAUNCHPG_MY_SERVER_PASSWORD" {
		t.Errorf("VarName = %q", got)
	}
}

// memStore is an in-memory CredentialStore for tests.
type memStore map[string]string

func (m memStore) Get(s string) (string, error) {
	if pw, ok := m[s]; ok {
		return pw, nil
	}
	return "", ErrNotFound
}
func (m memStore) Set(s, pw string) error { m[s] = pw; return nil }
func (m memStore) Delete(s string) error  { delete(m, s); return nil }

func TestResolverPrefersEnv(t *testing.T) {
	env := map[string]string{}
	envStore := NewEnvStore(func(k string) (string, bool) { v, ok := env[k]; return v, ok })
	keychain := memStore{"home": "from-keychain"}
	resolver := NewResolver(envStore, NewRegistry(map[string]CredentialStore{KindKeychain: keychain}))
	server := config.Server{Name: "home", CredentialStore: KindKeychain}

	if pw, _ := resolver.Password(server); pw != "from-keychain" {
		t.Errorf("without env: got %q", pw)
	}
	env["LAUNCHPG_HOME_PASSWORD"] = "from-env"
	if pw, _ := resolver.Password(server); pw != "from-env" {
		t.Errorf("with env: got %q", pw)
	}
}
