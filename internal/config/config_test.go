package config

import (
	"errors"
	"path/filepath"
	"testing"
)

func testServer(name string) Server {
	return Server{Name: name, Host: "localhost", User: "admin", CredentialStore: "keychain"}
}

func TestAddServerSetsFirstAsDefault(t *testing.T) {
	var cfg Config
	if err := cfg.AddServer(testServer("home")); err != nil {
		t.Fatal(err)
	}
	if err := cfg.AddServer(testServer("work")); err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultServer != "home" {
		t.Errorf("default = %q, want home", cfg.DefaultServer)
	}

	s, err := cfg.Server("")
	if err != nil {
		t.Fatal(err)
	}
	if s.Port != DefaultPort || s.SSLMode != DefaultSSLMode || s.MaintenanceDB != DefaultMaintenanceDB {
		t.Errorf("defaults not applied: %+v", s)
	}
}

func TestAddServerRejectsDuplicatesAndInvalid(t *testing.T) {
	var cfg Config
	_ = cfg.AddServer(testServer("home"))
	if err := cfg.AddServer(testServer("home")); err == nil {
		t.Error("expected duplicate error")
	}
	if err := cfg.AddServer(testServer("bad name")); err == nil {
		t.Error("expected invalid name error")
	}
}

func TestRemoveServerClearsDefault(t *testing.T) {
	var cfg Config
	_ = cfg.AddServer(testServer("home"))
	if err := cfg.RemoveServer("home"); err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultServer != "" {
		t.Errorf("default = %q, want empty", cfg.DefaultServer)
	}
	if err := cfg.RemoveServer("home"); !errors.Is(err, ErrServerNotFound) {
		t.Errorf("err = %v, want ErrServerNotFound", err)
	}
}

func TestFileRepositoryRoundTrip(t *testing.T) {
	repo := NewFileRepository(filepath.Join(t.TempDir(), "sub", "config.yaml"))

	empty, err := repo.Load()
	if err != nil || len(empty.Servers) != 0 {
		t.Fatalf("Load on missing file = %+v, %v", empty, err)
	}

	var cfg Config
	_ = cfg.AddServer(testServer("home"))
	if err := repo.Save(&cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := repo.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DefaultServer != "home" || len(loaded.Servers) != 1 {
		t.Errorf("round trip mismatch: %+v", loaded)
	}
}
