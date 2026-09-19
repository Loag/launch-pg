package plan

import (
	"fmt"
	"io"
)

// Result reports how far an Apply got.
type Result struct {
	// Applied actions are committed on the server.
	Applied []Action
	// Failed is the action that errored, if any.
	Failed Action
	// RolledBack are actions from the failed transaction that were undone.
	RolledBack []Action
	// Skipped actions were never attempted.
	Skipped []Action
	// AuditErr is set when the audit log could not be written. It does not
	// mean the changes failed.
	AuditErr error
}

// Partial reports whether some but not all actions were applied.
func (r Result) Partial() bool {
	return r.Failed != nil && len(r.Applied) > 0
}

// WriteSummary describes the outcome for humans.
func (r Result) WriteSummary(w io.Writer) {
	if r.Failed == nil {
		fmt.Fprintf(w, "Applied %d action(s).\n", len(r.Applied))
		return
	}
	fmt.Fprintf(w, "Failed: %s\n", r.Failed.Describe())
	if len(r.Applied) > 0 {
		fmt.Fprintf(w, "%d action(s) were applied before the failure and remain in place:\n", len(r.Applied))
		for _, a := range r.Applied {
			fmt.Fprintf(w, "  - %s\n", a.Describe())
		}
	}
	if len(r.RolledBack) > 0 {
		fmt.Fprintf(w, "%d action(s) were rolled back.\n", len(r.RolledBack))
	}
	if len(r.Skipped) > 0 {
		fmt.Fprintf(w, "%d action(s) were not attempted.\n", len(r.Skipped))
	}
}
