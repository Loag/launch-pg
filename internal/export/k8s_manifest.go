package export

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"launch-pg/internal/model"
)

// Options shared by the Kubernetes renderers.
const (
	OptName      = "name"
	OptNamespace = "namespace"
	OptLabels    = "labels" // "k=v|k2=v2"
)

const defaultK8sName = "{role}-postgres"

type objectMeta struct {
	Name        string            `yaml:"name"`
	Namespace   string            `yaml:"namespace,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// meta builds metadata for a credential's object.
func meta(c model.Credential, opts Options) (objectMeta, error) {
	labels, err := parseLabels(opts[OptLabels])
	if err != nil {
		return objectMeta{}, err
	}
	labels["app.kubernetes.io/managed-by"] = "launch-pg"
	return objectMeta{
		Name:      K8sName(Expand(opts.Get(OptName, defaultK8sName), c)),
		Namespace: opts[OptNamespace],
		Labels:    labels,
	}, nil
}

func parseLabels(s string) (map[string]string, error) {
	labels := map[string]string{}
	if s == "" {
		return labels, nil
	}
	for _, pair := range strings.Split(s, "|") {
		k, v, ok := strings.Cut(pair, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("label %q: expected key=value (separate labels with |)", pair)
		}
		labels[k] = v
	}
	return labels, nil
}

// checkK8sOptions validates options shared by the Kubernetes renderers.
func checkK8sOptions(opts Options) error {
	_, err := parseLabels(opts[OptLabels])
	return err
}

// yamlDocs marshals objects as a multi-document YAML stream.
func yamlDocs(objects []any) ([]byte, error) {
	var buf bytes.Buffer
	for i, obj := range objects {
		if i > 0 {
			buf.WriteString("---\n")
		}
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		if err := enc.Encode(obj); err != nil {
			return nil, err
		}
		if err := enc.Close(); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// uniqueNames fails if several credentials render to the same object name
// (e.g. name=db-creds without {role} while exporting a whole project).
func uniqueNames(creds []model.Credential, opts Options) error {
	seen := map[string]string{}
	for _, c := range creds {
		name := K8sName(Expand(opts.Get(OptName, defaultK8sName), c))
		if other, dup := seen[name]; dup {
			return fmt.Errorf("roles %s and %s both render to name %q; include {role} in name=", other, c.User, name)
		}
		seen[name] = c.User
	}
	return nil
}
