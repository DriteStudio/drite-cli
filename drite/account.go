package drite

import (
	"context"
	"net/http"
	"net/url"
)

type AccountService struct {
	client *Client
}

type StepUpRequest struct {
	CurrentPassword string `json:"currentPassword,omitempty"`
	TOTPCode        string `json:"totpCode,omitempty"`
}

type UpdateProfileRequest struct {
	FirstName       string `json:"firstName,omitempty"`
	LastName        string `json:"lastName,omitempty"`
	CurrentPassword string `json:"currentPassword,omitempty"`
	NewPassword     string `json:"newPassword,omitempty"`
	TOTPCode        string `json:"totpCode,omitempty"`
	TOTPSecret      string `json:"totpSecret,omitempty"`
	DisableTOTP     *bool  `json:"disableTotp,omitempty"`
}

type PasskeyRegistrationRequest struct {
	Name     string `json:"name,omitempty"`
	Response any    `json:"response"`
}

type APIKeySecurityRequest struct {
	AllowedIPs      []string `json:"allowedIps"`
	CurrentPassword string   `json:"currentPassword,omitempty"`
	TOTPCode        string   `json:"totpCode,omitempty"`
}

type CreateWebhookRequest struct {
	URL             string   `json:"url"`
	Events          []string `json:"events"`
	Secret          string   `json:"secret,omitempty"`
	CurrentPassword string   `json:"currentPassword,omitempty"`
	TOTPCode        string   `json:"totpCode,omitempty"`
}

func (s *AccountService) Profile(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, "/api/auth/me", nil, nil)
}

func (s *AccountService) UpdateProfile(ctx context.Context, request UpdateProfileRequest) (*Response, error) {
	return s.client.Request(ctx, http.MethodPut, "/api/auth/me", nil, request)
}

func (s *AccountService) ResendVerification(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodPost, "/api/auth/resend", nil, nil)
}

func (s *AccountService) TOTPSecret(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, "/api/auth/totp-secret", nil, nil)
}

func (s *AccountService) RecoveryCodeSummary(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, "/api/auth/me/recovery-codes", nil, nil)
}

func (s *AccountService) GenerateRecoveryCodes(ctx context.Context, request StepUpRequest) (*Response, error) {
	return s.client.Request(ctx, http.MethodPost, "/api/auth/me/recovery-codes", nil, request)
}

func (s *AccountService) Sessions(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, "/api/auth/me/sessions", nil, nil)
}

func (s *AccountService) RevokeSession(ctx context.Context, sessionID string, request StepUpRequest) (*Response, error) {
	id, err := pathSegment(sessionID, "session ID")
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodDelete, "/api/auth/me/sessions/"+id, nil, request)
}

func (s *AccountService) Passkeys(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, "/api/auth/me/passkeys", nil, nil)
}

func (s *AccountService) BeginPasskeyRegistration(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodPost, "/api/auth/me/passkeys/register-options", nil, nil)
}

func (s *AccountService) FinishPasskeyRegistration(
	ctx context.Context,
	request PasskeyRegistrationRequest,
) (*Response, error) {
	return s.client.Request(ctx, http.MethodPost, "/api/auth/me/passkeys/register-verify", nil, request)
}

func (s *AccountService) DeletePasskey(ctx context.Context, passkeyID string) (*Response, error) {
	id, err := pathSegment(passkeyID, "passkey ID")
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodDelete, "/api/auth/me/passkeys/"+id, nil, nil)
}

func (s *AccountService) SetAPIKeySecurity(
	ctx context.Context,
	request APIKeySecurityRequest,
) (*Response, error) {
	return s.client.Request(ctx, http.MethodPut, "/api/auth/me/api-key/security", nil, request)
}

func (s *AccountService) APILogs(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, "/api/auth/me/api-logs", nil, nil)
}

func (s *AccountService) Webhooks(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, "/api/auth/me/webhooks", nil, nil)
}

func (s *AccountService) CreateWebhook(
	ctx context.Context,
	request CreateWebhookRequest,
) (*Response, error) {
	return s.client.Request(ctx, http.MethodPost, "/api/auth/me/webhooks", nil, request)
}

func (s *AccountService) DeleteWebhook(
	ctx context.Context,
	webhookID string,
	request StepUpRequest,
) (*Response, error) {
	id, err := pathSegment(webhookID, "webhook ID")
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodDelete, "/api/auth/me/webhooks/"+id, nil, request)
}

func (s *AccountService) CreateAPIKey(ctx context.Context, request StepUpRequest) (*Response, error) {
	return s.client.Request(ctx, http.MethodPost, "/api/auth/me/api-key", nil, request)
}

func (s *AccountService) RevokeAPIKey(ctx context.Context, request StepUpRequest) (*Response, error) {
	return s.client.Request(ctx, http.MethodDelete, "/api/auth/me/api-key", nil, request)
}

// Raw exists for account endpoints whose WebAuthn payload evolves independently
// from this SDK. Prefer the named functions above for stable contracts.
func (s *AccountService) Raw(
	ctx context.Context,
	method string,
	endpoint string,
	query url.Values,
	body any,
) (*Response, error) {
	return s.client.Request(ctx, method, endpoint, query, body)
}
