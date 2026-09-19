package model

import (
	"sort"
	"strings"
)

const markerPrefix = "launch-pg:v1"

// Marker is ownership metadata launch-pg stores in COMMENT ON ROLE/DATABASE,
// so the server itself records which project an object belongs to and which
// preset a role was declared with. No local state file is needed.
type Marker struct {
	Project string
	Preset  string
}

// String renders e.g. "launch-pg:v1 preset=readwrite project=myapp".
func (m Marker) String() string {
	fields := map[string]string{"project": m.Project, "preset": m.Preset}
	keys := make([]string, 0, len(fields))
	for k, v := range fields {
		if v != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	parts := []string{markerPrefix}
	for _, k := range keys {
		parts = append(parts, k+"="+fields[k])
	}
	return strings.Join(parts, " ")
}

// ParseMarker reads a comment. ok is false if launch-pg didn't write it.
func ParseMarker(comment string) (m Marker, ok bool) {
	fields := strings.Fields(comment)
	if len(fields) == 0 || fields[0] != markerPrefix {
		return Marker{}, false
	}
	for _, f := range fields[1:] {
		key, value, found := strings.Cut(f, "=")
		if !found {
			continue
		}
		switch key {
		case "project":
			m.Project = value
		case "preset":
			m.Preset = value
		}
	}
	return m, true
}
