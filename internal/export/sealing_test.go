package export

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// hybridDecrypt mirrors sealed-secrets' controller-side decryption.
func hybridDecrypt(t *testing.T, key *rsa.PrivateKey, ciphertext, label []byte) []byte {
	t.Helper()
	n := binary.BigEndian.Uint16(ciphertext)
	rsaCT, aesCT := ciphertext[2:2+n], ciphertext[2+n:]
	sessionKey, err := rsa.DecryptOAEP(sha256.New(), nil, key, rsaCT, label)
	if err != nil {
		t.Fatalf("rsa decrypt: %v", err)
	}
	block, _ := aes.NewCipher(sessionKey)
	gcm, _ := cipher.NewGCM(block)
	plain, err := gcm.Open(nil, make([]byte, gcm.NonceSize()), aesCT, nil)
	if err != nil {
		t.Fatalf("gcm open: %v", err)
	}
	return plain
}

func writeTestCert(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "sealed-secret"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "cert.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	return key, path
}

func TestSealedSecretDecryptsWithStrictLabel(t *testing.T) {
	key, certPath := writeTestCert(t)
	r := NewSealedSecretRenderer(rand.Reader)
	opts := Options{"cert": certPath, "namespace": "apps"}
	if err := r.Check(opts); err != nil {
		t.Fatal(err)
	}

	out, err := r.Render(creds("myapp_app"), opts)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "PWMYAPP_APP") {
		t.Fatal("plaintext password in sealed output")
	}

	var doc sealedSecret
	if err := yaml.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	ct, err := base64.StdEncoding.DecodeString(doc.Spec.EncryptedData["password"])
	if err != nil {
		t.Fatal(err)
	}
	label := []byte("apps/" + doc.Metadata.Name)
	if got := string(hybridDecrypt(t, key, ct, label)); got != "PWMYAPP_APP" {
		t.Errorf("decrypted = %q", got)
	}
}

func TestSealedSecretCheck(t *testing.T) {
	_, certPath := writeTestCert(t)
	r := NewSealedSecretRenderer(rand.Reader)
	if err := r.Check(Options{"cert": certPath}); err == nil {
		t.Error("strict scope without namespace should fail")
	}
	if err := r.Check(Options{"cert": certPath, "scope": "cluster-wide"}); err != nil {
		t.Errorf("cluster-wide without namespace: %v", err)
	}
	if err := r.Check(Options{"namespace": "x"}); err == nil {
		t.Error("missing cert should fail")
	}
}
