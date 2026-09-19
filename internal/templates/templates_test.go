package templates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuiltinsExpand(t *testing.T) {
	cat := NewCatalog(filepath.Join(t.TempDir(), "missing"))
	all, err := cat.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("got %d built-in templates, want 3", len(all))
	}
	for _, tmpl := range all {
		p, err := tmpl.Expand("myapp")
		if err != nil {
			t.Errorf("%s: %v", tmpl.Name, err)
			continue
		}
		owner, _ := p.Owner()
		if owner.Name != "myapp_owner" {
			t.Errorf("%s: owner = %s", tmpl.Name, owner.Name)
		}
	}
}

func TestUserTemplateOverridesBuiltin(t *testing.T) {
	dir := t.TempDir()
	yaml := "name: minimal\nschema: app\nroles:\n  - suffix: owner\n    preset: owner\n"
	if err := os.WriteFile(filepath.Join(dir, "mine.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	tmpl, err := NewCatalog(dir).Get("minimal")
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Schema != "app" {
		t.Errorf("schema = %q, want user override", tmpl.Schema)
	}
}

func TestExpandRejectsBadNames(t *testing.T) {
	tmpl := Template{Name: "x", Roles: []RoleTemplate{{Suffix: "owner", Preset: "owner"}}}
	if _, err := tmpl.Expand("My-App"); err == nil {
		t.Error("expected invalid name error")
	}
}
