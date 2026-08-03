package drite

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHostingUpgradeOptionsRoute(t *testing.T) {
	t.Parallel()
	var method, path string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		method = request.Method
		path = request.URL.EscapedPath()
		writer.Header().Set("Content-Type", "application/json")
		io.WriteString(writer, `{"data":{"options":[]}}`)
	}))
	defer server.Close()

	client, err := NewClient("token", WithBaseURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Hosting.UpgradeOptions(context.Background(), "hosting /1"); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodGet || path != "/api/auth/hosting/hosting%20%2F1/upgrade-options" {
		t.Fatalf("request = %s %s", method, path)
	}
}

func TestHostingUpgradeRouteAndBody(t *testing.T) {
	t.Parallel()
	var method, path, contentType string
	var body map[string]string
	var handlerErr error
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		method = request.Method
		path = request.URL.EscapedPath()
		contentType = request.Header.Get("Content-Type")
		handlerErr = json.NewDecoder(request.Body).Decode(&body)
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusAccepted)
		io.WriteString(writer, `{"jobId":"job-1"}`)
	}))
	defer server.Close()

	client, err := NewClient("token", WithBaseURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Hosting.Upgrade(
		context.Background(),
		"hosting-1",
		UpgradeHostingRequest{PlanID: "plan-2"},
	); err != nil {
		t.Fatal(err)
	}
	if handlerErr != nil {
		t.Fatal(handlerErr)
	}
	if method != http.MethodPost || path != "/api/auth/hosting/hosting-1/upgrade" {
		t.Fatalf("request = %s %s", method, path)
	}
	if contentType != "application/json" {
		t.Fatalf("content type = %q", contentType)
	}
	if len(body) != 1 || body["planId"] != "plan-2" {
		t.Fatalf("body = %#v", body)
	}
}

func TestServiceRoutes(t *testing.T) {
	t.Parallel()
	type requestSeen struct {
		method string
		path   string
	}
	var seen []requestSeen
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		entry := requestSeen{method: request.Method, path: request.URL.EscapedPath()}
		seen = append(seen, entry)
		writer.Header().Set("Content-Type", "application/json")
		io.WriteString(writer, `{"success":true}`)
	}))
	defer server.Close()
	client, _ := NewClient("token", WithBaseURL(server.URL))
	ctx := context.Background()

	calls := []func() error{
		func() error { _, err := client.VPS.Start(ctx, "vps-1"); return err },
		func() error {
			_, err := client.Hosting.VerifyDomain(ctx, VerifyDomainRequest{Domain: "example.com", Token: "abc"})
			return err
		},
		func() error {
			_, err := client.Billing.PayDueItem(ctx, PayDueItemRequest{Type: "vps", ServiceID: "vps-1"})
			return err
		},
		func() error {
			_, err := client.Tickets.Reply(ctx, "ticket-1", ReplyTicketRequest{Message: "hello"})
			return err
		},
		func() error {
			_, err := client.Reseller.CreateVPS(ctx, CreateVPSRequest{Name: "customer-vps"})
			return err
		},
	}
	for _, call := range calls {
		if err := call(); err != nil {
			t.Fatal(err)
		}
	}
	expected := []requestSeen{
		{method: "POST", path: "/api/auth/vps/vps-1/start"},
		{method: "POST", path: "/api/auth/hosting/verify-domain"},
		{method: "POST", path: "/api/auth/billing/due-items/pay"},
		{method: "POST", path: "/api/auth/ticket/ticket-1/reply"},
		{method: "POST", path: "/api/reseller/vps"},
	}
	if len(seen) != len(expected) {
		t.Fatalf("seen %d requests", len(seen))
	}
	for index := range expected {
		if seen[index].method != expected[index].method || seen[index].path != expected[index].path {
			t.Errorf("request %d = %s %s", index, seen[index].method, seen[index].path)
		}
	}
}

func TestTicketAttachmentPolicy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		contentType string
		allowed     bool
	}{
		{name: "debug.LOG", allowed: true},
		{name: "notes.txt", contentType: "application/octet-stream", allowed: true},
		{name: "screen.png", contentType: "image/png", allowed: true},
		{name: "payload.html", contentType: "image/png", allowed: true},
		{name: "invoice.pdf", contentType: "application/pdf", allowed: false},
		{name: "archive.zip", contentType: "application/zip", allowed: false},
	}
	for _, test := range tests {
		_, err := ValidateTicketAttachment(test.name, test.contentType)
		if test.allowed && err != nil {
			t.Errorf("%s should be allowed: %v", test.name, err)
		}
		if !test.allowed && err == nil {
			t.Errorf("%s should be rejected", test.name)
		}
	}
}

func TestTicketUploadMultipart(t *testing.T) {
	t.Parallel()
	var filename, contentType, value string
	var handlerErr error
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		reader, err := request.MultipartReader()
		if err != nil {
			handlerErr = err
			http.Error(writer, "invalid multipart request", http.StatusBadRequest)
			return
		}
		part, err := reader.NextPart()
		if err != nil {
			handlerErr = err
			http.Error(writer, "missing multipart attachment", http.StatusBadRequest)
			return
		}
		filename = part.FileName()
		contentType = part.Header.Get("Content-Type")
		content, _ := io.ReadAll(part)
		value = string(content)
		if _, err := reader.NextPart(); err != io.EOF {
			handlerErr = err
			http.Error(writer, "unexpected multipart tail", http.StatusBadRequest)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		io.WriteString(writer, `{"key":"tickets/u1/attachment.log"}`)
	}))
	defer server.Close()
	client, _ := NewClient("token", WithBaseURL(server.URL))
	response, err := client.Tickets.UploadAttachment(
		context.Background(),
		"debug.log",
		"application/octet-stream",
		strings.NewReader("hello"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if handlerErr != nil {
		t.Fatalf("multipart handler: %v", handlerErr)
	}
	if filename != "debug.log" || contentType != "text/plain; charset=utf-8" || value != "hello" {
		t.Fatalf("multipart filename=%q contentType=%q value=%q", filename, contentType, value)
	}
	var payload map[string]string
	if err := response.Decode(&payload); err != nil || payload["key"] == "" {
		t.Fatalf("payload=%v err=%v", payload, err)
	}
}

func TestTicketUploadRejectsTooLarge(t *testing.T) {
	t.Parallel()
	client, _ := NewClient("token")
	oversized := io.LimitReader(strings.NewReader(strings.Repeat("x", 1024)), MaxTicketAttachmentBytes+1)
	// Use a synthetic reader that produces one byte beyond the limit without
	// allocating a 10 MiB fixture.
	reader := io.MultiReader(
		io.LimitReader(zeroReader{}, MaxTicketAttachmentBytes),
		strings.NewReader("x"),
		oversized,
	)
	if _, err := client.Tickets.UploadAttachment(
		context.Background(), "debug.log", "", reader,
	); err == nil {
		t.Fatal("expected upload size error")
	}
}

type zeroReader struct{}

func (zeroReader) Read(buffer []byte) (int, error) {
	for index := range buffer {
		buffer[index] = 0
	}
	return len(buffer), nil
}
