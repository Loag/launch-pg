package audit

// Log persists audit events.
type Log interface {
	Record(Event) error
}

// Discard is a Log that drops everything (tests, or auditing disabled).
type Discard struct{}

func (Discard) Record(Event) error { return nil }
