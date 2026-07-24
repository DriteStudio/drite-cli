package drite

import (
	"context"
	"net/http"
	"net/url"
)

type BillingService struct {
	client *Client
}

type TransactionListOptions struct {
	Page      int
	Limit     int
	Month     string
	StartDate string
	EndDate   string
}

func (options TransactionListOptions) query() url.Values {
	query := make(url.Values)
	queryInt(query, "page", options.Page)
	queryInt(query, "limit", options.Limit)
	if options.Month != "" {
		query.Set("month", options.Month)
	}
	if options.StartDate != "" {
		query.Set("startDate", options.StartDate)
	}
	if options.EndDate != "" {
		query.Set("endDate", options.EndDate)
	}
	return query
}

type PayDueItemRequest struct {
	Type      string `json:"type"`
	ServiceID string `json:"serviceId"`
}

func (s *BillingService) Transactions(
	ctx context.Context,
	options TransactionListOptions,
) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, "/api/auth/transactions", options.query(), nil)
}

func (s *BillingService) ExportTransactions(
	ctx context.Context,
	options TransactionListOptions,
) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, "/api/auth/transactions/export", options.query(), nil)
}

func (s *BillingService) DueItems(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, "/api/auth/billing/due-items", nil, nil)
}

func (s *BillingService) PayDueItem(
	ctx context.Context,
	request PayDueItemRequest,
) (*Response, error) {
	return s.client.Request(ctx, http.MethodPost, "/api/auth/billing/due-items/pay", nil, request)
}

func (s *BillingService) TopupHistory(ctx context.Context) (*Response, error) {
	return s.client.Request(ctx, http.MethodGet, "/api/auth/topup/history", nil, nil)
}

func (s *BillingService) TopupStatus(ctx context.Context, referenceNumber string) (*Response, error) {
	reference, err := pathSegment(referenceNumber, "reference number")
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodGet, "/api/auth/topup/status/"+reference, nil, nil)
}

func (s *BillingService) SignedDocumentURL(
	ctx context.Context,
	documentID string,
	documentType string,
) (*Response, error) {
	query := url.Values{"id": {documentID}, "type": {documentType}}
	return s.client.Request(ctx, http.MethodGet, "/api/auth/biller/signed-url", query, nil)
}
