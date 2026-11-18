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
	"sync/atomic"
	"testing"
	"time"
)

const readyGuest = `{"data":{"status":"ready","username":"deploy_user","ip":"2001:db8::1","checkedAt":"2026-11-18T10:00:00Z"}}`

func TestVPSWaitPollsUntilReady(t *testing.T) {
	setConfigHome(t)
	var requests atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.EscapedPath() != "/api/auth/vps/id%20%2F1/ssh-readiness" ||
			r.Header.Get("Authorization") != "Bearer dr_live_fixture" {
			t.Errorf("unexpected readiness request %s %s", r.Method, r.URL.EscapedPath())
		}
		if requests.Add(1) == 1 {
			io.WriteString(w, `{"data":{"status":"starting","username":null,"ip":null,"checkedAt":null}}`)
			return
		}
		io.WriteString(w, readyGuest)
	}))
	defer api.Close()

	var stdout, stderr bytes.Buffer
	app := NewApp(strings.NewReader(""), &stdout, &stderr)
	args := []string{"--base-url", api.URL, "--token", "dr_live_fixture", "--compact", "vps", "wait", "id /1", "--timeout", "3s", "--interval", "1s"}
	if code := app.Run(context.Background(), args); code != 0 || strings.TrimSpace(stdout.String()) != readyGuest || requests.Load() != 2 {
		t.Fatalf("exit = %d, requests = %d, stdout = %q, stderr = %q", code, requests.Load(), stdout.String(), stderr.String())
	}
}

func TestVPSWaitRetriesServiceUnavailableAfterRetryAfter(t *testing.T) {
	setConfigHome(t)
	var requests atomic.Int32
	var firstPoll atomic.Int64
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now().UnixNano()
		if requests.Add(1) == 1 {
			firstPoll.Store(now)
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(http.StatusServiceUnavailable)
			io.WriteString(w, `{"message":"Unable to check SSH readiness"}`)
			return
		}
		if elapsed := time.Duration(now - firstPoll.Load()); elapsed < 2*time.Second {
			t.Errorf("poll retried %s after 503 with Retry-After 2s", elapsed)
		}
		io.WriteString(w, readyGuest)
	}))
	defer api.Close()

	var stdout, stderr bytes.Buffer
	app := NewApp(strings.NewReader(""), &stdout, &stderr)
	args := []string{"--base-url", api.URL, "--token", "dr_live_fixture", "--compact", "vps", "wait", "vm", "--timeout", "6s", "--interval", "1s"}
	if code := app.Run(context.Background(), args); code != 0 || strings.TrimSpace(stdout.String()) != readyGuest ||
		requests.Load() != 2 || stderr.Len() != 0 {
		t.Fatalf("exit = %d, requests = %d, stdout = %q, stderr = %q", code, requests.Load(), stdout.String(), stderr.String())
	}
}

func TestVPSWaitReportsTerminalAndInvalidReadiness(t *testing.T) {
	setConfigHome(t)
	for _, tc := range []struct {
		name string
		body string
		err  string
	}{
		{"stopped", `{"data":{"status":"stopped"}}`, "status stopped"},
		{"unsupported", `{"data":{"status":"unsupported"}}`, "status unsupported"},
		{"failed", `{"data":{"status":"failed"}}`, "status failed"},
		{"invalid status", `{"data":{"status":"mystery"}}`, "invalid SSH readiness status"},
		{"missing target", `{"data":{"status":"ready"}}`, "no ready target"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				io.WriteString(w, tc.body)
			}))
			defer api.Close()

			var stdout, stderr bytes.Buffer
			app := NewApp(strings.NewReader(""), &stdout, &stderr)
			args := []string{"--base-url", api.URL, "--token", "dr_live_fixture", "vps", "wait", "vm", "--timeout", "3s"}
			if code := app.Run(context.Background(), args); code != 1 || stdout.Len() != 0 ||
				!strings.Contains(stderr.String(), tc.err) || requests.Load() != 1 {
				t.Fatalf("exit = %d, requests = %d, stdout = %q, stderr = %q", code, requests.Load(), stdout.String(), stderr.String())
			}
		})
	}
}

