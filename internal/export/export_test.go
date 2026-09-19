package export

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"launch-pg/internal/model"
)

func creds(roles ...string) []model.Credential {
	out := make([]model.Credential, len(roles))
	for i, r := range roles {
		out[i] = model.Credential{
			Host: "db.local", Port: 5432, Database: "myapp",
			User: r, Password: "PW" + strings.ToUpper(r), SSLMode: "require",
		}
	}
	return out
}

func testExporter() *Exporter {
	return NewExporter(
		[]Renderer{NewEnvRenderer("env"), JSONRenderer{}, K8sSecretRenderer{}, ExternalSecretRenderer{}},
		nil,
	)
}

func TestParseTarget(t *testing.T) {
	tgt, err := ParseTarget("k8s:namespace=apps,labels=a=b|c=d")
	if err != nil {
		t.Fatal(err)
	}
	if tgt.Kind != "k8s" || tgt.Options["namespace"] != "apps" || tgt.Options["labels"] != "a=b|c=d" {
		t.Errorf("parsed = %+v", tgt)
	}
	for _, bad := range []string{"", ":x=1", "k8s:novalue", "k8s:a=1,a=2"} {
		if _, err := ParseTarget(bad); err == nil {
			t.Errorf("ParseTarget(%q) expected error", bad)
		}
	}
}

func TestExporterParseValidatesEarly(t *testing.T) {
	e := testExporter()
	if _, err := e.Parse([]string{"nope"}); err == nil {
		t.Error("expected unknown kind error")
	}
	if _, err := e.Parse([]string{"externalsecret"}); err == nil {
		t.Error("expected missing store= error")
	}
	if _, err := e.Parse([]string{"env:port=abc"}); err == nil {
		t.Error("expected bad port error")
	}
}

func TestNeedsPassword(t *testing.T) {
	e := testExporter()
	only, _ := e.Parse([]string{"externalsecret:store=vault"})
	if e.NeedsPassword(only) {
		t.Error("externalsecret alone should not need a password")
	}
	both, _ := e.Parse([]string{"externalsecret:store=vault", "env"})
	if !e.NeedsPassword(both) {
		t.Error("env needs a password")
	}
}

func TestEnvPrefixesMultipleRoles(t *testing.T) {
	out, err := NewEnvRenderer("env").Render(creds("myapp_app", "myapp_ro"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{"MYAPP_APP_PGPASSWORD=PWMYAPP_APP", "MYAPP_RO_PGUSER=myapp_ro",
		"MYAPP_APP_DATABASE_URL=postgres://myapp_app:PWMYAPP_APP@db.local:5432/myapp?sslmode=require"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %s in:\n%s", want, s)
		}
	}

	single, _ := NewEnvRenderer("env").Render(creds("myapp_app"), Options{})
	if !strings.HasPrefix(string(single), "DATABASE_URL=") {
		t.Errorf("single credential should be unprefixed:\n%s", single)
	}
}

func TestEnvQuoting(t *testing.T) {
	if got := envQuote(`a b"$`); got != `"a b\"\$"` {
		t.Errorf("envQuote = %s", got)
	}
}

func TestK8sSecretManifest(t *testing.T) {
	out, err := K8sSecretRenderer{}.Render(creds("myapp_app", "myapp_ro"), Options{"namespace": "apps"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{
		"kind: Secret", "name: myapp-app-postgres", "namespace: apps",
		"app.kubernetes.io/managed-by: launch-pg", "password: PWMYAPP_APP", "port: \"5432\"", "---",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

func TestK8sRejectsCollidingNames(t *testing.T) {
	_, err := K8sSecretRenderer{}.Render(creds("a", "b"), Options{"name": "fixed"})
	if err == nil {
		t.Error("expected name collision error")
	}
}

func TestExternalSecretHasNoSecretMaterial(t *testing.T) {
	out, err := ExternalSecretRenderer{}.Render(creds("myapp_app"), Options{"store": "vault-backend"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(s, "PWMYAPP_APP") {
		t.Error("ExternalSecret must not contain the password")
	}
	for _, want := range []string{"kind: ExternalSecret", "key: launch-pg/myapp/myapp_app", "kind: ClusterSecretStore"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

func TestExportWritesFilesPrivatelyAndAppliesOverrides(t *testing.T) {
	e := testExporter()
	path := filepath.Join(t.TempDir(), "out", "db.json")
	targets, err := e.Parse([]string{"json:out=" + path + ",host=pg.svc.cluster.local"})
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	reports, err := e.Export(context.Background(), &stdout, creds("myapp_app"), targets)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 || reports[0].Location != path || stdout.Len() != 0 {
		t.Errorf("reports = %+v stdout = %q", reports, stdout.String())
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %o", info.Mode().Perm())
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `"host": "pg.svc.cluster.local"`) {
		t.Errorf("host override not applied:\n%s", data)
	}
}

func TestK8sName(t *testing.T) {
	if got := K8sName("MyApp_app-postgres"); got != "myapp-app-postgres" {
		t.Errorf("K8sName = %q", got)
	}
}
