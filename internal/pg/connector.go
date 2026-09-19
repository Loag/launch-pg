package pg

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"launch-pg/internal/config"
)

const connectTimeout = 10 * time.Second

// PasswordResolver supplies the admin password for a server.
type PasswordResolver interface {
	Password(server config.Server) (string, error)
}

// Connector opens admin sessions to a server.
type Connector interface {
	// Connect opens a session to database, or the server's maintenance
	// database when database is empty.
	Connect(ctx context.Context, server config.Server, database string) (Session, error)
}

// PgxConnector implements Connector with pgx.
type PgxConnector struct {
	passwords PasswordResolver
}

func NewConnector(passwords PasswordResolver) *PgxConnector {
	return &PgxConnector{passwords: passwords}
}

func (c *PgxConnector) Connect(ctx context.Context, server config.Server, database string) (Session, error) {
	password, err := c.passwords.Password(server)
	if err != nil {
		return nil, fmt.Errorf("admin password for %q: %w", server.Name, err)
	}
	if database == "" {
		database = server.MaintenanceDB
	}

	cfg, err := pgx.ParseConfig(connString(server, database))
	if err != nil {
		return nil, fmt.Errorf("build connection config: %w", err)
	}
	// Set after parsing so the password never needs escaping into a DSN.
	cfg.Password = password

	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect to %s/%s: %w", server.Name, database, err)
	}
	return &connSession{conn: conn}, nil
}

// connString builds a keyword/value DSN with every value quoted.
func connString(s config.Server, database string) string {
	pairs := [][2]string{
		{"host", s.Host},
		{"port", strconv.Itoa(s.Port)},
		{"user", s.User},
		{"dbname", database},
		{"sslmode", s.SSLMode},
		{"connect_timeout", strconv.Itoa(int(connectTimeout.Seconds()))},
		{"application_name", "launch-pg"},
	}
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = p[0] + "=" + quoteDSNValue(p[1])
	}
	return strings.Join(parts, " ")
}

func quoteDSNValue(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `'`, `\'`)
	return "'" + v + "'"
}
