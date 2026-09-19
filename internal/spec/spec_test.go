package spec

import (
	"path/filepath"
	"strings"
	"testing"

	"launch-pg/internal/model"
	"launch-pg/internal/templates"
)

const fullSpec = `
server: home
project: myapp
template: webapp
database:
  schema: app
  extensions: [citext, pgcrypto]
roles:
  - name: myapp_worker
    preset: readwrite
    connectionLimit: 10
  - name: myapp_ro
    preset: readonly
    login: false
export:
  - k8s:namespace=myapp
`

func resolver(t *testing.T) *Resolver {
	return NewResolver(templates.NewCatalog(filepath.Join(t.TempDir(), "none")))
}

func TestResolveMergesTemplateAndOverrides(t *testing.T) {
	f, err := Parse(strings.NewReader(fullSpec))
	if err != nil {
		t.Fatal(err)
	}
	p, err := resolver(t).Resolve(f)
	if err != nil {
		t.Fatal(err)
	}

	if p.Database.Schema != "app" {
		t.Errorf("schema = %q", p.Database.Schema)
	}
	if got := strings.Join(p.Database.Extensions, ","); got != "pgcrypto,citext" {
		t.Errorf("extensions = %s (want template's plus new, no duplicates)", got)
	}

	roles := map[string]model.RoleSpec{}
	for _, r := range p.Roles {
		roles[r.Name] = r
	}
	if len(roles) != 4 {
		t.Errorf("roles = %v", p.Roles)
	}
	if r := roles["myapp_ro"]; r.Login {
		t.Error("myapp_ro override should disable login")
	}
	if r := roles["myapp_worker"]; r.ConnectionLimit != 10 || !r.Login {
		t.Errorf("myapp_worker = %+v", r)
	}
	if f.Export[0] != "k8s:namespace=myapp" {
		t.Errorf("export = %v", f.Export)
	}
}

func TestResolveWithoutTemplateNeedsOwner(t *testing.T) {
	f, err := Parse(strings.NewReader("project: myapp\nroles:\n  - name: myapp_app\n    preset: readwrite\n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver(t).Resolve(f); err == nil {
		t.Error("expected missing owner error")
	}
}

func TestParseRejectsUnknownFields(t *testing.T) {
	if _, err := Parse(strings.NewReader("project: myapp\nrolez: []\n")); err == nil {
		t.Error("expected unknown field error")
	}
	if _, err := Parse(strings.NewReader("")); err == nil {
		t.Error("expected empty spec error")
	}
}
