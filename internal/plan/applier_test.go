package plan

import (
	"context"
	"reflect"
	"testing"
	"time"

	"launch-pg/internal/audit"
	"launch-pg/internal/config"
)

var testServer = config.Server{Name: "home", MaintenanceDB: "postgres"}

func fixedNow() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }

func TestApplyBatchesTransactionsPerDatabase(t *testing.T) {
	conn := &fakeConnector{}
	log := &memAudit{}
	p := &Plan{}
	p.Add(
		fakeAction{name: "role a"},
		fakeAction{name: "role b"},
		fakeAction{name: "create db", nonTx: true},
		fakeAction{name: "schema", db: "app"},
		fakeAction{name: "grant", db: "app"},
	)

	res, err := NewApplier(conn, log, fixedNow).Apply(context.Background(), testServer, p)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"postgres: BEGIN", "postgres: -- role a", "postgres: -- role b", "postgres: COMMIT",
		"postgres: -- create db",
		"app: BEGIN", "app: -- schema", "app: -- grant", "app: COMMIT",
	}
	if !reflect.DeepEqual(conn.log, want) {
		t.Errorf("executed:\n%v\nwant:\n%v", conn.log, want)
	}
	if len(res.Applied) != 5 || res.Failed != nil {
		t.Errorf("result = %+v", res)
	}
	if len(log.events) != 5 || log.events[0].Status != audit.StatusApplied {
		t.Errorf("audit events = %+v", log.events)
	}
}

func TestApplyRollsBackFailedTransaction(t *testing.T) {
	conn := &fakeConnector{failOn: "grant"}
	p := &Plan{}
	p.Add(
		fakeAction{name: "role"},
		fakeAction{name: "grant"},
		fakeAction{name: "later", db: "app"},
	)

	res, err := NewApplier(conn, audit.Discard{}, fixedNow).Apply(context.Background(), testServer, p)
	if err == nil {
		t.Fatal("expected error")
	}
	if res.Failed == nil || res.Failed.Describe() != "grant" {
		t.Errorf("failed = %v", res.Failed)
	}
	if len(res.RolledBack) != 1 || len(res.Skipped) != 1 || len(res.Applied) != 0 {
		t.Errorf("result = %+v", res)
	}
	if last := conn.log[len(conn.log)-1]; last != "postgres: ROLLBACK" {
		t.Errorf("last statement = %q, want ROLLBACK", last)
	}
}

func TestApplyClosesSessionBeforeDroppingDatabase(t *testing.T) {
	conn := &fakeConnector{}
	p := &Plan{}
	p.Add(
		fakeAction{name: "drop owned", db: "app"},
		droppingAction{fakeAction{name: "drop db", nonTx: true, drops: "app"}},
	)

	if _, err := NewApplier(conn, audit.Discard{}, fixedNow).Apply(context.Background(), testServer, p); err != nil {
		t.Fatal(err)
	}
	if len(conn.closed) == 0 || conn.closed[0] != "app" {
		t.Errorf("closed = %v, want app closed first", conn.closed)
	}
}

func TestRedactedStatementsNeverAudited(t *testing.T) {
	log := &memAudit{}
	p := &Plan{}
	p.Add(fakeAction{name: "pw", sql: []Statement{Redacted("PASSWORD 'secret'", "PASSWORD '***'")}})

	if _, err := NewApplier(&fakeConnector{}, log, fixedNow).Apply(context.Background(), testServer, p); err != nil {
		t.Fatal(err)
	}
	if got := log.events[0].Statements[0]; got != "PASSWORD '***'" {
		t.Errorf("audited statement = %q", got)
	}
}
