package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"launch-pg/internal/config"
	"launch-pg/internal/model"
)

type serversLoadedMsg struct {
	servers     []config.Server
	defaultName string
}

// serversScreen is the root: pick and manage servers.
type serversScreen struct {
	env         *env
	servers     []config.Server
	defaultName string
	cur         cursor
	loading     bool
	err         error
}

func newServersScreen(e *env) *serversScreen {
	return &serversScreen{env: e, loading: true}
}

func (s *serversScreen) Title() string { return "servers" }
func (s *serversScreen) Init() tea.Cmd { return s.load }

func (s *serversScreen) load() tea.Msg {
	servers, def, err := s.env.svc.Servers.List()
	if err != nil {
		return errMsg{err}
	}
	return serversLoadedMsg{servers: servers, defaultName: def}
}

func (s *serversScreen) groups() []actionGroup {
	general := actionGroup{title: "General", actions: []action{
		{key: "a", label: "Add a server…", desc: "Connect launch-pg to a Postgres server.",
			open: func() screen { return addServerForm(s.env) }},
		{key: "T", label: "Browse templates", desc: "See what each project template creates.",
			open: func() screen { return newTemplatesScreen(s.env) }},
	}}
	if len(s.servers) == 0 {
		return []actionGroup{general}
	}
	name := s.servers[s.cur.pos].Name
	return []actionGroup{{title: name, actions: []action{
		{key: "right", label: "Open databases", desc: "Browse, create, clone and drop databases on " + name + ".",
			open: func() screen { return newDatabasesScreen(s.env, name) }},
		{key: "t", label: "Test connection", desc: "Check the connection and the admin role's privileges.",
			run: func() tea.Cmd { return s.test(name) }},
		{key: "s", label: "Set as default", desc: "Mark " + name + " as the default server.",
			run: func() tea.Cmd { return s.setDefault(name) }},
		{key: "x", label: "Remove…", desc: "Forget this server and its stored password. The server itself is untouched.",
			open: func() screen { return removeServerConfirm(s.env, name) }},
	}}, general}
}

func (s *serversScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case serversLoadedMsg:
		s.loading, s.err = false, nil
		s.servers, s.defaultName = msg.servers, msg.defaultName
		s.cur.clamp(len(s.servers))
	case showMsg:
		return s, push(newMessageScreen(s.env, msg.title, msg.body))
	case errMsg:
		s.loading, s.err = false, msg.err
	case resumeMsg:
		return s, s.load
	case tea.KeyMsg:
		key := msg.String()
		if s.cur.move(key, len(s.servers)) {
			return s, nil
		}
		switch key {
		case "q", "esc":
			return s, tea.Quit
		case "r":
			s.loading = true
			return s, s.load
		}
		if cmd, ok := handleListKey(s.env, key, s.menuTitle(), s.groups()); ok {
			return s, cmd
		}
	}
	return s, nil
}

func (s *serversScreen) menuTitle() string {
	if len(s.servers) == 0 {
		return "actions"
	}
	return s.servers[s.cur.pos].Name
}

func (s *serversScreen) test(name string) tea.Cmd {
	return func() tea.Msg {
		info, err := s.env.svc.Servers.Test(s.env.ctx, name)
		if err != nil {
			return errMsg{err}
		}
		return showMsg{title: "test " + name, body: serverInfoText(info)}
	}
}

func (s *serversScreen) setDefault(name string) tea.Cmd {
	return func() tea.Msg {
		if err := s.env.svc.Servers.SetDefault(name); err != nil {
			return errMsg{err}
		}
		return showMsg{title: "default server", body: okStyle.Render("✓ "+name+" is now the default server.") + "\n"}
	}
}

func serverInfoText(info model.ServerInfo) string {
	var b strings.Builder
	b.WriteString(okStyle.Render(fmt.Sprintf("✓ Connected to Postgres %s as %q", info.Version, info.CurrentUser)) + "\n\n")
	for _, p := range []struct {
		name string
		has  bool
	}{{"superuser", info.Superuser}, {"createdb", info.CreateDB}, {"createrole", info.CreateRole}} {
		mark := faintStyle.Render("–")
		if p.has {
			mark = okStyle.Render("✓")
		}
		fmt.Fprintf(&b, "  %s %s\n", mark, p.name)
	}
	for _, p := range info.Problems() {
		b.WriteString("\n" + errorLine(fmt.Errorf("%s", p)))
	}
	return b.String()
}

func (s *serversScreen) View() string {
	var b strings.Builder
	switch {
	case s.loading && s.servers == nil:
		b.WriteString(s.env.loading("Loading servers…"))
	case len(s.servers) == 0:
		b.WriteString(titleStyle.Render("Welcome to launch-pg") + "\n\n")
		b.WriteString("No servers yet. Press " + keyStyle.Render("a") + " to connect one,\n")
		b.WriteString("or " + keyStyle.Render("enter") + " to see everything you can do.\n")
	default:
		rows := make([][]string, len(s.servers))
		for i, srv := range s.servers {
			name := srv.Name
			if srv.Name == s.defaultName {
				name += " ★"
			}
			rows[i] = []string{name, fmt.Sprintf("%s:%d", srv.Host, srv.Port), srv.User, srv.CredentialStore}
		}
		t := table{columns: []column{
			{title: "NAME", width: 20}, {title: "ADDRESS"}, {title: "ADMIN", width: 16}, {title: "PASSWORD IN", width: 12},
		}, rows: rows}
		b.WriteString(t.render(s.cur.pos, s.env.bodyWidth(), s.env.bodyHeight()-2))
		b.WriteString("\n" + faintStyle.Render("★ default server"))
	}
	if s.err != nil {
		b.WriteString("\n\n" + errorLine(s.err))
	}
	return b.String()
}

func (s *serversScreen) Help() string {
	return hint("enter", "actions", "→", "open", "a", "add", "?", "help", "q", "quit")
}
