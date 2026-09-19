package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"launch-pg/internal/config"
	"launch-pg/internal/credstore"
)

var sslModes = []option{
	{"disable", "disable", "No TLS."},
	{"allow", "allow", "TLS only if the server insists."},
	{"prefer", "prefer", "TLS when available (Postgres default)."},
	{"require", "require", "Always TLS, without verifying the certificate."},
	{"verify-ca", "verify-ca", "TLS and verify the certificate authority."},
	{"verify-full", "verify-full", "TLS, verify the CA and the hostname."},
}

var credentialStores = []option{
	{credstore.KindKeychain, "OS keychain", "Stored in macOS Keychain / Linux Secret Service. Recommended."},
	{credstore.KindFile, "File", "Stored in ~/.config/launch-pg/pgpass with 0600 permissions."},
	{credstore.KindEnv, "Environment variable", "Nothing stored; read from LAUNCHPG_<NAME>_PASSWORD each run."},
}

// addServerForm registers a server, stores its admin password, and tests it.
func addServerForm(e *env) screen {
	return newForm(e, "add server",
		"Connect launch-pg to a Postgres server. The admin role needs CREATEDB and CREATEROLE; superuser isn't required.",
		"Add server",
		[]field{
			textInput("Name", "", "home", "A short name for this server."),
			textInput("Host", "localhost", "", ""),
			textInput("Port", strconv.Itoa(config.DefaultPort), "", ""),
			textInput("Admin role", "", "postgres", ""),
			secretInput("Password", "Leave empty only when using the environment-variable store."),
			choice("SSL mode", sslModes, config.DefaultSSLMode, ""),
			textInput("Maintenance database", config.DefaultMaintenanceDB, "", "The database launch-pg connects to for server-wide changes."),
			choice("Where to keep the password", credentialStores, credstore.KindKeychain, ""),
			toggle("Make this the default server", false, ""),
		},
		func(v []string) tea.Cmd {
			port, err := strconv.Atoi(v[2])
			if err != nil {
				return errCmd(fmt.Errorf("port must be a number"))
			}
			server := config.Server{
				Name: v[0], Host: v[1], Port: port, User: v[3],
				SSLMode: v[5], MaintenanceDB: v[6], CredentialStore: v[7],
			}
			password, makeDefault := v[4], isYes(v[8])
			return func() tea.Msg {
				if err := e.svc.Servers.Add(server, password, makeDefault); err != nil {
					return errMsg{err}
				}
				return showMsg{title: "add server", body: addedServerReport(e, server)}
			}
		})
}

// addedServerReport tests the new server; a failed test still keeps it.
func addedServerReport(e *env, server config.Server) string {
	var b strings.Builder
	b.WriteString(okStyle.Render(fmt.Sprintf("✓ Added server %q.", server.Name)) + "\n")
	if server.CredentialStore == credstore.KindEnv {
		fmt.Fprintf(&b, "Set $%s before connecting (then restart launch-pg).\n", credstore.VarName(server.Name))
	}
	b.WriteString("\n")
	info, err := e.svc.Servers.Test(e.ctx, server.Name)
	if err != nil {
		b.WriteString(warnStyle.Render("⚠ Connection test failed: "+err.Error()) + "\n")
		b.WriteString("The server is saved. Fix the problem, then choose \"Test connection\" on it.\n")
		return b.String()
	}
	b.WriteString(serverInfoText(info))
	return b.String()
}

func removeServerConfirm(e *env, name string) screen {
	return newConfirm(e, "remove "+name,
		fmt.Sprintf("Remove server %q from launch-pg and delete its stored admin password?\n"+
			"Nothing on the Postgres server itself is changed.", name),
		func() error { return e.svc.Servers.Remove(name) })
}
