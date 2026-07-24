package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/DriteStudio/drite-cli/drite"
)

var Version = "dev"

type App struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type globalOptions struct {
	token        string
	baseURL      string
	compact      bool
	apiKeyHeader bool
	timeout      time.Duration
}

func NewApp(stdin io.Reader, stdout io.Writer, stderr io.Writer) *App {
	return &App{Stdin: stdin, Stdout: stdout, Stderr: stderr}
}

func (app *App) Run(ctx context.Context, arguments []string) int {
	options, remaining, err := parseGlobalOptions(arguments)
	if err != nil {
		fmt.Fprintln(app.Stderr, err)
		return 2
	}
	if len(remaining) == 0 || remaining[0] == "help" {
		app.printHelp()
		return 0
	}
	if remaining[0] == "version" {
		fmt.Fprintln(app.Stdout, Version)
		return 0
	}

	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintln(app.Stderr, err)
		return 1
	}
	if remaining[0] == "auth" {
		return app.runAuth(config, remaining[1:])
	}
	if remaining[0] == "config" {
		return app.runConfig(config, remaining[1:])
	}

	token := firstNonEmpty(options.token, os.Getenv("DRITE_API_KEY"), config.APIKey)
	baseURL := firstNonEmpty(options.baseURL, os.Getenv("DRITE_API_URL"), config.BaseURL, drite.DefaultBaseURL)
	httpClient := &httpClientWithTimeout{timeout: options.timeout}
	clientOptions := []drite.Option{
		drite.WithBaseURL(baseURL),
		drite.WithHTTPClient(httpClient.client()),
		drite.WithUserAgent("drite-cli/" + Version),
	}
	if options.apiKeyHeader {
		clientOptions = append(clientOptions, drite.WithAPIKeyHeader())
	}
	client, err := drite.NewClient(token, clientOptions...)
	if err != nil {
		fmt.Fprintln(app.Stderr, err)
		return 1
	}

	response, err := app.execute(ctx, client, remaining)
	if response != nil {
		if printErr := printResponse(app.Stdout, response, options.compact); printErr != nil {
			fmt.Fprintln(app.Stderr, printErr)
			return 1
		}
	}
	if err != nil {
		if response == nil {
			fmt.Fprintln(app.Stderr, err)
		}
		return 1
	}
	return 0
}

type httpClientWithTimeout struct {
	timeout time.Duration
}

func (value httpClientWithTimeout) client() *http.Client {
	timeout := value.timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &http.Client{Timeout: timeout}
}

func parseGlobalOptions(arguments []string) (globalOptions, []string, error) {
	options := globalOptions{timeout: 30 * time.Second}
	index := 0
	for index < len(arguments) {
		arg := arguments[index]
		if !strings.HasPrefix(arg, "--") {
			break
		}
		name, inline, hasInline := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
		nextValue := func() (string, error) {
			if hasInline {
				return inline, nil
			}
			index++
			if index >= len(arguments) {
				return "", fmt.Errorf("--%s requires a value", name)
			}
			return arguments[index], nil
		}
		switch name {
		case "compact":
			if !hasInline {
				options.compact = true
				break
			}
			value, err := parseRequiredBool(inline, "--compact")
			if err != nil {
				return options, nil, err
			}
			options.compact = value
		case "api-key-header":
			if !hasInline {
				options.apiKeyHeader = true
				break
			}
			value, err := parseRequiredBool(inline, "--api-key-header")
			if err != nil {
				return options, nil, err
			}
			options.apiKeyHeader = value
		case "token":
			value, err := nextValue()
			if err != nil {
				return options, nil, err
			}
			options.token = value
		case "base-url":
			value, err := nextValue()
			if err != nil {
				return options, nil, err
			}
			options.baseURL = value
		case "timeout":
			value, err := nextValue()
			if err != nil {
				return options, nil, err
			}
			duration, err := time.ParseDuration(value)
			if err != nil || duration <= 0 {
				return options, nil, fmt.Errorf("--timeout must be a positive duration such as 30s")
			}
			options.timeout = duration
		default:
			return options, nil, fmt.Errorf("unknown global flag --%s", name)
		}
		index++
	}
	return options, arguments[index:], nil
}

