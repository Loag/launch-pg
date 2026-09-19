package export

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"launch-pg/internal/model"
)

// LookupEnv matches os.LookupEnv.
type LookupEnv func(string) (string, bool)

// VaultPusher writes to a HashiCorp Vault KV engine over its HTTP API.
// Address and token come from options or VAULT_ADDR / VAULT_TOKEN;
// VAULT_NAMESPACE is honored for Vault Enterprise / HCP.
type VaultPusher struct {
	client *http.Client
	env    LookupEnv
}

func NewVaultPusher(client *http.Client, env LookupEnv) *VaultPusher {
	return &VaultPusher{client: client, env: env}
}

const (
	OptAddr  = "addr"
	OptMount = "mount"
	OptPath  = "path"
	OptKV    = "kv"
)

func (*VaultPusher) Kind() string { return "vault" }

func (v *VaultPusher) Check(opts Options) error {
	if v.addr(opts) == "" {
		return errors.New("vault address missing: set addr= or $VAULT_ADDR")
	}
	if _, ok := v.env("VAULT_TOKEN"); !ok {
		return errors.New("$VAULT_TOKEN is not set")
	}
	if kv := opts.Get(OptKV, "2"); kv != "1" && kv != "2" {
		return fmt.Errorf("kv=%s: must be 1 or 2", kv)
	}
	return nil
}

func (v *VaultPusher) Push(ctx context.Context, c model.Credential, opts Options) (string, error) {
	mount := strings.Trim(opts.Get(OptMount, "secret"), "/")
	path := strings.Trim(Expand(opts.Get(OptPath, DefaultPath), c), "/")

	var (
		url  string
		body any
	)
	if opts.Get(OptKV, "2") == "2" {
		url = fmt.Sprintf("%s/v1/%s/data/%s", v.addr(opts), mount, path)
		body = map[string]any{"data": FieldMap(c)}
	} else {
		url = fmt.Sprintf("%s/v1/%s/%s", v.addr(opts), mount, path)
		body = FieldMap(c)
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	token, _ := v.env("VAULT_TOKEN")
	req.Header.Set("X-Vault-Token", token)
	req.Header.Set("Content-Type", "application/json")
	if ns, ok := v.env("VAULT_NAMESPACE"); ok && ns != "" {
		req.Header.Set("X-Vault-Namespace", ns)
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("vault request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("vault returned %s: %s", resp.Status, strings.TrimSpace(string(msg)))
	}
	return "vault:" + mount + "/" + path, nil
}

func (v *VaultPusher) addr(opts Options) string {
	addr, _ := v.env("VAULT_ADDR")
	return strings.TrimRight(opts.Get(OptAddr, addr), "/")
}
