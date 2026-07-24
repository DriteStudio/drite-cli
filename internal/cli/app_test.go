package cli

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

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