func TestVPSWaitCancelsInFlightRequest(t *testing.T) {
	setConfigHome(t)
	started := make(chan struct{})
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer api.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go func() {
		select {
		case <-started:
			cancel()
		case <-ctx.Done():
		}
	}()
	var stderr bytes.Buffer
	app := NewApp(strings.NewReader(""), io.Discard, &stderr)
	args := []string{"--base-url", api.URL, "--token", "dr_live_fixture", "vps", "wait", "vm"}
	if code := app.Run(ctx, args); code != 1 || !strings.Contains(stderr.String(), "context canceled") {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
}

func TestVPSWaitTimesOutWhenReadinessIsUnknown(t *testing.T) {
	setConfigHome(t)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"data":{"status":"unknown","username":null,"ip":null,"checkedAt":null}}`)
	}))
	defer api.Close()
	var stderr bytes.Buffer
	app := NewApp(strings.NewReader(""), io.Discard, &stderr)
	args := []string{"--base-url", api.URL, "--token", "dr_live_fixture", "vps", "wait", "vm", "--timeout", "400ms"}
	if code := app.Run(context.Background(), args); code != 1 || !strings.Contains(stderr.String(), "last status unknown") ||
		!strings.Contains(stderr.String(), "deadline exceeded") {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
}

func installFakeSSH(t *testing.T, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("native process fixture uses a POSIX script")
	}
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "ssh"), []byte("#!/bin/sh\n"+script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestVPSSSHUsesNativeClientAndPassesTerminalStreams(t *testing.T) {
	setConfigHome(t)
	t.Setenv("DRITE_API_KEY", "dr_live_fixture")
	installFakeSSH(t, `
if [ "${DRITE_API_KEY+x}" = x ]; then
  exit 90
fi
printf '[%s]\n' "$@"
IFS= read -r from_stdin
printf 'stdin=%s\n' "$from_stdin"
printf 'native stderr\n' >&2
exit 23
`)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer dr_live_fixture" {
			t.Error("readiness request omitted available API key")
		}
		io.WriteString(w, readyGuest)
	}))
	defer api.Close()

	var stdout, stderr bytes.Buffer
	app := NewApp(strings.NewReader("interactive input\n"), &stdout, &stderr)
	args := []string{"--base-url", api.URL, "vps", "ssh", "vm", "-i", "identity key file"}
	code := app.Run(context.Background(), args)
	if want := "[-l]\n[deploy_user]\n[-i]\n[identity key file]\n[--]\n[2001:db8::1]\nstdin=interactive input\n"; code != 23 ||
		stdout.String() != want || stderr.String() != "native stderr\n" {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
}

func TestVPSSSHRejectsAnInvalidTargetBeforeLaunchingSSH(t *testing.T) {
	setConfigHome(t)
	installFakeSSH(t, "exit 90\n")
	for _, tc := range []struct {
		name string
		body string
		err  string
	}{
		{"username", `{"data":{"status":"ready","username":"-oProxyCommand=unexpected","ip":"203.0.113.1"}}`, "invalid Linux username"},
		{"address", `{"data":{"status":"ready","username":"deploy_user","ip":"hostname.example"}}`, "invalid IP address"},
		{"zone", `{"data":{"status":"ready","username":"deploy_user","ip":"fe80::1%eth0"}}`, "invalid IP address"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.WriteString(w, tc.body)
			}))
			defer api.Close()

			var stderr bytes.Buffer
			app := NewApp(strings.NewReader(""), io.Discard, &stderr)
			args := []string{"--base-url", api.URL, "--token", "dr_live_fixture", "vps", "ssh", "vm"}
			if code := app.Run(context.Background(), args); code != 1 || !strings.Contains(stderr.String(), tc.err) {
				t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
			}
		})
	}
}
