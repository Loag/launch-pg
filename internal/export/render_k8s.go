package export

import (
	"launch-pg/internal/model"
)

// K8sSecretRenderer writes one Opaque Secret per credential using
// stringData (Kubernetes base64-encodes it on apply).
type K8sSecretRenderer struct{}

func (K8sSecretRenderer) Kind() string        { return "k8s" }
func (K8sSecretRenderer) NeedsPassword() bool { return true }
func (K8sSecretRenderer) Check(opts Options) error {
	return checkK8sOptions(opts)
}

type k8sSecret struct {
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   objectMeta        `yaml:"metadata"`
	Type       string            `yaml:"type"`
	StringData map[string]string `yaml:"stringData"`
}

func (K8sSecretRenderer) Render(creds []model.Credential, opts Options) ([]byte, error) {
	if err := uniqueNames(creds, opts); err != nil {
		return nil, err
	}
	objects := make([]any, 0, len(creds))
	for _, c := range creds {
		m, err := meta(c, opts)
		if err != nil {
			return nil, err
		}
		objects = append(objects, k8sSecret{
			APIVersion: "v1",
			Kind:       "Secret",
			Metadata:   m,
			Type:       "Opaque",
			StringData: FieldMap(c),
		})
	}
	return yamlDocs(objects)
}
