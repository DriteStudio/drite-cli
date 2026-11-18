package cli

import (
	"context"
	"fmt"

	"github.com/DriteStudio/drite-cli/drite"
)

func runVPS(
	ctx context.Context,
	client *drite.Client,
	arguments []string,
) (*drite.Response, error) {
	action, rest, err := requireAction(arguments, "usage: drite vps ACTION")
	if err != nil {
		return nil, err
	}
	switch action {
	case "list":
		flags := newFlagSet("vps list")
		take := flags.Int("take", 0, "number of results")
		skip := flags.Int("skip", 0, "number to skip")
		if err := flags.Parse(rest); err != nil {
			return nil, err
		}
		return client.VPS.List(ctx, drite.VPSListOptions{Take: *take, Skip: *skip})
	case "plans":
		flags := newFlagSet("vps plans")
		templateID := flags.String("template-id", "", "filter by template")
		if err := flags.Parse(rest); err != nil {
			return nil, err
		}
		return client.VPS.Plans(ctx, *templateID)
	case "templates":
		return client.VPS.Templates(ctx)
	case "available-ips":
		id, err := requireID(rest, "host ID")
		if err != nil {
			return nil, err
		}
		return client.VPS.AvailableIPs(ctx, id)
	case "job":
		id, err := requireID(rest, "job ID")
		if err != nil {
			return nil, err
		}
		return client.VPS.Job(ctx, id)
	case "failed":
		flags := newFlagSet("vps failed")
		take := flags.Int("take", 0, "number of results")
		skip := flags.Int("skip", 0, "number to skip")
		if err := flags.Parse(rest); err != nil {
			return nil, err
		}
		return client.VPS.Failed(ctx, drite.VPSListOptions{Take: *take, Skip: *skip})
	case "ack-failed":
		id, err := requireID(rest, "failure ID")
		if err != nil {
			return nil, err
		}
		return client.VPS.AcknowledgeFailed(ctx, id)
	case "create":
		var body drite.CreateVPSRequest
		if response, err := vpsBodyWithSSHKeys(ctx, client.Account, "vps create", rest, true, &body, &body.SSHKeyIDs); err != nil {
			return response, err
		}
		return client.VPS.Create(ctx, body)
	case "custom-quote":
		body, err := bodyFromArgs[drite.CustomVPSQuoteRequest]("vps custom quote", rest, true)
		if err != nil {
			return nil, err
		}
		return client.VPS.QuoteCustom(ctx, body)
	case "custom-create":
		var body drite.CreateCustomVPSRequest
		if response, err := vpsBodyWithSSHKeys(ctx, client.Account, "vps custom create", rest, true, &body, &body.SSHKeyIDs); err != nil {
			return response, err
		}
		return client.VPS.CreateCustom(ctx, body)
	}

	id, err := requireID(rest, "VPS ID")
	if err != nil {
		return nil, err
	}
	bodyArgs := rest[1:]
	switch action {
	case "get":
		return client.VPS.Get(ctx, id)
	case "status":
		return client.VPS.Status(ctx, id)
	case "stats":
		return client.VPS.Stats(ctx, id)
	case "activity":
		return client.VPS.Activity(ctx, id)
	case "upgrade-options":
		return client.VPS.UpgradeOptions(ctx, id)
	case "snapshots":
		return client.VPS.Snapshots(ctx, id)
	case "snapshot-create":
		body, err := bodyFromArgs[drite.CreateSnapshotRequest]("snapshot create", bodyArgs, false)
		if err != nil {
			return nil, err
		}
		return client.VPS.CreateSnapshot(ctx, id, body)
	case "snapshot-delete", "snapshot-rollback":
		snapshotID, err := requireID(bodyArgs, "snapshot ID")
		if err != nil {
			return nil, err
		}
		if action == "snapshot-rollback" {
			return client.VPS.RollbackSnapshot(ctx, id, snapshotID)
		}
		return client.VPS.DeleteSnapshot(ctx, id, snapshotID)
	case "ipv6-request":
		return client.VPS.RequestIPv6(ctx, id)
	case "ipv6-release":
		return client.VPS.ReleaseIPv6(ctx, id)
	case "ssh-keys":
		var body drite.SetSSHPublicKeysRequest
		if response, err := vpsBodyWithSSHKeys(ctx, client.Account, "vps ssh-keys", bodyArgs, false, &body, &body.SSHKeyIDs); err != nil {
			return response, err
		}
		return client.VPS.SetSSHPublicKeys(ctx, id, body)
	case "renew":
		body, err := bodyFromArgs[drite.RenewRequest]("vps renew", bodyArgs, true)
		if err != nil {
			return nil, err
		}
		return client.VPS.Renew(ctx, id, body)
	case "auto-renewal":
		flags := newFlagSet("vps auto-renewal")
		enabledValue := flags.String("enabled", "", "enable auto-renewal: true or false (required)")
		if err := flags.Parse(bodyArgs); err != nil {
			return nil, err
		}
		enabled, err := parseRequiredBool(*enabledValue, "--enabled")
		if err != nil {
			return nil, err
		}
		return client.VPS.SetAutoRenewal(ctx, id, enabled)
	case "upgrade":
		body, err := bodyFromArgs[drite.UpgradeVPSRequest]("vps upgrade", bodyArgs, true)
		if err != nil {
			return nil, err
		}
		return client.VPS.Upgrade(ctx, id, body)
	case "rename":
		body, err := bodyFromArgs[drite.RenameVPSRequest]("vps rename", bodyArgs, true)
		if err != nil {
			return nil, err
		}
		return client.VPS.Rename(ctx, id, body)
	case "reinstall":
		var body drite.ReinstallVPSRequest
		if response, err := vpsBodyWithSSHKeys(ctx, client.Account, "vps reinstall", bodyArgs, true, &body, &body.SSHKeyIDs); err != nil {
			return response, err
		}
		return client.VPS.Reinstall(ctx, id, body)
	case "reset-password":
		body, err := bodyFromArgs[drite.ResetPasswordRequest]("vps reset password", bodyArgs, true)
		if err != nil {
			return nil, err
		}
		return client.VPS.ResetPassword(ctx, id, body)
	case "start":
		return client.VPS.Start(ctx, id)
	case "stop":
		return client.VPS.Stop(ctx, id)
	case "reboot":
		return client.VPS.Reboot(ctx, id)
	case "force-stop":
		return client.VPS.ForceStop(ctx, id)
	case "network-reset":
		return client.VPS.NetworkReset(ctx, id)
	case "delete":
		return client.VPS.Delete(ctx, id)
	default:
		return nil, fmt.Errorf("unknown VPS action %q", action)
	}
}
