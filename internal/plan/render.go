package plan

import (
	"fmt"
	"io"
)

// Render writes the plan as annotated SQL, safe to print (secrets redacted).
// maintenanceDB names the database used for actions with no Database().
func Render(w io.Writer, p *Plan, maintenanceDB string) error {
	current := ""
	for i, a := range p.Actions {
		db := a.Database()
		if db == "" {
			db = maintenanceDB
		}
		if i == 0 || db != current {
			if _, err := fmt.Fprintf(w, "\n\\connect %s\n", db); err != nil {
				return err
			}
			current = db
		}

		note := ""
		if !a.Transactional() {
			note = " (outside transaction)"
		}
		if _, err := fmt.Fprintf(w, "-- %d. %s%s\n", i+1, a.Describe(), note); err != nil {
			return err
		}
		for _, s := range a.Statements() {
			if _, err := fmt.Fprintf(w, "%s;\n", s); err != nil {
				return err
			}
		}
	}
	return nil
}
