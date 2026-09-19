package model

// DatabaseInfo is an existing database as seen by introspection.
type DatabaseInfo struct {
	Name     string
	Owner    string
	Encoding string
	// SizeBytes is nil when the admin role lacks CONNECT on the database.
	SizeBytes  *int64
	IsTemplate bool
	AllowConn  bool
	// CanConnect is whether the admin role has CONNECT on it.
	CanConnect bool
	Comment    string
	// DefaultACL means no explicit grants exist (PUBLIC may CONNECT).
	DefaultACL bool
	// ACL lists explicit database privileges; the owner's are implicit.
	ACL []ACLEntry
	// Connections is the number of sessions currently open on it.
	Connections int
}

// ACLEntry is one database privilege held by a role (or "PUBLIC").
type ACLEntry struct {
	Grantee   string
	Privilege string
}

// Marker returns launch-pg's ownership marker, if the database has one.
func (d DatabaseInfo) Marker() (Marker, bool) { return ParseMarker(d.Comment) }

// Project returns the launch-pg project the database is tagged with, or "".
func (d DatabaseInfo) Project() string {
	m, _ := d.Marker()
	return m.Project
}

// Grantees returns roles holding privileges, excluding the owner.
func (d DatabaseInfo) Grantees() []string {
	seen := map[string]bool{}
	var out []string
	for _, e := range d.ACL {
		if e.Grantee == d.Owner || seen[e.Grantee] {
			continue
		}
		seen[e.Grantee] = true
		out = append(out, e.Grantee)
	}
	return out
}
