package sqlgen

import (
	"fmt"
	"regexp"
	"strings"
)

// maxIdentLen is Postgres' NAMEDATALEN-1.
const maxIdentLen = 63

var objectNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// ValidateName enforces the naming rule for anything launch-pg creates
// (databases, roles, schemas): lowercase, starts with a letter, <= 63 bytes.
// Keeping names unquoted-safe means they are also usable in psql and URLs.
func ValidateName(kind, name string) error {
	if len(name) == 0 || len(name) > maxIdentLen {
		return fmt.Errorf("%s name %q must be 1-%d characters", kind, name, maxIdentLen)
	}
	if !objectNamePattern.MatchString(name) {
		return fmt.Errorf("%s name %q must be lowercase letters, digits and '_', starting with a letter", kind, name)
	}
	if strings.HasPrefix(name, "pg_") {
		return fmt.Errorf("%s name %q uses the reserved prefix \"pg_\"", kind, name)
	}
	return nil
}
