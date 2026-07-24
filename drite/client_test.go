package drite

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestClientBearerAndBasePath(t *testing.T) {
	t.Parallel()
	var gotPath, gotQuery, gotAuthorization, gotAgent string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotPath = request.URL.EscapedPath()
		gotQuery = request.URL.RawQuery
		gotAuthorization = request.Header.Get("Authorization")
		gotAgent = request.Header.Get("User-Agent")
		writer.Header().Set("Content-Type", "application/json")
		io.WriteString(writer, `{"ok":true}`)
	}))
	defer server.Close()

	client, err := NewClient(
		"dr_live_test",
		WithBaseURL(server.URL+"/gateway"),
		WithUserAgent("customer-app/1.0"),
	)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Request(
		context.Background(),
		http.MethodGet,
		"/api/auth/vps/id%2Fwith%2Fslash",
		url.Values{"take": {"10"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", response.StatusCode)
	}
	if gotPath != "/gateway/api/auth/vps/id%2Fwith%2Fslash" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotQuery != "take=10" {
		t.Fatalf("query = %q", gotQuery)
	}
	if gotAuthorization != "Bearer dr_live_test" {
		t.Fatalf("authorization = %q", gotAuthorization)
	}
	if gotAgent != "customer-app/1.0" {
		t.Fatalf("user-agent = %q", gotAgent)
	}
}

func TestClientAPIKeyHeader(t *testing.T) {
	t.Parallel()
	var bearer, apiKey string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		bearer = request.Header.Get("Authorization")
		apiKey = request.Header.Get("X-API-Key")
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	client, err := NewClient("secret", WithBaseURL(server.URL), WithAPIKeyHeader())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Account.Profile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if bearer != "" || apiKey != "secret" {
		t.Fatalf("bearer=%q apiKey=%q", bearer, apiKey)
	}
}

func TestAPIErrorPreservesContract(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("X-Request-ID", "req-123")
		writer.WriteHeader(http.StatusTooManyRequests)
		io.WriteString(writer, `{"message":"active ticket exists","code":"TICKET_ACTIVE_LIMIT"}`)
	}))
	defer server.Close()
	client, _ := NewClient("secret", WithBaseURL(server.URL))
	response, err := client.Tickets.Create(context.Background(), CreateTicketRequest{
		Subject: "help", Category: "technical", Message: "body",
	})
	if response == nil || response.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("response = %#v", response)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %T %v", err, err)
	}
	if apiErr.Code != "TICKET_ACTIVE_LIMIT" || apiErr.Message != "active ticket exists" {
		t.Fatalf("API error = %#v", apiErr)
	}
	if apiErr.RequestID != "req-123" {
		t.Fatalf("request ID = %q", apiErr.RequestID)
	}
}

func TestAPIErrorFallsBackToPlainText(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusBadGateway)
		io.WriteString(writer, "upstream unavailable")
	}))
	defer server.Close()
	client, _ := NewClient("secret", WithBaseURL(server.URL))
	_, err := client.Account.Profile(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %T %v", err, err)
	}
	if apiErr.Message != "upstream unavailable" {
		t.Fatalf("message = %q", apiErr.Message)
	}
}

func TestPublicRequestDoesNotNeedOrLeakToken(t *testing.T) {
	t.Parallel()
	var authorization string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		authorization = request.Header.Get("Authorization")
		io.WriteString(writer, `{}`)
	}))
	defer server.Close()
	client, _ := NewClient("", WithBaseURL(server.URL))
	if _, err := client.Public.Plans(context.Background()); err != nil {
		t.Fatal(err)
	}
	if authorization != "" {
		t.Fatalf("public authorization = %q", authorization)
	}
	if _, err := client.Account.Profile(context.Background()); !errors.Is(err, ErrMissingToken) {
		t.Fatalf("authenticated error = %v", err)
	}
}

func TestClientRejectsAbsoluteEndpoint(t *testing.T) {
	t.Parallel()
	client, _ := NewClient("secret")
	_, err := client.Request(context.Background(), http.MethodGet, "https://evil.example/api", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "start with /") {
		t.Fatalf("error = %v", err)
	}
}

func TestResponseLimit(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		io.WriteString(writer, strings.Repeat("x", 9))
	}))
	defer server.Close()
	client, _ := NewClient("secret", WithBaseURL(server.URL), WithMaxResponseBytes(8))
	if _, err := client.Account.Profile(context.Background()); err == nil {
		t.Fatal("expected response size error")
	}
}
