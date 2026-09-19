// Command launch-pg is the composition root: it builds every dependency and
// hands them to the TUI. No other package constructs its own collaborators.
package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"launch-pg/internal/app"
	"launch-pg/internal/audit"
	"launch-pg/internal/backup"
	"launch-pg/internal/config"
	"launch-pg/internal/credstore"
	"launch-pg/internal/export"
	"launch-pg/internal/introspect"
	"launch-pg/internal/password"
	"launch-pg/internal/pg"
	"launch-pg/internal/plan"
	"launch-pg/internal/planner"
	"launch-pg/internal/preset"
	"launch-pg/internal/service"
	"launch-pg/internal/spec"
	"launch-pg/internal/templates"
	"launch-pg/internal/tui"
)

// version is set at build time: -ldflags "-X main.version=v0.1.0"
var version = "dev"

const keychainService = "launch-pg"

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `launch-pg %s: provision Postgres databases, roles and permissions.

Usage: launch-pg [--version] [--help]

Runs an interactive terminal UI. Configuration lives in ~/.config/launch-pg/.
`, version)
	}
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}
	if flag.NArg() > 0 {
		flag.Usage()
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	services, err := buildServices()
	if err == nil {
		err = tui.Run(ctx, services)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func buildServices() (app.Services, error) {
	paths, err := config.DefaultPaths()
	if err != nil {
		return app.Services{}, err
	}

	// Configuration and credentials.
	configs := config.NewFileRepository(paths.ConfigFile())
	envStore := credstore.NewEnvStore(os.LookupEnv)
	stores := credstore.NewRegistry(map[string]credstore.CredentialStore{
		credstore.KindKeychain: credstore.NewKeychainStore(keychainService),
		credstore.KindFile:     credstore.NewFileStore(paths.PassFile()),
		credstore.KindEnv:      envStore,
	})
	resolver := credstore.NewResolver(envStore, stores)

	// Database access.
	connector := pg.NewConnector(resolver)
	catalog := introspect.NewCatalog()
	stateLoader := service.NewStateLoader(configs, connector, catalog)

	// Planning and applying.
	presets := preset.Default()
	issuer := password.NewIssuer(password.Random{}, password.NewSCRAM(rand.Reader))
	plans := planner.New(presets, issuer)
	applier := plan.NewApplier(connector, audit.NewJSONLFile(paths.AuditFile()), time.Now)
	templateCatalog := templates.NewCatalog(paths.TemplatesDir())
	dumper := backup.NewPgDump("pg_dump", resolver, backup.ExecRunner{})

	// Credential export.
	exporter := export.NewExporter(
		[]export.Renderer{
			export.NewEnvRenderer("env"),
			export.NewEnvRenderer("compose"),
			export.JSONRenderer{},
			export.K8sSecretRenderer{},
			export.NewSealedSecretRenderer(rand.Reader),
			export.ExternalSecretRenderer{},
		},
		[]export.Pusher{
			export.NewVaultPusher(&http.Client{Timeout: 30 * time.Second}, os.LookupEnv),
			export.NewAWSSecretsPusher(export.DefaultSecretsManagerFactory),
		},
	)

	return app.Services{
		Servers:   service.NewServerService(configs, stores, connector, catalog),
		Projects:  service.NewProjectService(configs, stateLoader, templateCatalog, spec.NewResolver(templateCatalog), plans),
		Roles:     service.NewRoleService(stateLoader, plans),
		Databases: service.NewDatabaseService(stateLoader, plans, dumper, paths.BackupsDir(), time.Now),
		Applier:   service.NewApplyService(applier, dumper),
		Exporter:  exporter,
		Templates: templateCatalog,
		Presets:   presetInfo(presets),
		Version:   version,
	}, nil
}

func presetInfo(r *preset.Registry) []app.PresetInfo {
	var out []app.PresetInfo
	for _, name := range r.Names() {
		p, _ := r.Get(name)
		out = append(out, app.PresetInfo{Name: p.Name(), Description: p.Description()})
	}
	return out
}
