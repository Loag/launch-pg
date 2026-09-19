package plan

// Action is one logical change. Actions only describe SQL; the Applier runs it.
type Action interface {
	// Kind is a stable machine name, e.g. "create_role".
	Kind() string
	// Describe is a one-line human summary.
	Describe() string
	// Database the action must run in; "" means the maintenance database.
	Database() string
	// Transactional is false for statements Postgres refuses to run inside
	// a transaction block (CREATE/DROP DATABASE).
	Transactional() bool
	// Validate checks fields before any SQL is generated or run.
	Validate() error
	Statements() []Statement
}

// DatabaseDropper is implemented by actions that remove a database, so the
// Applier can first close its own connection to it.
type DatabaseDropper interface {
	DroppedDatabase() string
}
