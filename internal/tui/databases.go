package tui

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"launch-pg/internal/humanize"
	"launch-pg/internal/model"
)

type databasesLoadedMsg struct {
	server string
	dbs    []model.DatabaseInfo
}

// databasesScreen lists a server's databases and starts project-level actions.
type databasesScreen struct {
	env     *env
	server  string
	dbs     []model.DatabaseInfo
	cur     cursor
	loading bool
	err     error
}

func newDatabasesScreen(e *env, server string) *databasesScreen {
	return &databasesScreen{env: e, server: server, loading: true}
}

func (s *databasesScreen) Title() string { return s.server }
func (s *databasesScreen) Init() tea.Cmd { return s.load }

func (s *databasesScreen) load() tea.Msg {
	dbs, err := s.env.svc.Databases.List(s.env.ctx, s.server)
	if err != nil {
		return errMsg{err}
	}
	return databasesLoadedMsg{server: s.server, dbs: dbs}
}

func (s *databasesScreen) groups() []actionGroup {
	general := actionGroup{title: "Server " + s.server, actions: []action{
		{key: "n", label: "New project…", desc: "Create a database and its roles from a template.",
			open: func() screen { return newProjectForm(s.env, s.server) }},
		{key: "a", label: "Apply a project.yaml…", desc: "Make the server match a declarative spec file.",
			open: func() screen { return applySpecForm(s.env, s.server) }},
		{key: "R", label: "All roles on this server", desc: "Create, drop and manage every role.",
			open: func() screen { return newRolesScreen(s.env, s.server) }},
	}}
	if len(s.dbs) == 0 {
		return []actionGroup{general}
	}
	db := s.dbs[s.cur.pos]
	return []actionGroup{{title: db.Name, actions: []action{
		{key: "right", label: "Open", desc: "See who can access " + db.Name + " and manage its roles.",
			open: func() screen { return newDetailScreen(s.env, s.server, db.Name) }},
		{key: "c", label: "Clone…", desc: "Copy " + db.Name + " to a new database (e.g. for staging).",
			open: func() screen { return cloneForm(s.env, s.server, db) }},
		{key: "x", label: "Drop…", desc: "Delete " + db.Name + ", optionally with a backup and its project's roles.",
			open: func() screen { return dropDatabaseForm(s.env, s.server, db) }},
	}}, general}
}

func (s *databasesScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case databasesLoadedMsg:
		if msg.server == s.server {
			s.loading, s.err, s.dbs = false, nil, msg.dbs
			s.cur.clamp(len(s.dbs))
		}
	case errMsg:
		s.loading, s.err = false, msg.err
	case resumeMsg:
		return s, s.load
	case tea.KeyMsg:
		key := msg.String()
		if s.cur.move(key, len(s.dbs)) {
			return s, nil
		}
		switch key {
		case "esc", "q", "left":
			return s, pop
		case "r":
			s.loading = true
			return s, s.load
		}
		title := s.server
		if len(s.dbs) > 0 {
			title = s.dbs[s.cur.pos].Name
		}
		if cmd, ok := handleListKey(s.env, key, title, s.groups()); ok {
			return s, cmd
		}
	}
	return s, nil
}

func (s *databasesScreen) View() string {
	var b strings.Builder
	switch {
	case s.loading && s.dbs == nil:
		b.WriteString(s.env.loading("Loading databases…"))
	case len(s.dbs) == 0:
		b.WriteString("No databases yet. Press " + keyStyle.Render("n") + " to create a project.\n")
	default:
		rows := make([][]string, len(s.dbs))
		for i, d := range s.dbs {
			rows[i] = []string{d.Name, d.Owner, humanize.Bytes(d.SizeBytes), orDash(d.Project()), strconv.Itoa(d.Connections)}
		}
		t := table{
			columns: []column{
				{title: "DATABASE"}, {title: "OWNER", width: 20}, {title: "SIZE", width: 9, right: true},
				{title: "PROJECT", width: 16}, {title: "CONNS", width: 5, right: true},
			},
			rows: rows,
			rowStyle: func(i int) lipgloss.Style {
				if s.dbs[i].Project() == "" {
					return faintStyle
				}
				return lipgloss.NewStyle()
			},
		}
		b.WriteString(t.render(s.cur.pos, s.env.bodyWidth(), s.env.bodyHeight()-2))
		b.WriteString("\n" + faintStyle.Render("dimmed = not managed by launch-pg"))
	}
	if s.err != nil {
		b.WriteString("\n\n" + errorLine(s.err))
	}
	return b.String()
}

func (s *databasesScreen) Help() string {
	return hint("enter", "actions", "→", "open", "n", "new project", "?", "help", "esc", "back")
}
