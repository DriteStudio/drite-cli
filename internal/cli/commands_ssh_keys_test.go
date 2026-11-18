package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/DriteStudio/drite-cli/drite"
)

const sshKeyAPIPath = "/api/auth/me/ssh-keys"

func TestSSHKeyLibraryCommands(t *testing.T) {
	setConfigHome(t)
	const publicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8g laptop"
	file := filepath.Join(t.TempDir(), "laptop.pub")
	if err := os.WriteFile(file, []byte(publicKey+"\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name     string
		args     []string
		method   string
		endpoint string
		status   int
		output   string
	}{
		{"list", []string{"list"}, http.MethodGet, sshKeyAPIPath, http.StatusOK, `{"data":[]}`},
		{"add", []string{"add", "--name", "Laptop", "--file", file}, http.MethodPost, sshKeyAPIPath, http.StatusCreated, `{"data":{"id":"key-1"}}`},
		{"delete", []string{"delete", "key /1"}, http.MethodDelete, sshKeyAPIPath + "/key%20%2F1", http.StatusNoContent, "{}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != tc.method || r.URL.EscapedPath() != tc.endpoint || r.Header.Get("Authorization") != "Bearer dr_live_fixture" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.EscapedPath())
				}
				if r.Method == http.MethodPost {
					var body drite.AddSSHKeyRequest
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name != "Laptop" || body.PublicKey != publicKey {
						t.Error("the POST body did not contain the requested public key and name", err)
					}
				}
				w.WriteHeader(tc.status)
				if tc.status != http.StatusNoContent {
					io.WriteString(w, tc.output)
				}
			}))
			defer server.Close()

			var stdout, stderr bytes.Buffer
			app := NewApp(strings.NewReader(""), &stdout, &stderr)
			args := append([]string{"--base-url", server.URL, "--token", "dr_live_fixture", "--compact", "ssh-key"}, tc.args...)
			if code := app.Run(context.Background(), args); code != 0 {
				t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
			}
			if got := strings.TrimSpace(stdout.String()); got != tc.output || requests.Load() != 1 {
				t.Fatalf("stdout = %q, requests = %d", got, requests.Load())
			}
		})
	}
}

func TestSSHKeyAddRejectsPrivateKeyBeforeSending(t *testing.T) {
	setConfigHome(t)
	file := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(file, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nfixture\n-----END OPENSSH PRIVATE KEY-----\n"), 0600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("a private key file must never be sent")
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	var stderr bytes.Buffer
	app := NewApp(strings.NewReader(""), io.Discard, &stderr)
	args := []string{"--base-url", server.URL, "--token", "dr_live_fixture", "ssh-key", "add", "--name", "Laptop", "--file", file}
	if code := app.Run(context.Background(), args); code != 1 || !strings.Contains(stderr.String(), "use the .pub file") {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
}

func TestVPSOrdersResolveSavedSSHKeySelectors(t *testing.T) {
	setConfigHome(t)
	cases := []struct {
		args []string
		path string
	}{
		{[]string{"vps", "create"}, "/api/auth/vps"},
		{[]string{"vps", "custom-create"}, "/api/auth/vps/custom"},
		{[]string{"vps", "reinstall", "vm"}, "/api/auth/vps/vm/reinstall"},
		{[]string{"reseller", "vps", "create"}, "/api/reseller/vps"},
		{[]string{"reseller", "vps", "custom-create"}, "/api/reseller/vps/custom"},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			var reads, orders atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet && r.URL.Path == sshKeyAPIPath {
					reads.Add(1)
					io.WriteString(w, `{"data":[{"id":"key-1","name":"Laptop"},{"id":"key-2","name":"CI"}]}`)
					return
				}
				orders.Add(1)
				if r.Method != http.MethodPost || r.URL.Path != tc.path {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				var body struct {
					KeyIDs     []string `json:"sshKeyIds"`
					PublicKeys []string `json:"sshPublicKeys"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil ||
					!reflect.DeepEqual(body.KeyIDs, []string{"key-direct", "key-1", "key-2"}) ||
					!reflect.DeepEqual(body.PublicKeys, []string{"inline-key"}) {
					t.Errorf("order payload = %#v, error = %v", body, err)
				}
				io.WriteString(w, "{}")
			}))
			defer server.Close()

			var stderr bytes.Buffer
			app := NewApp(strings.NewReader(""), io.Discard, &stderr)
			args := append([]string{"--base-url", server.URL, "--token", "dr_live_fixture"}, tc.args...)
			args = append(args, "--data", `{"sshKeyIds":["key-direct","key-1"],"sshPublicKeys":["inline-key"]}`, "--ssh-key", "laptop", "--ssh-key", "key-2")
			if code := app.Run(context.Background(), args); code != 0 || reads.Load() != 1 || orders.Load() != 1 {
				t.Fatalf("exit = %d, reads = %d, orders = %d, stderr = %q", code, reads.Load(), orders.Load(), stderr.String())
			}
		})
	}
}

func TestVPSOrderWithUnknownSSHKeyDoesNotSubmit(t *testing.T) {
	setConfigHome(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != sshKeyAPIPath {
			t.Errorf("an unknown key reached %s %s", r.Method, r.URL.Path)
		}
		io.WriteString(w, `{"data":[]}`)
	}))
	defer server.Close()

	var stderr bytes.Buffer
	app := NewApp(strings.NewReader(""), io.Discard, &stderr)
	args := []string{"--base-url", server.URL, "--token", "dr_live_fixture", "vps", "create", "--data", "{}", "--ssh-key", "unknown"}
	if code := app.Run(context.Background(), args); code != 1 || !strings.Contains(stderr.String(), "was not found") {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
}

func TestVPSReplaceWithSavedSSHKeyWithoutJSONBody(t *testing.T) {
	setConfigHome(t)
	var replacements atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == sshKeyAPIPath:
			io.WriteString(w, `{"data":[{"id":"key-1","name":"Laptop"}]}`)
		case r.Method == http.MethodPut && r.URL.Path == "/api/auth/vps/vm/ssh-keys":
			replacements.Add(1)
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil ||
				!reflect.DeepEqual(body, map[string]any{"sshKeyIds": []any{"key-1"}}) {
				t.Errorf("replacement payload = %#v, error = %v", body, err)
			}
			io.WriteString(w, `{"rebootRequired":true}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	var stderr bytes.Buffer
	app := NewApp(strings.NewReader(""), io.Discard, &stderr)
	args := []string{"--base-url", server.URL, "--token", "dr_live_fixture", "vps", "ssh-keys", "vm", "--ssh-key", "Laptop"}
	if code := app.Run(context.Background(), args); code != 0 || replacements.Load() != 1 {
		t.Fatalf("exit = %d, replacements = %d, stderr = %q", code, replacements.Load(), stderr.String())
	}
}
