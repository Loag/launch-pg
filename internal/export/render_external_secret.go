package export

import (
	"errors"

	"launch-pg/internal/model"
)

// ExternalSecretRenderer writes External Secrets Operator manifests that
// pull each credential from a secret store. It contains no secret material,
// so it never needs a password; pair it with a pusher (vault, aws) using
// the same key.
type ExternalSecretRenderer struct{}

const (
	OptStore      = "store"
	OptStoreKind  = "storeKind"
	OptKey        = "key"
	OptAPIVersion = "apiVersion"
	OptRefresh    = "refresh"
)

func (ExternalSecretRenderer) Kind() string        { return "externalsecret" }
func (ExternalSecretRenderer) NeedsPassword() bool { return false }

func (ExternalSecretRenderer) Check(opts Options) error {
	if opts[OptStore] == "" {
		return errors.New("store= (the SecretStore/ClusterSecretStore name) is required")
	}
	return checkK8sOptions(opts)
}

type externalSecret struct {
	APIVersion string             `yaml:"apiVersion"`
	Kind       string             `yaml:"kind"`
	Metadata   objectMeta         `yaml:"metadata"`
	Spec       externalSecretSpec `yaml:"spec"`
}

type externalSecretSpec struct {
	RefreshInterval string         `yaml:"refreshInterval"`
	SecretStoreRef  storeRef       `yaml:"secretStoreRef"`
	Target          esTarget       `yaml:"target"`
	DataFrom        []esDataSource `yaml:"dataFrom"`
}

type storeRef struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"`
}

type esTarget struct {
	Name           string `yaml:"name"`
	CreationPolicy string `yaml:"creationPolicy"`
}

type esDataSource struct {
	Extract esExtract `yaml:"extract"`
}

type esExtract struct {
	Key string `yaml:"key"`
}

func (ExternalSecretRenderer) Render(creds []model.Credential, opts Options) ([]byte, error) {
	if err := uniqueNames(creds, opts); err != nil {
		return nil, err
	}
	objects := make([]any, 0, len(creds))
	for _, c := range creds {
		m, err := meta(c, opts)
		if err != nil {
			return nil, err
		}
		objects = append(objects, externalSecret{
			APIVersion: opts.Get(OptAPIVersion, "external-secrets.io/v1"),
			Kind:       "ExternalSecret",
			Metadata:   m,
			Spec: externalSecretSpec{
				RefreshInterval: opts.Get(OptRefresh, "1h"),
				SecretStoreRef:  storeRef{Name: opts[OptStore], Kind: opts.Get(OptStoreKind, "ClusterSecretStore")},
				Target:          esTarget{Name: m.Name, CreationPolicy: "Owner"},
				DataFrom:        []esDataSource{{Extract: esExtract{Key: Expand(opts.Get(OptKey, DefaultPath), c)}}},
			},
		})
	}
	return yamlDocs(objects)
}
