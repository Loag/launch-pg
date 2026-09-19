package model

// RoleInfo is an existing role as seen by introspection.
type RoleInfo struct {
	Name            string
	CanLogin        bool
	Superuser       bool
	CreateDB        bool
	CreateRole      bool
	ConnectionLimit int // -1 means unlimited
	MemberOf        []string
	Comment         string
}

// Marker returns launch-pg's ownership marker, if the role has one.
func (r RoleInfo) Marker() (Marker, bool) { return ParseMarker(r.Comment) }

// Project returns the launch-pg project the role is tagged with, or "".
func (r RoleInfo) Project() string {
	m, _ := r.Marker()
	return m.Project
}

// Preset returns the preset the role was tagged with, or "".
func (r RoleInfo) Preset() string {
	m, _ := r.Marker()
	return m.Preset
}
