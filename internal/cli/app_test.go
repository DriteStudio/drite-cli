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
	"runtime"
	"strings"
	"testing"
)

func TestHostingUpgradeOptionsCommand(t *testing.T) {
	setConfigHome(t)
	var method, path, authorization string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		method = request.Method
		path = request.URL.EscapedPath()
		authorization = request.Header.Get("Authorization")
		writer.Header().Set("Content-Type", "application/json")
		io.WriteString(writer, `{"data":{"options":[]}}`)
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	app := NewApp(strings.NewReader(""), &stdout, &stderr)
	code := app.Run(context.Background(), []string{
		"--token", "token",
		"--base-url", server.URL,
		"hosting", "upgrade-options", "hosting /1",
	})
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if method != http.MethodGet || path != "/api/auth/hosting/hosting%20%2F1/upgrade-options" {
		t.Fatalf("request = %s %s", method, path)
	}
	if authorization != "Bearer token" {
		t.Fatalf("authorization = %q", authorization)
	}
	if !strings.Contains(stdout.String(), `"options": []`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestHostingUpgradeCommand(t *testing.T) {
	setConfigHome(t)
	var method, path string
	var body map[string]string
	var handlerErr error
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		method = request.Method
		path = request.URL.EscapedPath()
		handlerErr = json.NewDecoder(request.Body).Decode(&body)
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusAccepted)
		io.WriteString(writer, `{"jobId":"job-1"}`)
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	app := NewApp(strings.NewReader(""), &stdout, &stderr)
	code := app.Run(context.Background(), []string{
		"--token", "token",
		"--base-url", server.URL,
		"hosting", "upgrade", "hosting-1",
		"--data", `{"planId":"plan-2"}`,
	})
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if handlerErr != nil {
		t.Fatal(handlerErr)
	}
	if method != http.MethodPost || path != "/api/auth/hosting/hosting-1/upgrade" {
		t.Fatalf("request = %s %s", method, path)
	}
	if len(body) != 1 || body["planId"] != "plan-2" {
		t.Fatalf("body = %#v", body)
	}
	if !strings.Contains(stdout.String(), `"jobId": "job-1"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestHostingUpgradeRequiresIDAndBody(t *testing.T) {
	setConfigHome(t)
	tests := []struct {
		name      string
		arguments []string
		message   string
	}{
		{
			name:      "options ID",
			arguments: []string{"hosting", "upgrade-options"},
			message:   "hosting ID is required",
		},
		{
			name:      "upgrade ID",
			arguments: []string{"hosting", "upgrade"},
			message:   "hosting ID is required",
		},
		{
			name:      "upgrade body",
			arguments: []string{"hosting", "upgrade", "hosting-1"},
			message:   "request body is required; use --data or --data-file",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			app := NewApp(strings.NewReader(""), &stdout, &stderr)
			code := app.Run(context.Background(), append([]string{"--token", "token"}, test.arguments...))
			if code != 1 {
				t.Fatalf("code = %d", code)
			}
			if !strings.Contains(stderr.String(), test.message) {
				t.Fatalf("stderr = %q", stderr.String())
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q", stdout.String())
			}
		})
	}
}

func TestHostingUpgradePrintsAPIErrorBody(t *testing.T) {
	setConfigHome(t)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusConflict)
		io.WriteString(writer, `{"code":"HOSTING_UPGRADE_IN_PROGRESS","message":"upgrade already queued"}`)
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	app := NewApp(strings.NewReader(""), &stdout, &stderr)
	code := app.Run(context.Background(), []string{
		"--token", "token",
		"--base-url", server.URL,
		"hosting", "upgrade", "hosting-1",
		"--data", `{"planId":"plan-2"}`,
	})
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), `"code": "HOSTING_UPGRADE_IN_PROGRESS"`) ||
		!strings.Contains(stdout.String(), `"message": "upgrade already queued"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func setConfigHome(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", root)
	} else {
		t.Setenv("XDG_CONFIG_HOME", root)
	}
	return root
}

func TestConfigRoundTripAndReplacement(t *testing.T) {
	setConfigHome(t)
	if err := SaveConfig(Config{APIKey: "first", BaseURL: "https://one.example"}); err != nil {
		t.Fatal(err)
	}
	if err := SaveConfig(Config{APIKey: "second", BaseURL: "https://two.example"}); err != nil {
		t.Fatal(err)
	}
	config, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.APIKey != "second" || config.BaseURL != "https://two.example" {
		t.Fatalf("config = %#v", config)
	}
	path, err := ConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "config.json" {
		t.Fatalf("config path = %q", path)
	}
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		t.Fatalf("config stat = %v, %v", info, err)
	}
}

func TestAuthStatusDoesNotPrintToken(t *testing.T) {
	setConfigHome(t)
	var stdout, stderr bytes.Buffer
	app := NewApp(strings.NewReader(""), &stdout, &stderr)
	if code := app.Run(context.Background(), []string{
		"auth", "login", "--token", "dr_live_secret",
	}); code != 0 {
		t.Fatalf("login code=%d stderr=%s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := app.Run(context.Background(), []string{"auth", "status"}); code != 0 {
		t.Fatalf("status code=%d stderr=%s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "dr_live_secret") {
		t.Fatalf("status leaked token: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"authenticated":true`) {
		t.Fatalf("status = %s", stdout.String())
	}
}

func TestPublicCommandDoesNotRequireToken(t *testing.T) {
	setConfigHome(t)
	t.Setenv("DRITE_API_KEY", "")
	var authorization string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		authorization = request.Header.Get("Authorization")
		writer.Header().Set("Content-Type", "application/json")
		io.WriteString(writer, `{"plans":[]}`)
	}))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	app := NewApp(strings.NewReader(""), &stdout, &stderr)
	code := app.Run(context.Background(), []string{
		"--base-url", server.URL,
		"public", "plans",
	})
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if authorization != "" {
		t.Fatalf("authorization = %q", authorization)
	}
	if !strings.Contains(stdout.String(), `"plans": []`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestParseRequiredBool(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		input string
		want  bool
	}{
		{input: "true", want: true},
		{input: "TRUE", want: true},
		{input: "false", want: false},
	} {
		got, err := parseRequiredBool(test.input, "--enabled")
		if err != nil {
			t.Fatalf("parse %q: %v", test.input, err)
		}
		if got != test.want {
			t.Fatalf("parse %q = %t", test.input, got)
		}
	}
	for _, input := range []string{"", "yes", "1"} {
		if _, err := parseRequiredBool(input, "--enabled"); err == nil {
			t.Fatalf("parse %q unexpectedly succeeded", input)
		}
	}
}

func TestGlobalBooleanFlagsRejectInvalidValues(t *testing.T) {
	t.Parallel()
	if _, _, err := parseGlobalOptions([]string{"--compact=maybe", "help"}); err == nil {
		t.Fatal("invalid --compact value unexpectedly succeeded")
	}
	if _, _, err := parseGlobalOptions([]string{"--api-key-header=1", "help"}); err == nil {
		t.Fatal("invalid --api-key-header value unexpectedly succeeded")
	}
}

func TestCurrentBackendRequests(t *testing.T) {
	setConfigHome(t)
	const customerVPS = "/api/auth/vps"
	const resellerVPS = "/api/reseller/vps"
	const account = "/api/auth/me"
	const hosting = "/api/auth/hosting"

	cases := []struct {
		name   string
		args   []string
		method string
		target string
		body   string
	}{
		{"VPS create keys", []string{"vps", "create", "--data", `{"name":"vm","sshPublicKeys":["ssh-key"]}`},
			"POST", customerVPS, `{"name":"vm","templateId":"","planId":"","durationType":"","password":"","sshPublicKeys":["ssh-key"]}`},
		{"custom VPS keys", []string{"vps", "custom-create", "--data", `{"name":"vm","sshPublicKeys":["ssh-key"]}`},
			"POST", customerVPS + "/custom", `{"templateId":"","durationType":"","cpu":0,"ram":0,"disk":0,"name":"vm","password":"","sshPublicKeys":["ssh-key"]}`},
		{"reinstall keeps keys", []string{"vps", "reinstall", "vm", "--data", `{"templateId":"image","password":"fixture"}`},
			"POST", customerVPS + "/vm/reinstall", `{"templateId":"image","password":"fixture"}`},
		{"reinstall clears keys", []string{"vps", "reinstall", "vm", "--data", `{"templateId":"image","password":"fixture","sshPublicKeys":[]}`},
			"POST", customerVPS + "/vm/reinstall", `{"templateId":"image","password":"fixture","sshPublicKeys":[]}`},
		{"replace SSH keys", []string{"vps", "ssh-keys", "vm", "--data", `{"keys":[]}`}, "PUT", customerVPS + "/vm/ssh-keys", `{"keys":[]}`},
		{"request IPv6", []string{"vps", "ipv6-request", "vm"}, "POST", customerVPS + "/vm/ipv6", ""},
		{"release IPv6", []string{"vps", "ipv6-release", "vm"}, "DELETE", customerVPS + "/vm/ipv6", ""},
		{"rollback snapshot", []string{"vps", "snapshot-rollback", "vm /1", "snapshot /1"},
			"POST", customerVPS + "/vm%20%2F1/snapshots/snapshot%20%2F1/rollback", ""},
		{"reseller plans", []string{"reseller", "vps", "plans"}, "GET", resellerVPS + "/plans", ""},
		{"reseller legacy plans", []string{"reseller", "vps-plans"}, "GET", resellerVPS + "/plans", ""},
		{"reseller templates", []string{"reseller", "vps", "templates"}, "GET", resellerVPS + "/templates", ""},
		{"reseller list", []string{"reseller", "vps", "list"}, "GET", resellerVPS, ""},
		{"reseller detail", []string{"reseller", "vps", "get", "vm"}, "GET", resellerVPS + "/vm", ""},
		{"reseller create", []string{"reseller", "vps", "create", "--data", `{"name":"vm","templateId":"image","planId":"offer","durationType":"monthly","password":"fixture"}`},
			"POST", resellerVPS, `{"name":"vm","templateId":"image","planId":"offer","durationType":"monthly","password":"fixture"}`},
		{"reseller custom quote", []string{"reseller", "vps", "custom-quote", "--data", `{"cpu":2,"ram":4,"disk":20,"durationType":"monthly"}`},
			"POST", resellerVPS + "/custom/quote", `{"durationType":"monthly","cpu":2,"ram":4,"disk":20}`},
		{"reseller custom create", []string{"reseller", "vps", "custom-create", "--data", `{"name":"vm","templateId":"image","cpu":2,"ram":4,"disk":20,"durationType":"monthly","password":"fixture"}`},
			"POST", resellerVPS + "/custom", `{"templateId":"image","durationType":"monthly","cpu":2,"ram":4,"disk":20,"name":"vm","password":"fixture"}`},
		{"account hosting prices", []string{"hosting", "plans", "--account"}, "GET", hosting + "/plans", ""},
		{"hosting upgrade status", []string{"hosting", "upgrade-status", "host"}, "GET", hosting + "/host/upgrade-status", ""},
		{"hosting fresh usage", []string{"hosting", "usage", "host", "--fresh"}, "GET", hosting + "/host/usage?fresh=1", ""},
		{"hosting databases", []string{"hosting", "databases", "host"}, "GET", hosting + "/host/databases", ""},
		{"hosting emails", []string{"hosting", "emails", "host"}, "GET", hosting + "/host/emails", ""},
		{"hosting sites", []string{"hosting", "sites", "host"}, "GET", hosting + "/host/sites", ""},
		{"ticket close", []string{"ticket", "close", "ticket"}, "PUT", "/api/auth/ticket/ticket/close", ""},
		{"ticket presign", []string{"ticket", "upload-url", "--data", `{"filename":"debug.log","mimeType":"text/plain"}`},
			"POST", "/api/auth/ticket/upload-url", `{"filename":"debug.log","mimeType":"text/plain; charset=utf-8"}`},
		{"webhook events", []string{"account", "webhook-events"}, "GET", account + "/webhook-events", ""},
		{"Discord webhook", []string{"account", "create-webhook", "--data", `{"type":"discord","url":"https://example.test","events":["vps.created"],"totpCode":"123456"}`},
			"POST", account + "/webhooks", `{"type":"discord","url":"https://example.test","events":["vps.created"],"totpCode":"123456"}`},
		{"test webhook", []string{"account", "test-webhook", "webhook", "--data", `{"event":"vps.created","totpCode":"123456"}`},
			"POST", account + "/webhooks/webhook/test", `{"totpCode":"123456","event":"vps.created"}`},
		{"announcement page", []string{"account", "announcements", "--view", "history", "--cursor", "next", "--limit", "10"},
			"GET", account + "/announcements?cursor=next&limit=10&view=history", ""},
		{"announcement detail", []string{"account", "announcement", "notice"}, "GET", account + "/announcements/notice", ""},
		{"announcement count", []string{"account", "announcements-unread-count"}, "GET", account + "/announcements/unread-count", ""},
		{"read announcement", []string{"account", "announcement-read", "notice"}, "POST", account + "/announcements/notice/read", ""},
		{"read all announcements", []string{"account", "announcements-read-all"}, "POST", account + "/announcements/read-all", ""},
		{"expiring API key", []string{"account", "create-api-key", "--data", `{"expiresAt":"2099-12-01T00:00:00Z","totpCode":"123456"}`},
			"POST", account + "/api-key", `{"totpCode":"123456","expiresAt":"2099-12-01T00:00:00Z"}`},
		{"delete passkey proof", []string{"account", "delete-passkey", "key", "--data", `{"stepUpProof":"fixture-proof"}`},
			"DELETE", account + "/passkeys/key", `{"stepUpProof":"fixture-proof"}`},
		{"passkey step-up options", []string{"account", "passkey-step-up-options"}, "POST", account + "/step-up/passkey/options", `{}`},
		{"passkey step-up verify", []string{"account", "passkey-step-up-verify", "--data", `{"flowId":"fixture-flow","response":{"id":"credential"}}`},
			"POST", account + "/step-up/passkey/verify", `{"flowId":"fixture-flow","response":{"id":"credential"}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tc.method || r.URL.RequestURI() != tc.target {
					t.Errorf("request = %s %s, want %s %s", r.Method, r.URL.RequestURI(), tc.method, tc.target)
				}
				if auth := r.Header.Get("Authorization"); auth != "Bearer dr_live_fixture" {
					t.Errorf("authorization = %q", auth)
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				if tc.body == "" {
					if len(body) > 0 {
						t.Errorf("unexpected body = %s", body)
					}
				} else {
					var got, want any
					if json.Unmarshal(body, &got) != nil || json.Unmarshal([]byte(tc.body), &want) != nil || !reflect.DeepEqual(got, want) {
						t.Errorf("body = %s, want %s", body, tc.body)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, `{"data":{"newBackendField":true}}`)
			}))
			defer server.Close()

			var stdout, stderr bytes.Buffer
			app := NewApp(strings.NewReader(""), &stdout, &stderr)
			args := append([]string{"--base-url", server.URL, "--token", "dr_live_fixture", "--compact"}, tc.args...)
			if code := app.Run(context.Background(), args); code != 0 {
				t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
			}
			if got := strings.TrimSpace(stdout.String()); got != `{"data":{"newBackendField":true}}` {
				t.Errorf("stdout = %q", got)
			}
		})
	}
}

func TestResellerOrdersRejectUnsupportedSSHKeys(t *testing.T) {
	setConfigHome(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unsupported SSH keys reached %s", r.URL.Path)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	for _, action := range []string{"create", "custom-create"} {
		t.Run(action, func(t *testing.T) {
			var stderr bytes.Buffer
			app := NewApp(strings.NewReader(""), io.Discard, &stderr)
			args := []string{"--base-url", server.URL, "--token", "dr_live_fixture", "reseller", "vps", action, "--data", `{"sshPublicKeys":["ssh-key"]}`}
			if code := app.Run(context.Background(), args); code != 1 || !strings.Contains(stderr.String(), "reseller VPS orders do not accept sshPublicKeys") {
				t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
			}
		})
	}
}
