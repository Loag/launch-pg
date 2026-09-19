// Package export writes role credentials to files and secret stores.
package export

import (
	"fmt"
	"strings"
)

// Options are a target's key=value settings.
type Options map[string]string

// Get returns the option or def when unset.
func (o Options) Get(key, def string) string {
	if v, ok := o[key]; ok && v != "" {
		return v
	}
	return def
}

// Target is one parsed --export value, e.g. "k8s:namespace=myapp,out=db.yaml".
type Target struct {
	Kind    string
	Options Options
	Raw     string
}

// ParseTarget parses "kind[:key=value,key=value]". Values may contain '='
// (the first '=' separates key and value) but not ','.
func ParseTarget(s string) (Target, error) {
	kind, rest, _ := strings.Cut(strings.TrimSpace(s), ":")
	if kind == "" {
		return Target{}, fmt.Errorf("export target %q: missing kind", s)
	}
	t := Target{Kind: kind, Options: Options{}, Raw: s}
	if rest == "" {
		return t, nil
	}
	for _, pair := range strings.Split(rest, ",") {
		key, value, ok := strings.Cut(pair, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return Target{}, fmt.Errorf("export target %q: expected key=value, got %q", s, pair)
		}
		if _, dup := t.Options[key]; dup {
			return Target{}, fmt.Errorf("export target %q: option %q given twice", s, key)
		}
		t.Options[key] = strings.TrimSpace(value)
	}
	return t, nil
}
