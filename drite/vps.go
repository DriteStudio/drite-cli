package drite

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type VPSService struct {
	client *Client
}

type DurationType string

const (
	DurationDaily   DurationType = "daily"
	DurationWeekly  DurationType = "weekly"
	DurationMonthly DurationType = "monthly"
	DurationYearly  DurationType = "yearly"
)

type VPSListOptions struct {
	Take int
	Skip int
}

func (options VPSListOptions) query() url.Values {
	query := make(url.Values)
	queryInt(query, "take", options.Take)
	if options.Skip > 0 {
		query.Set("skip", fmt.Sprintf("%d", options.Skip))
	}
	return query
}

type CreateVPSRequest struct {
	Name         string       `json:"name"`
	TemplateID   string       `json:"templateId"`
	PlanID       string       `json:"planId"`
	DurationType DurationType `json:"durationType"`
	Password     string       `json:"password"`
	IP           string       `json:"ip,omitempty"`
	NetworkRef   string       `json:"networkRef,omitempty"`
}

type CustomVPSQuoteRequest struct {
	TemplateID   string       `json:"templateId,omitempty"`
	DurationType DurationType `json:"durationType"`
	CPU          int          `json:"cpu"`
	RAM          int          `json:"ram"`
	Disk         int          `json:"disk"`
}

type CreateCustomVPSRequest struct {
	TemplateID   string       `json:"templateId"`
	DurationType DurationType `json:"durationType"`
	CPU          int          `json:"cpu"`
	RAM          int          `json:"ram"`
	Disk         int          `json:"disk"`
	Name         string       `json:"name"`
	Password     string       `json:"password"`
}

type RenewRequest struct {
	DurationType DurationType `json:"durationType"`
}

type AutoRenewalRequest struct {
	Enabled bool `json:"enabled"`
}

type UpgradeVPSRequest struct {
	PlanID string `json:"planId"`
}

type RenameVPSRequest struct {
	Name string `json:"name"`
}

type ReinstallVPSRequest struct {
	TemplateID string `json:"templateId"`
	Password   string `json:"password"`
}

type ResetPasswordRequest struct {
	Password string `json:"password"`
}

type CreateSnapshotRequest struct {
	Name string `json:"name,omitempty"`
}

type VPSControlAction string

const (
	VPSStart     VPSControlAction = "start"
	VPSStop      VPSControlAction = "stop"
	VPSReboot    VPSControlAction = "reboot"
	VPSForceStop VPSControlAction = "force-stop"
)

func (s *VPSService) Plans(ctx context.Context, templateID string) (*Response, error) {
	query := make(url.Values)
	if templateID != "" {
		query.Set("templateId", templateID)
	}
	return s.client.Request(ctx, http.MethodGet, "/api/auth/vps/plans", query, nil)
}

func (s *VPSService) Templates(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, "/api/auth/vps/templates", nil, nil)
}

func (s *VPSService) AvailableIPs(ctx context.Context, hostID string) (*Response, error) {
	id, err := pathSegment(hostID, "host ID")
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodGet, "/api/auth/vps/available-ips/"+id, nil, nil)
}

func (s *VPSService) List(ctx context.Context, options VPSListOptions) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, "/api/auth/vps", options.query(), nil)
}

func (s *VPSService) Job(ctx context.Context, jobID string) (*Response, error) {
	id, err := pathSegment(jobID, "job ID")
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodGet, "/api/auth/vps/job/"+id, nil, nil)
}

func (s *VPSService) Failed(ctx context.Context, options VPSListOptions) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, "/api/auth/vps/failed", options.query(), nil)
}

func (s *VPSService) AcknowledgeFailed(ctx context.Context, failureID string) (*Response, error) {
	id, err := pathSegment(failureID, "failure ID")
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodDelete, "/api/auth/vps/failed/"+id, nil, nil)
}

func (s *VPSService) QuoteCustom(
	ctx context.Context,
	request CustomVPSQuoteRequest,
) (*Response, error) {
	return s.client.Request(ctx, http.MethodPost, "/api/auth/vps/custom/quote", nil, request)
}

func (s *VPSService) CreateCustom(
	ctx context.Context,
	request CreateCustomVPSRequest,
) (*Response, error) {
	return s.client.Request(ctx, http.MethodPost, "/api/auth/vps/custom", nil, request)
}

func (s *VPSService) Get(ctx context.Context, vpsID string) (*Response, error) {
	path, err := vpsPath(vpsID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodGet, path, nil, nil)
}

func (s *VPSService) Status(ctx context.Context, vpsID string) (*Response, error) {
	path, err := vpsPath(vpsID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodGet, path+"/status", nil, nil)
}

func (s *VPSService) Stats(ctx context.Context, vpsID string) (*Response, error) {
	path, err := vpsPath(vpsID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodGet, path+"/stats", nil, nil)
}

