package export

import (
	"fmt"
	"regexp"
	"strings"

	"launch-pg/internal/model"
)

// EnvRenderer writes a .env / docker-compose env_file. With several
// credentials each key is prefixed with the role name (MYAPP_APP_PGUSER…).
type EnvRenderer struct {
	kind string
}

// NewEnvRenderer registers the env format under a kind ("env", "compose").
func NewEnvRenderer(kind string) EnvRenderer { return EnvRenderer{kind: kind} }

func (r EnvRenderer) Kind() string           { return r.kind }
func (EnvRenderer) NeedsPassword() bool      { return true }
func (EnvRenderer) Check(opts Options) error { return nil }

var safeEnvValue = regexp.MustCompile(`^[A-Za-z0-9_./:@%+=?&-]*$`)

func (EnvRenderer) Render(creds []model.Credential, opts Options) ([]byte, error) {
	var b strings.Builder
	for i, c := range creds {
		prefix := opts.Get("prefix", "")
		if len(creds) > 1 {
			prefix += envName(c.User) + "_"
			if i > 0 {
				b.WriteString("\n")
			}
			fmt.Fprintf(&b, "# %s\n", c.User)
		}
		for _, kv := range envVars(c) {
			fmt.Fprintf(&b, "%s%s=%s\n", prefix, kv.Key, envQuote(kv.Value))
		}
	}
	return []byte(b.String()), nil
}

func envVars(c model.Credential) []Field {
	return []Field{
		{"DATABASE_URL", c.URL()},
		{"PGHOST", c.Host},
		{"PGPORT", fmt.Sprint(c.Port)},
		{"PGDATABASE", c.Database},
		{"PGUSER", c.User},
		{"PGPASSWORD", c.Password},
		{"PGSSLMODE", c.SSLMode},
	}
}

func envName(role string) string {
	return strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(role))
}

func envQuote(v string) string {
	if safeEnvValue.MatchString(v) {
		return v
	}
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, "$", `\$`, "\n", `\n`).Replace(v) + `"`
}
