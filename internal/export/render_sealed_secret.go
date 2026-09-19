package export

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"launch-pg/internal/model"
)

// Sealing scopes, as in kubeseal --scope.
const (
	ScopeStrict        = "strict"
	ScopeNamespaceWide = "namespace-wide"
	ScopeClusterWide   = "cluster-wide"
)

const (
	OptCert  = "cert"
	OptScope = "scope"
)

// SealedSecretRenderer writes Bitnami SealedSecrets, encrypted offline with
// the controller's public certificate. Safe to commit to git.
type SealedSecretRenderer struct {
	random io.Reader
}

// NewSealedSecretRenderer takes the randomness source (crypto/rand.Reader).
func NewSealedSecretRenderer(random io.Reader) SealedSecretRenderer {
	return SealedSecretRenderer{random: random}
}

func (SealedSecretRenderer) Kind() string        { return "sealedsecret" }
func (SealedSecretRenderer) NeedsPassword() bool { return true }

func (SealedSecretRenderer) Check(opts Options) error {
	if opts[OptCert] == "" {
		return errors.New("cert= (path to the controller cert from `kubeseal --fetch-cert`) is required")
	}
	if _, err := loadSealingKey(opts[OptCert]); err != nil {
		return err
	}
	switch scope := opts.Get(OptScope, ScopeStrict); scope {
	case ScopeStrict, ScopeNamespaceWide:
		if opts[OptNamespace] == "" {
			return fmt.Errorf("namespace= is required for scope %s", scope)
		}
	case ScopeClusterWide:
	default:
		return fmt.Errorf("unknown scope %q (strict, namespace-wide, cluster-wide)", scope)
	}
	return checkK8sOptions(opts)
}

type sealedSecret struct {
	APIVersion string           `yaml:"apiVersion"`
	Kind       string           `yaml:"kind"`
	Metadata   objectMeta       `yaml:"metadata"`
	Spec       sealedSecretSpec `yaml:"spec"`
}

type sealedSecretSpec struct {
	EncryptedData map[string]string `yaml:"encryptedData"`
	Template      sealedTemplate    `yaml:"template"`
}

type sealedTemplate struct {
	Metadata objectMeta `yaml:"metadata"`
	Type     string     `yaml:"type"`
}

func (r SealedSecretRenderer) Render(creds []model.Credential, opts Options) ([]byte, error) {
	if err := uniqueNames(creds, opts); err != nil {
		return nil, err
	}
	pub, err := loadSealingKey(opts[OptCert])
	if err != nil {
		return nil, err
	}
	scope := opts.Get(OptScope, ScopeStrict)

	objects := make([]any, 0, len(creds))
	for _, c := range creds {
		m, err := meta(c, opts)
		if err != nil {
			return nil, err
		}
		m.Annotations = scopeAnnotations(scope)
		label := []byte(sealingLabel(scope, m.Namespace, m.Name))

		encrypted := map[string]string{}
		for _, f := range Fields(c) {
			ct, err := hybridEncrypt(r.random, pub, []byte(f.Value), label)
			if err != nil {
				return nil, fmt.Errorf("seal %s: %w", f.Key, err)
			}
			encrypted[f.Key] = base64.StdEncoding.EncodeToString(ct)
		}

		objects = append(objects, sealedSecret{
			APIVersion: "bitnami.com/v1alpha1",
			Kind:       "SealedSecret",
			Metadata:   m,
			Spec: sealedSecretSpec{
				EncryptedData: encrypted,
				Template:      sealedTemplate{Metadata: m, Type: "Opaque"},
			},
		})
	}
	return yamlDocs(objects)
}

// sealingLabel binds ciphertext to where it may be decrypted.
func sealingLabel(scope, namespace, name string) string {
	switch scope {
	case ScopeClusterWide:
		return ""
	case ScopeNamespaceWide:
		return namespace
	default:
		return namespace + "/" + name
	}
}

func scopeAnnotations(scope string) map[string]string {
	switch scope {
	case ScopeClusterWide:
		return map[string]string{"sealedsecrets.bitnami.com/cluster-wide": "true"}
	case ScopeNamespaceWide:
		return map[string]string{"sealedsecrets.bitnami.com/namespace-wide": "true"}
	default:
		return nil
	}
}