func (s *VPSService) Activity(ctx context.Context, vpsID string) (*Response, error) {
	path, err := vpsPath(vpsID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodGet, path+"/activity", nil, nil)
}

func (s *VPSService) UpgradeOptions(ctx context.Context, vpsID string) (*Response, error) {
	path, err := vpsPath(vpsID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodGet, path+"/upgrade-options", nil, nil)
}

func (s *VPSService) Upgrade(
	ctx context.Context,
	vpsID string,
	request UpgradeVPSRequest,
) (*Response, error) {
	path, err := vpsPath(vpsID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodPost, path+"/upgrade", nil, request)
}

func (s *VPSService) Snapshots(ctx context.Context, vpsID string) (*Response, error) {
	path, err := vpsPath(vpsID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodGet, path+"/snapshots", nil, nil)
}

func (s *VPSService) CreateSnapshot(
	ctx context.Context,
	vpsID string,
	request CreateSnapshotRequest,
) (*Response, error) {
	path, err := vpsPath(vpsID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodPost, path+"/snapshots", nil, request)
}

func (s *VPSService) DeleteSnapshot(
	ctx context.Context,
	vpsID string,
	snapshotID string,
) (*Response, error) {
	path, err := vpsPath(vpsID)
	if err != nil {
		return nil, err
	}
	snapshot, err := pathSegment(snapshotID, "snapshot ID")
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodDelete, path+"/snapshots/"+snapshot, nil, nil)
}

func (s *VPSService) Create(ctx context.Context, request CreateVPSRequest) (*Response, error) {
	return s.client.Request(ctx, http.MethodPost, "/api/auth/vps", nil, request)
}

func (s *VPSService) Renew(
	ctx context.Context,
	vpsID string,
	request RenewRequest,
) (*Response, error) {
	path, err := vpsPath(vpsID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodPost, path+"/renew", nil, request)
}

func (s *VPSService) SetAutoRenewal(
	ctx context.Context,
	vpsID string,
	enabled bool,
) (*Response, error) {
	path, err := vpsPath(vpsID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodPost, path+"/auto-renewal", nil, AutoRenewalRequest{Enabled: enabled})
}

func (s *VPSService) Rename(
	ctx context.Context,
	vpsID string,
	request RenameVPSRequest,
) (*Response, error) {
	path, err := vpsPath(vpsID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodPost, path+"/rename", nil, request)
}

func (s *VPSService) Reinstall(
	ctx context.Context,
	vpsID string,
	request ReinstallVPSRequest,
) (*Response, error) {
	path, err := vpsPath(vpsID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodPost, path+"/reinstall", nil, request)
}

func (s *VPSService) Control(
	ctx context.Context,
	vpsID string,
	action VPSControlAction,
) (*Response, error) {
	path, err := vpsPath(vpsID)
	if err != nil {
		return nil, err
	}
	switch action {
	case VPSStart, VPSStop, VPSReboot, VPSForceStop:
	default:
		return nil, fmt.Errorf("drite: invalid VPS control action %q", action)
	}
	return s.client.Request(ctx, http.MethodPost, path+"/control", nil, struct {
		Action VPSControlAction `json:"action"`
	}{Action: action})
}

func (s *VPSService) Start(ctx context.Context, vpsID string) (*Response, error) {
	return s.simpleAction(ctx, vpsID, "start")
}

func (s *VPSService) Stop(ctx context.Context, vpsID string) (*Response, error) {
	return s.simpleAction(ctx, vpsID, "stop")
}

func (s *VPSService) Reboot(ctx context.Context, vpsID string) (*Response, error) {
	return s.simpleAction(ctx, vpsID, "reboot")
}

func (s *VPSService) ForceStop(ctx context.Context, vpsID string) (*Response, error) {
	return s.simpleAction(ctx, vpsID, "force-stop")
}

func (s *VPSService) NetworkReset(ctx context.Context, vpsID string) (*Response, error) {
	return s.simpleAction(ctx, vpsID, "network-reset")
}

func (s *VPSService) ResetPassword(
	ctx context.Context,
	vpsID string,
	request ResetPasswordRequest,
) (*Response, error) {
	path, err := vpsPath(vpsID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodPost, path+"/reset-password", nil, request)
}

func (s *VPSService) Delete(ctx context.Context, vpsID string) (*Response, error) {
	path, err := vpsPath(vpsID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodDelete, path, nil, nil)
}

func (s *VPSService) simpleAction(
	ctx context.Context,
	vpsID string,
	action string,
) (*Response, error) {
	path, err := vpsPath(vpsID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodPost, path+"/"+action, nil, nil)
}

func vpsPath(vpsID string) (string, error) {
	id, err := pathSegment(vpsID, "VPS ID")
	if err != nil {
		return "", err
	}
	return "/api/auth/vps/" + id, nil
}
