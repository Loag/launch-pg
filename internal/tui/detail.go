package tui

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"launch-pg/internal/humanize"
	"launch-pg/internal/model"
)

type detailLoadedMsg struct {
	database string
	db       model.DatabaseInfo
	allDBs   []model.DatabaseInfo
	allRoles []model.RoleInfo
}

// detailScreen shows one database and the roles related to it (its owner,
// roles tagged with its project, and roles granted access), with role actions.
type detailScreen struct {
	env      *env
	server   string
	database string
	actions  roleActions
	db       model.DatabaseInfo
	roles    []model.RoleInfo
	cur      cursor
	loading  bool
	err      error
}

func newDetailScreen(e *env, server, database string) *detailScreen {
	return &detailScreen{
		env: e, server: server, database: database,
		actions: roleActions{env: e, server: server}, loading: true,
	}
}

func (s *detailScreen) Title() string { return s.database }
func (s *detailScreen) Init() tea.Cmd { return s.load }

func (s *detailScreen) load() tea.Msg {
	dbs, err := s.env.svc.Databases.List(s.env.ctx, s.server)
	if err != nil {
		return errMsg{err}
	}
	i := slices.IndexFunc(dbs, func(d model.DatabaseInfo) bool { return d.Name == s.database })
	if i < 0 {
		return errMsg{fmt.Errorf("database %s no longer exists", s.database)}
	}
	roles, err := s.env.svc.Roles.List(s.env.ctx, s.server)
	if err != nil {
		return errMsg{err}
	}
	return detailLoadedMsg{database: s.database, db: dbs[i], allDBs: dbs, allRoles: roles}
}

// relatedRoles returns the owner first, then other related roles by name.
func relatedRoles(db model.DatabaseInfo, all []model.RoleInfo) []model.RoleInfo {
	grantees := db.Grantees()
	var owner, others []model.RoleInfo
	for _, r := range all {
		switch {
		case r.Name == db.Owner:
			owner = append(owner, r)
		case db.Project() != "" && r.Project() == db.Project(), slices.Contains(grantees, r.Name):
			others = append(others, r)
		}
	}
	return append(owner, others...)
}

func (s *detailScreen) groups() []actionGroup {
	general := []action{s.actions.create(s.database)}
	if project := s.db.Project(); project != "" {
		general = append(general, action{
			key: "E", label: "Export the whole project…",
			desc: "Export every login role of project " + project + " in one go.",
			open: func() screen {
				return s.actions.exportMenu(s.projectLoginRoles(), s.database, "export project "+project)
			},
		})
	}
	generalGroup := actionGroup{title: "Database " + s.database, actions: general}
	if len(s.roles) == 0 {
		return []actionGroup{generalGroup}
	}
	role := s.roles[s.cur.pos]
	return []actionGroup{{title: role.Name, actions: s.actions.forRole(role, s.database)}, generalGroup}
}

func (s *detailScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case detailLoadedMsg:
		if msg.database == s.database {
			s.loading, s.err = false, nil
			s.db = msg.db
			s.roles = relatedRoles(msg.db, msg.allRoles)
			s.actions.roles = roleNames(msg.allRoles)
			s.actions.databases = databaseNames(msg.allDBs)
			s.cur.clamp(len(s.roles))
		}
	case errMsg:
		s.loading, s.err = false, msg.err
	case preparedMsg:
		return s, push(newPlanScreen(s.env, msg.prepared, msg.targets))
	case resumeMsg:
		return s, s.load
	case tea.KeyMsg:
		key := msg.String()
		if s.cur.move(key, len(s.roles)) {
			return s, nil
		}
		switch key {
		case "esc", "q", "left":
			return s, pop
		case "r":
			s.loading = true
			return s, s.load
		}
		title := s.database
		if len(s.roles) > 0 {
			title = s.roles[s.cur.pos].Name
		}
		if cmd, ok := handleListKey(s.env, key, title, s.groups()); ok {
			s.err = nil
			return s, cmd
		}
	}
	return s, nil
}

func (s *detailScreen) projectLoginRoles() []string {
	var roles []string
	for _, r := range s.roles {
		if r.Project() == s.db.Project() && r.CanLogin {
			roles = append(roles, r.Name)
		}
	}
	return roles
}

func (s *detailScreen) View() string {
	var b strings.Builder
	if s.loading && s.roles == nil {
		b.WriteString(s.env.loading("Loading " + s.database + "…"))
	} else {
		b.WriteString(s.summary() + "\n\n")
		b.WriteString(sectionStyle.Render("ROLES WITH ACCESS") + "\n")
		if len(s.roles) == 0 {
			b.WriteString(faintStyle.Render("None besides superusers. Press n to create a role, then grant it a preset.") + "\n")
		} else {
			b.WriteString(roleTable(s.roles).render(s.cur.pos, s.env.bodyWidth(), s.env.bodyHeight()-6))
			b.WriteString("\n" + faintStyle.Render(roleLegend))
		}
	}
	if s.err != nil {
		b.WriteString("\n\n" + errorLine(s.err))
	}
	return b.String()
}

func (s *detailScreen) summary() string {
	access := "anyone who can log in (default)"
	if !s.db.DefaultACL {
		access = "only listed roles"
	}
	pairs := [][2]string{
		{"owner", s.db.Owner},
		{"size", humanize.Bytes(s.db.SizeBytes)},
		{"project", orDash(s.db.Project())},
		{"connections", fmt.Sprint(s.db.Connections)},
		{"who can connect", access},
	}
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = labelStyle.Render(p[0]+" ") + p[1]
	}
	return strings.Join(parts, faintStyle.Render("  ·  "))
}

func (s *detailScreen) Help() string {
	return hint("enter", "actions", "g", "grant", "e", "export", "?", "help", "esc", "back")
}
