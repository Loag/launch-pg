package export

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"launch-pg/internal/model"
)

// Renderer turns credentials into a document (file or stdout).
type Renderer interface {
	Kind() string
	// NeedsPassword is false for formats that only reference a secret
	// stored elsewhere (e.g. ExternalSecret).
	NeedsPassword() bool
	// Check validates options before anything is changed on the server.
	Check(opts Options) error
	Render(creds []model.Credential, opts Options) ([]byte, error)
}

// Pusher writes each credential to a remote secret store.
type Pusher interface {
	Kind() string
	Check(opts Options) error
	// Push stores one credential and returns a human-readable location.
	Push(ctx context.Context, cred model.Credential, opts Options) (string, error)
}

// Report says where a credential went.
type Report struct {
	Target   string
	Role     string
	Location string
}

// Options every target accepts. The overrides replace connection details,
// e.g. an in-cluster host for a k8s Secret.
const (
	OptOut      = "out"
	OptHost     = "host"
	OptPort     = "port"
	OptSSLMode  = "sslmode"
	OptDatabase = "dbname"
)

// Exporter dispatches targets to renderers and pushers.
type Exporter struct {
	renderers map[string]Renderer
	pushers   map[string]Pusher
}

func NewExporter(renderers []Renderer, pushers []Pusher) *Exporter {
	e := &Exporter{renderers: map[string]Renderer{}, pushers: map[string]Pusher{}}
	for _, r := range renderers {
		e.renderers[r.Kind()] = r
	}
	for _, p := range pushers {
		e.pushers[p.Kind()] = p
	}
	return e
}

// Parse parses and validates targets so mistakes fail before any change.
func (e *Exporter) Parse(specs []string) ([]Target, error) {
	targets := make([]Target, 0, len(specs))
	var errs []error
	for _, s := range specs {
		t, err := ParseTarget(s)
		if err == nil {
			err = e.check(t)
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		targets = append(targets, t)
	}
	return targets, errors.Join(errs...)
}

func (e *Exporter) check(t Target) error {
	if p := t.Options[OptPort]; p != "" {
		if _, err := strconv.Atoi(p); err != nil {
			return fmt.Errorf("export %s: port %q is not a number", t.Raw, p)
		}
	}
	var err error
	switch {
	case e.renderers[t.Kind] != nil:
		err = e.renderers[t.Kind].Check(t.Options)
	case e.pushers[t.Kind] != nil:
		if t.Options[OptOut] != "" {
			return fmt.Errorf("export %s: %q is a secret store; out= does not apply", t.Raw, t.Kind)
		}
		err = e.pushers[t.Kind].Check(t.Options)
	default:
		return fmt.Errorf("export %s: unknown kind %q (have: %v)", t.Raw, t.Kind, e.Kinds())
	}
	if err != nil {
		return fmt.Errorf("export %s: %w", t.Raw, err)
	}
	return nil
}

// Kinds lists every supported target kind.
func (e *Exporter) Kinds() []string {
	var kinds []string
	for k := range e.renderers {
		kinds = append(kinds, k)
	}
	for k := range e.pushers {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	return kinds
}

// NeedsPassword reports whether any target needs the plaintext password.
func (e *Exporter) NeedsPassword(targets []Target) bool {
	for _, t := range targets {
		if r, ok := e.renderers[t.Kind]; ok && !r.NeedsPassword() {
			continue
		}
		return true
	}
	return false
}

// Export sends creds to every target. Rendered output without out= goes to
// stdout. Files are written 0600. All targets are attempted; errors are joined.
func (e *Exporter) Export(ctx context.Context, stdout io.Writer, creds []model.Credential, targets []Target) ([]Report, error) {
	var (
		reports []Report
		errs    []error
	)
	for _, t := range targets {
		adjusted := withOverrides(creds, t.Options)
		var (
			r   []Report
			err error
		)
		if renderer, ok := e.renderers[t.Kind]; ok {
			r, err = e.render(renderer, stdout, adjusted, t)
		} else {
			r, err = e.push(ctx, e.pushers[t.Kind], adjusted, t)
		}
		reports = append(reports, r...)
		if err != nil {
			errs = append(errs, fmt.Errorf("export %s: %w", t.Raw, err))
		}
	}
	return reports, errors.Join(errs...)
}

func (e *Exporter) render(r Renderer, stdout io.Writer, creds []model.Credential, t Target) ([]Report, error) {
	doc, err := r.Render(creds, t.Options)
	if err != nil {
		return nil, err
	}
	location := "stdout"
	if path := t.Options[OptOut]; path != "" {
		if err := writeSecretFile(path, doc); err != nil {
			return nil, err
		}
		location = path
	} else if _, err := stdout.Write(doc); err != nil {
		return nil, err
	}

	reports := make([]Report, len(creds))
	for i, c := range creds {
		reports[i] = Report{Target: t.Kind, Role: c.User, Location: location}
	}
	return reports, nil
}

func (e *Exporter) push(ctx context.Context, p Pusher, creds []model.Credential, t Target) ([]Report, error) {
	var (
		reports []Report
		errs    []error
	)
	for _, c := range creds {
		location, err := p.Push(ctx, c, t.Options)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", c.User, err))
			continue
		}
		reports = append(reports, Report{Target: t.Kind, Role: c.User, Location: location})
	}
	return reports, errors.Join(errs...)
}

func withOverrides(creds []model.Credential, opts Options) []model.Credential {
	out := make([]model.Credential, len(creds))
	for i, c := range creds {
		c.Host = opts.Get(OptHost, c.Host)
		c.SSLMode = opts.Get(OptSSLMode, c.SSLMode)
		c.Database = opts.Get(OptDatabase, c.Database)
		if p, err := strconv.Atoi(opts[OptPort]); err == nil {
			c.Port = p
		}
		out[i] = c
	}
	return out
}

func writeSecretFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create directory for %s: %w", path, err)
	}
	// Remove first so an existing file with looser permissions is replaced.
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
