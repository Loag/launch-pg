package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"launch-pg/internal/model"
)

// roleTable renders roles the same way on the roles and detail screens.
func roleTable(roles []model.RoleInfo) table {
	rows := make([][]string, len(roles))
	for i, r := range roles {
		limit := "∞"
		if r.ConnectionLimit >= 0 {
			limit = strconv.Itoa(r.ConnectionLimit)
		}
		login := "no"
		if r.CanLogin {
			login = "yes"
		}
		rows[i] = []string{r.Name, login, orDash(r.Preset()), orDash(r.Project()), limit, attributes(r)}
	}
	return table{
		columns: []column{
			{title: "ROLE"}, {title: "LOGIN", width: 5}, {title: "PRESET", width: 10},
			{title: "PROJECT", width: 14}, {title: "LIMIT", width: 5, right: true}, {title: "ATTRIBUTES", width: 22},
		},
		rows: rows,
		rowStyle: func(i int) lipgloss.Style {
			switch {
			case roles[i].Superuser:
				return errorStyle
			case !roles[i].CanLogin:
				return faintStyle
			}
			return lipgloss.NewStyle()
		},
	}
}

const roleLegend = "red = superuser (launch-pg won't change it) · dimmed = can't log in"

func attributes(r model.RoleInfo) string {
	var attrs []string
	if r.Superuser {
		attrs = append(attrs, "superuser")
	}
	if r.CreateDB {
		attrs = append(attrs, "createdb")
	}
	if r.CreateRole {
		attrs = append(attrs, "createrole")
	}
	return orDash(strings.Join(attrs, ","))
}

func roleNames(roles []model.RoleInfo) []string {
	names := make([]string, len(roles))
	for i, r := range roles {
		names[i] = r.Name
	}
	return names
}

func databaseNames(dbs []model.DatabaseInfo) []string {
	names := make([]string, len(dbs))
	for i, d := range dbs {
		names[i] = d.Name
	}
	return names
}
