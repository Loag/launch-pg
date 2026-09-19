package tui

import (
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// exportKind describes one export target for the guided builder.
type exportKind struct {
	kind   string
	label  string
	desc   string
	fields []exportField
}

// exportField is one option of a target (key=value in the target string).
type exportField struct {
	key         string
	label       string
	value       string
	placeholder string
	hint        string
	required    bool
	choices     []option
}

// Fields every target shares.
var (
	outField = exportField{key: "out", label: "Output file",
		placeholder: "leave empty to show it on screen", hint: "Written with 0600 permissions."}
	hostField = exportField{key: "host", label: "Host apps connect to (optional)",
		placeholder: "e.g. postgres.db.svc.cluster.local", hint: "Overrides the server's host in the credentials."}
	nameField = exportField{key: "name", label: "Kubernetes object name",
		placeholder: "{role}-postgres", hint: "{role} and {database} are replaced; needed when exporting several roles."}
	namespaceField = exportField{key: "namespace", label: "Namespace"}
)

var exportKinds = []exportKind{
	{kind: "k8s", label: "Kubernetes Secret", desc: "Opaque Secret with host, port, dbname, username, password, DATABASE_URL.",
		fields: []exportField{namespaceField, nameField, outField, hostField}},
	{kind: "sealedsecret", label: "Sealed Secret", desc: "Encrypted for the Sealed Secrets controller; safe to commit to git.",
		fields: []exportField{
			{key: "cert", label: "Controller certificate file", required: true,
				placeholder: "cert.pem", hint: "From `kubeseal --fetch-cert > cert.pem`."},
			{key: "namespace", label: "Namespace", hint: "Required unless the scope is cluster-wide."},
			{key: "scope", label: "Scope", choices: []option{
				{"strict", "strict", "Only decryptable with this exact name and namespace."},
				{"namespace-wide", "namespace-wide", "Can be renamed within the namespace."},
				{"cluster-wide", "cluster-wide", "Can be used in any namespace."},
			}},
			nameField, outField, hostField,
		}},
	{kind: "externalsecret", label: "External Secrets manifest", desc: "Points External Secrets Operator at a store. Contains no password, so nothing is rotated.",
		fields: []exportField{
			{key: "store", label: "Secret store name", required: true, placeholder: "vault-backend"},
			{key: "storeKind", label: "Store kind", choices: []option{
				{"ClusterSecretStore", "ClusterSecretStore", ""},
				{"SecretStore", "SecretStore", ""},
			}},
			namespaceField, nameField,
			{key: "key", label: "Key in the store", placeholder: "launch-pg/{database}/{role}",
				hint: "Matches the default Vault path / AWS name, so the pair works without changes."},
			outField,
		}},
	{kind: "env", label: ".env file", desc: "DATABASE_URL and PG* variables.",
		fields: []exportField{outField, {key: "prefix", label: "Variable prefix (optional)", placeholder: "APP_"}, hostField}},
	{kind: "compose", label: "docker compose env_file", desc: "Same as .env, for docker compose.",
		fields: []exportField{outField, hostField}},
	{kind: "json", label: "JSON", desc: "Array of credential objects.",
		fields: []exportField{outField, hostField}},
	{kind: "vault", label: "HashiCorp Vault", desc: "Writes a KV secret. Uses $VAULT_TOKEN (and $VAULT_ADDR if no address is given).",
		fields: []exportField{
			{key: "addr", label: "Vault address", placeholder: "$VAULT_ADDR"},
			{key: "mount", label: "KV mount", value: "secret"},
			{key: "path", label: "Path", placeholder: "launch-pg/{database}/{role}"},
			{key: "kv", label: "KV engine version", choices: []option{{"2", "v2", ""}, {"1", "v1", ""}}},
			hostField,
		}},
	{kind: "aws", label: "AWS Secrets Manager", desc: "Creates the secret or adds a new version. Uses your normal AWS credentials.",
		fields: []exportField{
			{key: "region", label: "Region", placeholder: "from AWS config"},
			{key: "profile", label: "Profile", placeholder: "default"},
			{key: "name", label: "Secret name", placeholder: "launch-pg/{database}/{role}"},
			hostField,
		}},
}

// exportMenu is step 1: choose where credentials go. onDone receives the
// finished target string and returns the command to run with it.
func exportMenu(e *env, title, database string, onDone func(target string) tea.Cmd) screen {
	var actions []action
	for _, k := range exportKinds {
		actions = append(actions, action{
			label: k.label, desc: k.desc,
			open: func() screen { return exportKindForm(e, k, database, onDone) },
		})
	}
	custom := action{
		label: "Custom target…", desc: "Type a target string, e.g. k8s:namespace=app,out=db.yaml.",
		open: func() screen {
			return newForm(e, "custom export", "", "", []field{
				textInput("Target", "", "kind:key=value,key=value", "Any option the kind supports."),
			}, func(v []string) tea.Cmd { return onDone(v[0]) })
		},
	}
	return newMenu(e, title, []actionGroup{
		{title: "Where should the credentials go?", actions: actions},
		{title: "Advanced", actions: []action{custom}},
	})
}

// exportKindForm is step 2: the options for one kind.
func exportKindForm(e *env, k exportKind, database string, onDone func(target string) tea.Cmd) screen {
	fields := make([]field, 0, len(k.fields)+1)
	for _, f := range k.fields {
		if f.choices != nil {
			fields = append(fields, choice(f.label, f.choices, f.choices[0].value, f.hint))
		} else {
			fields = append(fields, textInput(f.label, f.value, f.placeholder, f.hint))
		}
	}
	fields = append(fields, textInput("Database name in the credentials", database, "",
		"The database apps connect to."))

	return newForm(e, k.label, k.desc, "Export", fields, func(v []string) tea.Cmd {
		target, err := buildTarget(k, v[:len(k.fields)], v[len(k.fields)])
		if err != nil {
			return errCmd(err)
		}
		return onDone(target)
	})
}

// buildTarget turns form values into "kind:key=value,…", skipping empty
// optional values.
func buildTarget(k exportKind, values []string, database string) (string, error) {
	var opts []string
	for i, f := range k.fields {
		v := values[i]
		if v == "" {
			if f.required {
				return "", fmt.Errorf("%s is required", strings.ToLower(f.label))
			}
			continue
		}
		if strings.Contains(v, ",") {
			return "", errors.New("values can't contain commas")
		}
		opts = append(opts, f.key+"="+v)
	}
	if database != "" {
		opts = append(opts, "dbname="+database)
	}
	if len(opts) == 0 {
		return k.kind, nil
	}
	return k.kind + ":" + strings.Join(opts, ","), nil
}