func (app *App) runAuth(config Config, arguments []string) int {
	if len(arguments) == 0 {
		fmt.Fprintln(app.Stderr, "usage: drite auth login --token TOKEN | status | logout")
		return 2
	}
	switch arguments[0] {
	case "login":
		flags := newFlagSet("auth login")
		token := flags.String("token", "", "website-issued dr_live token")
		baseURL := flags.String("base-url", "", "API base URL")
		if err := flags.Parse(arguments[1:]); err != nil {
			fmt.Fprintln(app.Stderr, err)
			return 2
		}
		if strings.TrimSpace(*token) == "" {
			fmt.Fprintln(app.Stderr, "--token is required")
			return 2
		}
		config.APIKey = strings.TrimSpace(*token)
		if *baseURL != "" {
			config.BaseURL = *baseURL
		}
		if err := SaveConfig(config); err != nil {
			fmt.Fprintln(app.Stderr, err)
			return 1
		}
		path, _ := ConfigPath()
		fmt.Fprintf(app.Stdout, "Token saved to %s\n", path)
		return 0
	case "logout":
		config.APIKey = ""
		if err := SaveConfig(config); err != nil {
			fmt.Fprintln(app.Stderr, err)
			return 1
		}
		fmt.Fprintln(app.Stdout, "Token removed")
		return 0
	case "status":
		source := ""
		if os.Getenv("DRITE_API_KEY") != "" {
			source = "DRITE_API_KEY"
		} else if config.APIKey != "" {
			source, _ = ConfigPath()
		}
		fmt.Fprintf(
			app.Stdout,
			"{\"authenticated\":%t,\"tokenSource\":%s,\"baseUrl\":%s}\n",
			source != "",
			strconv.Quote(source),
			strconv.Quote(firstNonEmpty(os.Getenv("DRITE_API_URL"), config.BaseURL, drite.DefaultBaseURL)),
		)
		return 0
	default:
		fmt.Fprintln(app.Stderr, "usage: drite auth login --token TOKEN | status | logout")
		return 2
	}
}

func (app *App) runConfig(config Config, arguments []string) int {
	if len(arguments) == 0 || arguments[0] == "show" {
		path, _ := ConfigPath()
		fmt.Fprintf(
			app.Stdout,
			"{\"baseUrl\":%s,\"configPath\":%s}\n",
			strconv.Quote(firstNonEmpty(config.BaseURL, drite.DefaultBaseURL)),
			strconv.Quote(path),
		)
		return 0
	}
	if arguments[0] == "set-url" && len(arguments) == 2 {
		config.BaseURL = strings.TrimRight(arguments[1], "/")
		if err := SaveConfig(config); err != nil {
			fmt.Fprintln(app.Stderr, err)
			return 1
		}
		fmt.Fprintln(app.Stdout, "Base URL saved:", config.BaseURL)
		return 0
	}
	fmt.Fprintln(app.Stderr, "usage: drite config show | set-url URL")
	return 2
}

func (app *App) execute(
	ctx context.Context,
	client *drite.Client,
	arguments []string,
) (*drite.Response, error) {
	command := arguments[0]
	rest := arguments[1:]
	switch command {
	case "me":
		return client.Account.Profile(ctx)
	case "account":
		return runAccount(ctx, client, rest)
	case "vps":
		return runVPS(ctx, client, rest)
	case "hosting":
		return runHosting(ctx, client, rest)
	case "billing":
		return runBilling(ctx, client, rest)
	case "ticket", "tickets":
		return runTickets(ctx, client, rest)
	case "reseller":
		return runReseller(ctx, client, rest)
	case "public":
		return runPublic(ctx, client, rest)
	case "request", "raw":
		return runRaw(ctx, client, rest)
	default:
		return nil, fmt.Errorf("unknown command %q; run drite help", command)
	}
}

func newFlagSet(name string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	return flags
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func requireAction(arguments []string, usage string) (string, []string, error) {
	if len(arguments) == 0 {
		return "", nil, errors.New(usage)
	}
	return arguments[0], arguments[1:], nil
}

func requireID(arguments []string, name string) (string, error) {
	if len(arguments) == 0 || strings.TrimSpace(arguments[0]) == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return arguments[0], nil
}

func (app *App) printHelp() {
	fmt.Fprintln(app.Stdout, `Drite customer API CLI

Usage:
  drite [global flags] <command> <action> [arguments] [flags]

Global flags:
  --token TOKEN            override DRITE_API_KEY/config token
  --base-url URL           override DRITE_API_URL/config URL
  --compact                print compact JSON
  --api-key-header         send X-API-Key instead of Bearer
  --timeout 30s            request timeout

Commands:
  auth, config, me, account, vps, hosting, billing, ticket
  reseller, public, request, version

Mutation commands accept --data JSON or --data-file PATH.
See Docs.md for every action and Go SDK examples.`)
}
