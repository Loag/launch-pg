// Package audit records every change launch-pg makes to a server.
package audit

import "time"

type Status string

const (
	StatusApplied    Status = "applied"
	StatusFailed     Status = "failed"
	StatusRolledBack Status = "rolled_back"
)

// Event is one action's outcome. Statements are the redacted forms; secrets
// never reach the audit log.
type Event struct {
	Time        time.Time `json:"time"`
	PlanID      string    `json:"planId"`
	Server      string    `json:"server"`
	Database    string    `json:"database"`
	Action      string    `json:"action"`
	Description string    `json:"description"`
	Statements  []string  `json:"statements"`
	Status      Status    `json:"status"`
	Error       string    `json:"error,omitempty"`
}
