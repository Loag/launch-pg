package tui

import (
	"testing"

	"launch-pg/internal/model"
)

func TestRelatedRolesOwnerFirst(t *testing.T) {
	db := model.DatabaseInfo{
		Name: "myapp", Owner: "myapp_owner",
		Comment: "launch-pg:v1 project=myapp",
		ACL:     []model.ACLEntry{{Grantee: "reporting", Privilege: "CONNECT"}},
	}
	all := []model.RoleInfo{
		{Name: "admin"},
		{Name: "myapp_app", Comment: "launch-pg:v1 preset=readwrite project=myapp"},
		{Name: "myapp_owner", Comment: "launch-pg:v1 preset=owner project=myapp"},
		{Name: "other_app", Comment: "launch-pg:v1 project=other"},
		{Name: "reporting"},
	}
	got := relatedRoles(db, all)
	var names []string
	for _, r := range got {
		names = append(names, r.Name)
	}
	want := []string{"myapp_owner", "myapp_app", "reporting"}
	if len(names) != len(want) {
		t.Fatalf("roles = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("roles = %v, want %v", names, want)
		}
	}
}

func TestCursorClamps(t *testing.T) {
	var c cursor
	c.move("up", 3)
	if c.pos != 0 {
		t.Errorf("pos = %d after up at top", c.pos)
	}
	c.move("end", 3)
	c.move("down", 3)
	if c.pos != 2 {
		t.Errorf("pos = %d after down at bottom", c.pos)
	}
	if c.move("x", 3) {
		t.Error("unrelated key reported as handled")
	}
}

func TestIsYes(t *testing.T) {
	for _, s := range []string{"y", "Yes", " TRUE "} {
		if !isYes(s) {
			t.Errorf("isYes(%q) = false", s)
		}
	}
	for _, s := range []string{"", "n", "nope"} {
		if isYes(s) {
			t.Errorf("isYes(%q) = true", s)
		}
	}
}
