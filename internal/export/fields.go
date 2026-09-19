package export

import (
	"regexp"
	"strconv"
	"strings"

	"launch-pg/internal/model"
)

// DefaultPath is where pushers store a credential and where ExternalSecret
// manifests look for it, so the two line up without configuration.
const DefaultPath = "launch-pg/{database}/{role}"

// Field is one key of an exported credential.
type Field struct {
	Key   string
	Value string
}

// Fields returns the standard secret keys in a fixed order. Every secret
// format (k8s, Vault, AWS) uses these names.
func Fields(c model.Credential) []Field {
	return []Field{
		{"host", c.Host},
		{"port", strconv.Itoa(c.Port)},
		{"dbname", c.Database},
		{"username", c.User},
		{"password", c.Password},
		{"sslmode", c.SSLMode},
		{"DATABASE_URL", c.URL()},
	}
}

// FieldMap is Fields as a map, for JSON payloads.
func FieldMap(c model.Credential) map[string]string {
	m := map[string]string{}
	for _, f := range Fields(c) {
		m[f.Key] = f.Value
	}
	return m
}

// Expand substitutes {role} and {database} in a name template.
func Expand(template string, c model.Credential) string {
	return strings.NewReplacer("{role}", c.User, "{database}", c.Database).Replace(template)
}

var k8sInvalid = regexp.MustCompile(`[^a-z0-9.-]+`)

// K8sName makes a string a valid Kubernetes object name (RFC 1123 subdomain).
func K8sName(s string) string {
	s = k8sInvalid.ReplaceAllString(strings.ToLower(s), "-")
	s = strings.Trim(s, "-.")
	if len(s) > 253 {
		s = strings.TrimRight(s[:253], "-.")
	}
	return s
}
