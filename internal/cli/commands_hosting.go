package cli

import (
	"context"
	"fmt"

	"github.com/DriteStudio/drite-cli/drite"
)

func runHosting(
	ctx context.Context,
	client *drite.Client,
	arguments []string,
) (*drite.Response, error) {
	action, rest, err := requireAction(arguments, "usage: drite hosting ACTION")
	if err != nil {
		return nil, err
	}
	switch action {
	case "plans":
		return client.Hosting.Plans(ctx)
	case "list":
		return client.Hosting.List(ctx)
	case "check-domain":
		domain, err := requireID(rest, "domain")
		if err != nil {
			return nil, err
		}
		return client.Hosting.CheckDomain(ctx, domain)
	case "verify-domain":
		body, err := bodyFromArgs[drite.VerifyDomainRequest]("hosting verify domain", rest, true)
		if err != nil {
			return nil, err
		}
		return client.Hosting.VerifyDomain(ctx, body)
	case "deploy":
		body, err := bodyFromArgs[drite.DeployHostingRequest]("hosting deploy", rest, true)
		if err != nil {
			return nil, err
		}
		return client.Hosting.Deploy(ctx, body)
	}

	id, err := requireID(rest, "hosting ID")
	if err != nil {
		return nil, err
	}
	switch action {
	case "get":
		return client.Hosting.Get(ctx, id)
	case "access":
		return client.Hosting.Access(ctx, id)
	case "renew":
		body, err := bodyFromArgs[drite.RenewRequest]("hosting renew", rest[1:], false)
		if err != nil {
			return nil, err
		}
		return client.Hosting.Renew(ctx, id, body.DurationType)
	case "activation-status":
		return client.Hosting.ActivationStatus(ctx, id)
	case "activity":
		return client.Hosting.Activity(ctx, id)
	case "toggle-auto-renewal":
		return client.Hosting.ToggleAutoRenewal(ctx, id)
	case "stats":
		return client.Hosting.Stats(ctx, id)
	case "disk":
		return client.Hosting.Disk(ctx, id)
	case "traffic":
		return client.Hosting.Traffic(ctx, id)
	case "reset-password":
		body, err := bodyFromArgs[drite.ResetPasswordRequest]("hosting reset password", rest[1:], true)
		if err != nil {
			return nil, err
		}
		return client.Hosting.ResetPassword(ctx, id, body.Password)
	case "delete":
		return client.Hosting.Delete(ctx, id)
	default:
		return nil, fmt.Errorf("unknown hosting action %q", action)
	}
}
