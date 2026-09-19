package plan

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"launch-pg/internal/audit"
	"launch-pg/internal/config"
	"launch-pg/internal/pg"
)

// Applier executes plans. Consecutive transactional actions on the same
// database share one transaction; non-transactional actions run alone.
// Execution stops at the first failure.
type Applier struct {
	connector pg.Connector
	audit     audit.Log
	now       func() time.Time
}

func NewApplier(connector pg.Connector, log audit.Log, now func() time.Time) *Applier {
	return &Applier{connector: connector, audit: log, now: now}
}

func (a *Applier) Apply(ctx context.Context, server config.Server, p *Plan) (Result, error) {
	if err := p.Validate(); err != nil {
		return Result{Skipped: p.Actions}, err
	}

	run := &applyRun{
		applier: a,
		server:  server,
		planID:  newPlanID(),
		pool:    newSessionPool(a.connector, server),
	}
	defer run.pool.closeAll(ctx)

	all := batches(p.Actions)
	for i, b := range all {
		if err := run.runBatch(ctx, b); err != nil {
			for _, rest := range all[i+1:] {
				run.result.Skipped = append(run.result.Skipped, rest.actions...)
			}
			return run.finish(), err
		}
	}
	return run.finish(), nil
}

// applyRun holds the state of a single Apply call.
type applyRun struct {
	applier   *Applier
	server    config.Server
	planID    string
	pool      *sessionPool
	result    Result
	auditErrs []error
}

func (r *applyRun) finish() Result {
	r.result.AuditErr = errors.Join(r.auditErrs...)
	return r.result
}

func (r *applyRun) runBatch(ctx context.Context, b batch) error {
	for _, act := range b.actions {
		if d, ok := act.(DatabaseDropper); ok {
			if err := r.pool.close(ctx, d.DroppedDatabase()); err != nil {
				return r.fail(b.actions, 0, fmt.Errorf("close connection to %s: %w", d.DroppedDatabase(), err))
			}
		}
	}

	session, err := r.pool.get(ctx, b.database)
	if err != nil {
		return r.fail(b.actions, 0, err)
	}
	if !b.transactional {
		return r.runSingle(ctx, session, b.actions[0])
	}
	return r.runTransaction(ctx, session, b.actions)
}

func (r *applyRun) runSingle(ctx context.Context, session pg.Session, act Action) error {
	if err := execAction(ctx, session, act); err != nil {
		return r.fail([]Action{act}, 0, err)
	}
	r.applied(act)
	return nil
}

func (r *applyRun) runTransaction(ctx context.Context, session pg.Session, actions []Action) error {
	if err := session.Exec(ctx, "BEGIN"); err != nil {
		return r.fail(actions, 0, fmt.Errorf("begin transaction: %w", err))
	}
	for i, act := range actions {
		if err := execAction(ctx, session, act); err != nil {
			rollbackErr := session.Exec(ctx, "ROLLBACK")
			r.rolledBack(actions[:i])
			return errors.Join(r.fail(actions, i, err), rollbackErr)
		}
	}
	if err := session.Exec(ctx, "COMMIT"); err != nil {
		last := len(actions) - 1
		r.rolledBack(actions[:last])
		return r.fail(actions, last, fmt.Errorf("commit: %w", err))
	}
	for _, act := range actions {
		r.applied(act)
	}
	return nil
}

func execAction(ctx context.Context, session pg.Session, act Action) error {
	for _, s := range act.Statements() {
		if err := session.Exec(ctx, s.SQL); err != nil {
			return fmt.Errorf("%s: %w", act.Describe(), err)
		}
	}
	return nil
}

// fail marks actions[i] failed and everything after it skipped.
func (r *applyRun) fail(actions []Action, i int, err error) error {
	r.result.Failed = actions[i]
	r.result.Skipped = append(r.result.Skipped, actions[i+1:]...)
	r.record(actions[i], audit.StatusFailed, err)
	return err
}

func (r *applyRun) applied(act Action) {
	r.result.Applied = append(r.result.Applied, act)
	r.record(act, audit.StatusApplied, nil)
}

func (r *applyRun) rolledBack(actions []Action) {
	r.result.RolledBack = append(r.result.RolledBack, actions...)
	for _, act := range actions {
		r.record(act, audit.StatusRolledBack, nil)
	}
}

func (r *applyRun) record(act Action, status audit.Status, err error) {
	db := act.Database()
	if db == "" {
		db = r.server.MaintenanceDB
	}
	stmts := act.Statements()
	printable := make([]string, len(stmts))
	for i, s := range stmts {
		printable[i] = s.String()
	}
	event := audit.Event{
		Time:        r.applier.now(),
		PlanID:      r.planID,
		Server:      r.server.Name,
		Database:    db,
		Action:      act.Kind(),
		Description: act.Describe(),
		Statements:  printable,
		Status:      status,
	}
	if err != nil {
		event.Error = err.Error()
	}
	if auditErr := r.applier.audit.Record(event); auditErr != nil {
		r.auditErrs = append(r.auditErrs, auditErr)
	}
}

func newPlanID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
