package cli

import (
	"context"
	"fmt"

	"github.com/DriteStudio/drite-cli/drite"
)

func runAccount(
	ctx context.Context,
	client *drite.Client,
	arguments []string,
) (*drite.Response, error) {
	action, rest, err := requireAction(arguments, "usage: drite account ACTION")
	if err != nil {
		return nil, err
	}
	switch action {
	case "profile":
		return client.Account.Profile(ctx)
	case "update":
		body, err := bodyFromArgs[drite.UpdateProfileRequest]("account update", rest, true)
		if err != nil {
			return nil, err
		}
		return client.Account.UpdateProfile(ctx, body)
	case "resend-verification":
		return client.Account.ResendVerification(ctx)
	case "totp-secret":
		return client.Account.TOTPSecret(ctx)
	case "recovery-codes":
		return client.Account.RecoveryCodeSummary(ctx)
	case "generate-recovery-codes":
		body, err := bodyFromArgs[drite.StepUpRequest]("generate recovery codes", rest, false)
		if err != nil {
			return nil, err
		}
		return client.Account.GenerateRecoveryCodes(ctx, body)
	case "sessions":
		return client.Account.Sessions(ctx)
	case "revoke-session":
		id, err := requireID(rest, "session ID")
		if err != nil {
			return nil, err
		}
		body, err := bodyFromArgs[drite.StepUpRequest]("revoke session", rest[1:], false)
		if err != nil {
			return nil, err
		}
		return client.Account.RevokeSession(ctx, id, body)
	case "passkeys":
		return client.Account.Passkeys(ctx)
	case "passkey-options":
		return client.Account.BeginPasskeyRegistration(ctx)
	case "passkey-register":
		body, err := bodyFromArgs[drite.PasskeyRegistrationRequest]("passkey register", rest, true)
		if err != nil {
			return nil, err
		}
		return client.Account.FinishPasskeyRegistration(ctx, body)
	case "delete-passkey":
		id, err := requireID(rest, "passkey ID")
		if err != nil {
			return nil, err
		}
		body, err := bodyFromArgs[drite.StepUpRequest]("delete passkey", rest[1:], false)
		if err != nil {
			return nil, err
		}
		return client.Account.DeletePasskeyWithStepUp(ctx, id, body)
	case "passkey-step-up-options":
		return client.Account.BeginPasskeyStepUp(ctx)
	case "passkey-step-up-verify":
		body, err := bodyFromArgs[drite.PasskeyStepUpRequest]("passkey step-up verify", rest, true)
		if err != nil {
			return nil, err
		}
		return client.Account.FinishPasskeyStepUp(ctx, body)
	case "api-logs":
		return client.Account.APILogs(ctx)
	case "webhook-events":
		return client.Account.WebhookEvents(ctx)
	case "test-webhook":
		id, err := requireID(rest, "webhook ID")
		if err != nil {
			return nil, err
		}
		body, err := bodyFromArgs[drite.TestWebhookRequest]("test webhook", rest[1:], true)
		if err != nil {
			return nil, err
		}
		return client.Account.TestWebhook(ctx, id, body)
	case "webhooks":
		return client.Account.Webhooks(ctx)
	case "create-webhook":
		body, err := bodyFromArgs[drite.CreateWebhookRequest]("create webhook", rest, true)
		if err != nil {
			return nil, err
		}
		return client.Account.CreateWebhook(ctx, body)
	case "delete-webhook":
		id, err := requireID(rest, "webhook ID")
		if err != nil {
			return nil, err
		}
		body, err := bodyFromArgs[drite.StepUpRequest]("delete webhook", rest[1:], false)
		if err != nil {
			return nil, err
		}
		return client.Account.DeleteWebhook(ctx, id, body)
	case "create-api-key":
		body, err := bodyFromArgs[drite.CreateAPIKeyRequest]("create API key", rest, false)
		if err != nil {
			return nil, err
		}
		return client.Account.CreateAPIKeyWithOptions(ctx, body)
	case "revoke-api-key":
		body, err := bodyFromArgs[drite.StepUpRequest]("revoke API key", rest, false)
		if err != nil {
			return nil, err
		}
		return client.Account.RevokeAPIKey(ctx, body)
	case "api-key-security":
		body, err := bodyFromArgs[drite.APIKeySecurityRequest]("API key security", rest, true)
		if err != nil {
			return nil, err
		}
		return client.Account.SetAPIKeySecurity(ctx, body)
	case "announcements":
		flags := newFlagSet("account announcements")
		view := flags.String("view", "", "active or history")
		cursor := flags.String("cursor", "", "cursor from the previous page")
		limit := flags.Int("limit", 0, "page size")
		if err := flags.Parse(rest); err != nil {
			return nil, err
		}
		return client.Account.Announcements(ctx, drite.AnnouncementListOptions{
			View: *view, Cursor: *cursor, Limit: *limit,
		})
	case "announcements-unread-count":
		return client.Account.AnnouncementUnreadCount(ctx)
	case "announcements-read-all":
		return client.Account.ReadAllAnnouncements(ctx)
	case "announcement", "announcement-read":
		id, err := requireID(rest, "announcement ID")
		if err != nil {
			return nil, err
		}
		if action == "announcement-read" {
			return client.Account.ReadAnnouncement(ctx, id)
		}
		return client.Account.Announcement(ctx, id)
	default:
		return nil, fmt.Errorf("unknown account action %q", action)
	}
}
