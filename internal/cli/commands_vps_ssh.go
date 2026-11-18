package cli

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/DriteStudio/drite-cli/drite"
)

func (app *App) runVPSAccess(ctx context.Context, client *drite.Client, action string, arguments []string) (*drite.Response, error) {
	id, err := requireID(arguments, "VPS ID")
	if err != nil {
		return nil, err
	}
	flags := newFlagSet("vps " + action)
	timeout := flags.Duration("timeout", 5*time.Minute, "maximum wait for SSH readiness")
	interval := flags.Duration("interval", 2*time.Second, "SSH readiness polling interval (1s to 30s)")
	var identity string
	if action == "ssh" {
		flags.StringVar(&identity, "identity-file", "", "private key path for the local SSH client")
		flags.StringVar(&identity, "i", "", "short form of --identity-file")
	}
	if err := parseBodyFlags(flags, arguments[1:]); err != nil {
		return nil, err
	}
	if *timeout <= 0 {
		return nil, fmt.Errorf("--timeout must be positive")
	}
	if *interval < time.Second || *interval > 30*time.Second {
		return nil, fmt.Errorf("--interval must be between 1s and 30s")
	}

	response, err := client.VPS.WaitForSSH(ctx, id, drite.WaitForSSHOptions{Timeout: *timeout, Interval: *interval})
	if err != nil || action == "wait" {
		return response, err
	}

	var envelope struct {
		Data drite.SSHReadiness `json:"data"`
	}
	if err := response.Decode(&envelope); err != nil {
		return nil, err
	}
	username, ip, err := envelope.Data.Target()
	if err != nil {
		return nil, err
	}

	args := []string{"-l", username}
	if identity != "" {
		args = append(args, "-i", identity)
	}
	args = append(args, "--", ip.String())
	ssh := exec.CommandContext(ctx, "ssh", args...)
	ssh.Stdin, ssh.Stdout, ssh.Stderr = app.Stdin, app.Stdout, app.Stderr
	environment := ssh.Environ()
	ssh.Env = environment[:0]
	for _, entry := range environment {
		key, _, _ := strings.Cut(entry, "=")
		if !strings.EqualFold(key, "DRITE_API_KEY") {
			ssh.Env = append(ssh.Env, entry)
		}
	}
	if err := ssh.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("run local ssh client: %w", err)
	}
	return nil, nil
}
