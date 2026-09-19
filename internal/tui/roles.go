package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"launch-pg/internal/model"
)

type rolesLoadedMsg struct {
	server string
	roles  []model.RoleInfo
	dbs    []model.DatabaseInfo
}

// rolesScreen lists every role on a server (pg_* hidden) with role actions.
type rolesScreen struct {
	env     *env
	server  string
	actions roleActions
	roles   []model.RoleInfo
	cur     cursor
	loading bool
	err     error
}

func newRolesScreen(e *env, server string) *rolesScreen {
	return &rolesScreen{env: e, server: server, actions: roleActions{env: e, server: server}, loading: true}
}

func (s *rolesScreen) Title() string { return "roles" }
func (s *rolesScreen) Init() tea.Cmd { return s.load }

func (s *rolesScreen) load() tea.Msg {
	roles, err := s.env.svc.Roles.List(s.env.ctx, s.server)
	if err != nil {
		return errMsg{err}
	}
	dbs, err := s.env.svc.Databases.List(s.env.ctx, s.server)
	if err != nil {
		return errMsg{err}
	}
	return rolesLoadedMsg{server: s.server, roles: roles, dbs: dbs}
}

func (s *rolesScreen) groups() []actionGroup {
	general := actionGroup{title: "Server " + s.server, actions: []action{s.actions.create("")}}
	if len(s.roles) == 0 {
		return []actionGroup{general}
	}
	role := s.roles[s.cur.pos]
	return []actionGroup{{title: role.Name, actions: s.actions.forRole(role, "")}, general}
}

func (s *rolesScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case rolesLoadedMsg:
		if msg.server == s.server {
			s.loading, s.err, s.roles = false, nil, msg.roles
			s.actions.roles = roleNames(msg.roles)
			s.actions.databases = databaseNames(msg.dbs)
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
		title := "roles"
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

func (s *rolesScreen) View() string {
	var b strings.Builder
	switch {
	case s.loading && s.roles == nil:
		b.WriteString(s.env.loading("Loading roles…"))
	case len(s.roles) == 0:
		b.WriteString("No roles. Press " + keyStyle.Render("n") + " to create one.\n")
	default:
		b.WriteString(roleTable(s.roles).render(s.cur.pos, s.env.bodyWidth(), s.env.bodyHeight()-2))
		b.WriteString("\n" + faintStyle.Render(roleLegend))
	}
	if s.err != nil {
		b.WriteString("\n\n" + errorLine(s.err))
	}
	return b.String()
}

func (s *rolesScreen) Help() string {
	return hint("enter", "actions", "n", "new role", "?", "help", "esc", "back")
}
