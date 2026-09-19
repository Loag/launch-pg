package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"launch-pg/internal/config"
	"launch-pg/internal/pg"
)

// Dumper backs up a single database to a file.
type Dumper interface {
	// Check fails early if backups can't work against this server version.
	Check(ctx context.Context, serverMajor int) error
	Dump(ctx context.Context, server config.Server, database, path string) error
}

// PgDump shells out to pg_dump in custom format (restore with pg_restore).
type PgDump struct {
	binary    string
	passwords pg.PasswordResolver
	runner    CommandRunner
}

func NewPgDump(binary string, passwords pg.PasswordResolver, runner CommandRunner) *PgDump {
	return &PgDump{binary: binary, passwords: passwords, runner: runner}
}

var versionPattern = regexp.MustCompile(`\(PostgreSQL\) (\d+)`)

// Check verifies pg_dump exists and is at least the server's major version;
// an older pg_dump refuses to dump a newer server.
func (d *PgDump) Check(ctx context.Context, serverMajor int) error {
	out, err := d.runner.Run(ctx, d.binary, []string{"--version"}, nil)
	if err != nil {
		return fmt.Errorf("pg_dump is required for --backup: %w", err)
	}
	m := versionPattern.FindSubmatch(out)
	if m == nil {
		return fmt.Errorf("unrecognized pg_dump version output: %q", out)
	}
	major, _ := strconv.Atoi(string(m[1]))
	if major < serverMajor {
		return fmt.Errorf("pg_dump %d is older than server %d; install a newer client", major, serverMajor)
	}
	return nil
}

func (d *PgDump) Dump(ctx context.Context, server config.Server, database, path string) error {
	password, err := d.passwords.Password(server)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create backup dir: %w", err)
	}

	env := []string{
		"PGHOST=" + server.Host,
		"PGPORT=" + strconv.Itoa(server.Port),
		"PGUSER=" + server.User,
		"PGSSLMODE=" + server.SSLMode,
		"PGPASSWORD=" + password,
		"PGAPPNAME=launch-pg",
	}
	args := []string{"--format=custom", "--file=" + path, "--dbname=" + database}
	if _, err := d.runner.Run(ctx, d.binary, args, env); err != nil {
		return fmt.Errorf("backup of %s failed: %w", database, err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("restrict backup permissions: %w", err)
	}
	return nil
}
