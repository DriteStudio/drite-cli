package drite

import (
	"context"
	"net/http"
)

// ResellerService is available when the website-issued API key belongs to a
// reseller account. Normal customer keys receive the backend's 401 response.
type ResellerService struct {
	client *Client
}

func (s *ResellerService) VPSPlans(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, "/api/reseller/vps/plans", nil, nil)
}

func (s *ResellerService) VPSTemplates(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, "/api/reseller/vps/templates", nil, nil)
}

func (s *ResellerService) QuoteCustomVPS(
	ctx context.Context,
	request CustomVPSQuoteRequest,
) (*Response, error) {
	return s.client.Request(ctx, http.MethodPost, "/api/reseller/vps/custom/quote", nil, request)
}

func (s *ResellerService) CreateCustomVPS(
	ctx context.Context,
	request CreateCustomVPSRequest,
) (*Response, error) {
	return s.client.Request(ctx, http.MethodPost, "/api/reseller/vps/custom", nil, request)
}

func (s *ResellerService) VPSList(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, "/api/reseller/vps", nil, nil)
}

func (s *ResellerService) VPS(ctx context.Context, vpsID string) (*Response, error) {
	id, err := pathSegment(vpsID, "VPS ID")
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodGet, "/api/reseller/vps/"+id, nil, nil)
}

func (s *ResellerService) CreateVPS(
	ctx context.Context,
	request CreateVPSRequest,
) (*Response, error) {
	return s.client.Request(ctx, http.MethodPost, "/api/reseller/vps", nil, request)
}
