package sqlgen

import "testing"

func TestIdent(t *testing.T) {
	cases := map[string]string{
		"myapp":     `"myapp"`,
		`my"app`:    `"my""app"`,
		"MixedCase": `"MixedCase"`,
	}
	for in, want := range cases {
		if got := Ident(in); got != want {
			t.Errorf("Ident(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestLiteral(t *testing.T) {
	cases := map[string]string{
		"plain":      `'plain'`,
		"it's":       `'it''s'`,
		`back\slash`: `E'back\\slash'`,
		`both'\`:     `E'both''\\'`,
	}
	for in, want := range cases {
		if got := Literal(in); got != want {
			t.Errorf("Literal(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestValidateName(t *testing.T) {
	valid := []string{"myapp", "my_app_2", "a"}
	invalid := []string{"", "MyApp", "2app", "my-app", "pg_app", "has space",
		"a234567890123456789012345678901234567890123456789012345678901234"}

	for _, n := range valid {
		if err := ValidateName("database", n); err != nil {
			t.Errorf("ValidateName(%q) unexpected error: %v", n, err)
		}
	}
	for _, n := range invalid {
		if err := ValidateName("database", n); err == nil {
			t.Errorf("ValidateName(%q) expected error", n)
		}
	}
}
