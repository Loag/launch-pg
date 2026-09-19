package pg

import (
	"strings"
	"testing"

	"launch-pg/internal/config"
)

func TestConnStringQuotesValues(t *testing.T) {
	s := config.Server{Host: "db.local", Port: 5433, User: `o'brien\x`, SSLMode: "require"}
	got := connString(s, "postgres")

	for _, want := range []string{
		`host='db.local'`,
		`port='5433'`,
		`user='o\'brien\\x'`,
		`dbname='postgres'`,
		`sslmode='require'`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("connString missing %s\n got: %s", want, got)
		}
	}
	if strings.Contains(got, "password") {
		t.Error("connString must not contain the password")
	}
}
