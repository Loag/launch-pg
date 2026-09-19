package export

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVaultPusherKV2(t *testing.T) {
	var (
		gotPath  string
		gotToken string
		gotBody  map[string]map[string]string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotToken = r.URL.Path, r.Header.Get("X-Vault-Token")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	env := map[string]string{"VAULT_ADDR": srv.URL, "VAULT_TOKEN": "tok"}
	p := NewVaultPusher(srv.Client(), func(k string) (string, bool) { v, ok := env[k]; return v, ok })
	if err := p.Check(Options{}); err != nil {
		t.Fatal(err)
	}

	loc, err := p.Push(context.Background(), creds("myapp_app")[0], Options{})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/secret/data/launch-pg/myapp/myapp_app" {
		t.Errorf("path = %s", gotPath)
	}
	if gotToken != "tok" || gotBody["data"]["password"] != "PWMYAPP_APP" {
		t.Errorf("token = %q body = %v", gotToken, gotBody)
	}
	if loc != "vault:secret/launch-pg/myapp/myapp_app" {
		t.Errorf("location = %s", loc)
	}
}

func TestVaultPusherReportsErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"errors":["permission denied"]}`, http.StatusForbidden)
	}))
	defer srv.Close()

	env := map[string]string{"VAULT_TOKEN": "tok"}
	p := NewVaultPusher(srv.Client(), func(k string) (string, bool) { v, ok := env[k]; return v, ok })
	if _, err := p.Push(context.Background(), creds("r")[0], Options{"addr": srv.URL}); err == nil {
		t.Error("expected error on 403")
	}
}
