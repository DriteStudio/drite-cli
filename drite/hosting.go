package drite

import (
	"context"
	"net/http"
	"net/url"
)

type HostingService struct {
	client *Client
}

type VerifyDomainRequest struct {
	Domain string `json:"domain"`
	Token  string `json:"token"`
}

type DeployHostingRequest struct {
	PlanID                  string `json:"planId"`
	DurationDays            int    `json:"duration"`
	Domain                  string `json:"domain"`
	Password                string `json:"password"`
	DomainVerificationToken string `json:"domainVerificationToken,omitempty"`
}

type UpgradeHostingRequest struct {
	PlanID string `json:"planId"`
}

func (s *HostingService) Plans(ctx context.Context) (*Response, error) {
	return s.client.PublicRequest(ctx, http.MethodGet, "/api/un_auth/hosting/plans", nil)
}

func (s *HostingService) CheckDomain(ctx context.Context, domain string) (*Response, error) {
	return s.client.Request(
		ctx,
		http.MethodGet,
		"/api/auth/hosting/check-domain",
		url.Values{"domain": {domain}},
		nil,
	)
}

func (s *HostingService) VerifyDomain(
	ctx context.Context,
	request VerifyDomainRequest,
) (*Response, error) {
	return s.client.Request(ctx, http.MethodPost, "/api/auth/hosting/verify-domain", nil, request)
}

func (s *HostingService) List(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, "/api/auth/hosting/list", nil, nil)
}

func (s *HostingService) Get(ctx context.Context, hostingID string) (*Response, error) {
	path, err := hostingPath(hostingID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodGet, path, nil, nil)
}

func (s *HostingService) Deploy(
	ctx context.Context,
	request DeployHostingRequest,
) (*Response, error) {
	return s.client.Request(ctx, http.MethodPost, "/api/auth/hosting/deploy", nil, request)
}

func (s *HostingService) Access(ctx context.Context, hostingID string) (*Response, error) {
	return s.postAction(ctx, hostingID, "access", nil)
}

func (s *HostingService) Renew(
	ctx context.Context,
	hostingID string,
	duration DurationType,
) (*Response, error) {
	body := struct {
		DurationType DurationType `json:"durationType,omitempty"`
	}{DurationType: duration}
	return s.postAction(ctx, hostingID, "renew", body)
}

func (s *HostingService) ActivationStatus(ctx context.Context, hostingID string) (*Response, error) {
	return s.getAction(ctx, hostingID, "activation-status")
}

func (s *HostingService) Activity(ctx context.Context, hostingID string) (*Response, error) {
	return s.getAction(ctx, hostingID, "activity")
}

func (s *HostingService) UpgradeOptions(ctx context.Context, hostingID string) (*Response, error) {
	return s.getAction(ctx, hostingID, "upgrade-options")
}

func (s *HostingService) Upgrade(
	ctx context.Context,
	hostingID string,
	request UpgradeHostingRequest,
) (*Response, error) {
	return s.postAction(ctx, hostingID, "upgrade", request)
}

// ToggleAutoRenewal mirrors the backend toggle endpoint. Read Hosting.Get
// first if the caller needs to guarantee a particular final state.
func (s *HostingService) ToggleAutoRenewal(ctx context.Context, hostingID string) (*Response, error) {
	return s.postAction(ctx, hostingID, "autorenew", nil)
}

func (s *HostingService) Stats(ctx context.Context, hostingID string) (*Response, error) {
	return s.getAction(ctx, hostingID, "stats")
}

func (s *HostingService) Disk(ctx context.Context, hostingID string) (*Response, error) {
	return s.getAction(ctx, hostingID, "disk")
}

func (s *HostingService) Traffic(ctx context.Context, hostingID string) (*Response, error) {
	return s.getAction(ctx, hostingID, "traffic")
}

func (s *HostingService) ResetPassword(
	ctx context.Context,
	hostingID string,
	password string,
) (*Response, error) {
	return s.postAction(ctx, hostingID, "reset-password", ResetPasswordRequest{Password: password})
}

func (s *HostingService) Delete(ctx context.Context, hostingID string) (*Response, error) {
	path, err := hostingPath(hostingID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodDelete, path, nil, nil)
}

func (s *HostingService) getAction(
	ctx context.Context,
	hostingID string,
	action string,
) (*Response, error) {
	path, err := hostingPath(hostingID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodGet, path+"/"+action, nil, nil)
}

func (s *HostingService) postAction(
	ctx context.Context,
	hostingID string,
	action string,
	body any,
) (*Response, error) {
	path, err := hostingPath(hostingID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodPost, path+"/"+action, nil, body)
}

func hostingPath(hostingID string) (string, error) {
	id, err := pathSegment(hostingID, "hosting ID")
	if err != nil {
		return "", err
	}
	return "/api/auth/hosting/" + id, nil
}
