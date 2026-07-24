package cli

import (
	"context"
	"fmt"

	"github.com/DriteStudio/drite-cli/drite"
)

func runContainers(
	ctx context.Context,
	client *drite.Client,
	arguments []string,
) (*drite.Response, error) {
	action, rest, err := requireAction(arguments, "usage: drite container ACTION")
	if err != nil {
		return nil, err
	}
	switch action {
	case "plans":
		return client.Containers.Plans(ctx)
	case "registry-templates":
		return client.Containers.RegistryTemplates(ctx)
	case "registries":
		return client.Containers.Registries(ctx)
	case "registry-create":
		body, err := bodyFromArgs[drite.CreateRegistryRequest]("registry create", rest, true)
		if err != nil {
			return nil, err
		}
		return client.Containers.CreateRegistry(ctx, body)
	case "registry-delete":
		id, err := requireID(rest, "registry ID")
		if err != nil {
			return nil, err
		}
		return client.Containers.DeleteRegistry(ctx, id)
	case "apps", "list":
		return client.Containers.Apps(ctx)
	case "create":
		body, err := bodyFromArgs[drite.CreateContainerAppRequest]("container create", rest, true)
		if err != nil {
			return nil, err
		}
		return client.Containers.CreateApp(ctx, body)
	}

	appID, err := requireID(rest, "container app ID")
	if err != nil {
		return nil, err
	}
	remaining := rest[1:]
	switch action {
	case "get":
		return client.Containers.App(ctx, appID)
	case "update":
		body, err := bodyFromArgs[drite.UpdateContainerAppRequest]("container update", remaining, true)
		if err != nil {
			return nil, err
		}
		return client.Containers.UpdateApp(ctx, appID, body)
	case "deploy", "start", "stop", "restart", "delete":
		return client.Containers.Action(ctx, appID, drite.ContainerAction(action))
	case "auto-renewal":
		flags := newFlagSet("container auto-renewal")
		enabledValue := flags.String("enabled", "", "enable auto-renewal: true or false (required)")
		if err := flags.Parse(remaining); err != nil {
			return nil, err
		}
		enabled, err := parseRequiredBool(*enabledValue, "--enabled")
		if err != nil {
			return nil, err
		}
		return client.Containers.SetAutoRenewal(ctx, appID, enabled)
	case "renew":
		body, err := bodyFromArgs[drite.RenewRequest]("container renew", remaining, false)
		if err != nil {
			return nil, err
		}
		return client.Containers.Renew(ctx, appID, body.DurationType)
	case "operations":
		return client.Containers.Operations(ctx, appID)
	case "runtime":
		return client.Containers.Runtime(ctx, appID)
	case "logs":
		flags := newFlagSet("container logs")
		pod := flags.String("pod", "", "pod name")
		tail := flags.Int("tail", 500, "tail lines")
		since := flags.Int("since", 3600, "seconds of history")
		if err := flags.Parse(remaining); err != nil {
			return nil, err
		}
		return client.Containers.Logs(ctx, appID, drite.ContainerLogOptions{
			Pod: *pod, TailLines: *tail, SinceSeconds: *since,
		})
	case "env":
		return client.Containers.Environment(ctx, appID)
	case "env-set":
		key, err := requireID(remaining, "environment key")
		if err != nil {
			return nil, err
		}
		body, err := bodyFromArgs[struct {
			Value string `json:"value"`
		}]("container env set", remaining[1:], true)
		if err != nil {
			return nil, err
		}
		return client.Containers.SetEnvironmentVariable(ctx, appID, key, body.Value)
	case "env-delete":
		key, err := requireID(remaining, "environment key")
		if err != nil {
			return nil, err
		}
		return client.Containers.DeleteEnvironmentVariable(ctx, appID, key)
	case "registry-attach", "registry-detach":
		registryID, err := requireID(remaining, "registry ID")
		if err != nil {
			return nil, err
		}
		if action == "registry-attach" {
			return client.Containers.AttachRegistry(ctx, appID, registryID)
		}
		return client.Containers.DetachRegistry(ctx, appID, registryID)
	case "domain-add":
		hostname, err := requireID(remaining, "hostname")
		if err != nil {
			return nil, err
		}
		return client.Containers.AddDomain(ctx, appID, hostname)
	case "domain-verify", "domain-delete":
		domainID, err := requireID(remaining, "domain ID")
		if err != nil {
			return nil, err
		}
		if action == "domain-verify" {
			return client.Containers.VerifyDomain(ctx, appID, domainID)
		}
		return client.Containers.DeleteDomain(ctx, appID, domainID)
	default:
		return nil, fmt.Errorf("unknown container action %q", action)
	}
}
