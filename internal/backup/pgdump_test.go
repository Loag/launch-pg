package backup

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"launch-pg/internal/config"
)

type fakeRunner struct {
	output string
	args   []string
	env    []string
}

func (f *fakeRunner) Run(_ context.Context, _ string, args, env []string) ([]byte, error) {
	f.args, f.env = args, env
	for _, a := range args {
		if len(a) > len("--file=") && a[:len("--file=")] == "--file=" {
			_ = os.WriteFile(a[len("--file="):], []byte("dump"), 0o644)
		}
	}
	return []byte(f.output), nil
}

type staticPassword string

func (s staticPassword) Password(config.Server) (string, error) { return string(s), nil }

func TestCheckComparesMajorVersions(t *testing.T) {
	d := NewPgDump("pg_dump", staticPassword("x"), &fakeRunner{output: "pg_dump (PostgreSQL) 15.4\n"})
	if err := d.Check(context.Background(), 15); err != nil {
		t.Errorf("same major: %v", err)
	}
	if err := d.Check(context.Background(), 16); err == nil {
		t.Error("expected error for older pg_dump")
	}
}

func TestDumpPassesConnectionViaEnv(t *testing.T) {
	runner := &fakeRunner{}
	d := NewPgDump("pg_dump", staticPassword("s3cret"), runner)
	path := filepath.Join(t.TempDir(), "sub", "myapp.dump")
	server := config.Server{Host: "db", Port: 5432, User: "admin", SSLMode: "require"}

	if err := d.Dump(context.Background(), server, "myapp", path); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(runner.env, "PGPASSWORD=s3cret") {
		t.Errorf("env = %v", runner.env)
	}
	for _, a := range runner.args {
		if a == "s3cret" {
			t.Error("password must not appear in argv")
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("backup mode = %o, want 600", info.Mode().Perm())
	}
}
