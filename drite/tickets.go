package drite

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path/filepath"
	"strings"
)

const (
	MaxTicketAttachments     = 5
	MaxTicketAttachmentBytes = int64(10 << 20)
)

type TicketService struct {
	client *Client
}

type TicketListOptions struct {
	Page     int
	Limit    int
	Status   string
	Category string
	Search   string
}

type CreateTicketRequest struct {
	Subject     string   `json:"subject"`
	Category    string   `json:"category"`
	Priority    string   `json:"priority,omitempty"`
	Message     string   `json:"message"`
	ServiceType string   `json:"serviceType,omitempty"`
	ServiceID   string   `json:"serviceId,omitempty"`
	Attachments []string `json:"attachments,omitempty"`
}

type ReplyTicketRequest struct {
	Message     string   `json:"message"`
	Attachments []string `json:"attachments,omitempty"`
}

type PresignTicketAttachmentRequest struct {
	Filename string `json:"filename"`
	MIMEType string `json:"mimeType"`
}

func (s *TicketService) List(
	ctx context.Context,
	options TicketListOptions,
) (*Response, error) {
	query := make(url.Values)
	queryInt(query, "page", options.Page)
	queryInt(query, "limit", options.Limit)
	if options.Status != "" {
		query.Set("status", options.Status)
	}
	if options.Category != "" {
		query.Set("category", options.Category)
	}
	if options.Search != "" {
		query.Set("search", options.Search)
	}
	return s.client.Request(ctx, http.MethodGet, "/api/auth/ticket/list", query, nil)
}

func (s *TicketService) Get(ctx context.Context, ticketID string) (*Response, error) {
	path, err := ticketPath(ticketID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodGet, path, nil, nil)
}

func (s *TicketService) Updates(
	ctx context.Context,
	ticketID string,
	updatedSince string,
) (*Response, error) {
	path, err := ticketPath(ticketID)
	if err != nil {
		return nil, err
	}
	query := make(url.Values)
	if updatedSince != "" {
		query.Set("updatedSince", updatedSince)
	}
	return s.client.Request(ctx, http.MethodGet, path+"/updates", query, nil)
}

func (s *TicketService) Create(
	ctx context.Context,
	request CreateTicketRequest,
) (*Response, error) {
	if err := validateAttachmentCount(request.Attachments); err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodPost, "/api/auth/ticket", nil, request)
}

func (s *TicketService) Reply(
	ctx context.Context,
	ticketID string,
	request ReplyTicketRequest,
) (*Response, error) {
	if err := validateAttachmentCount(request.Attachments); err != nil {
		return nil, err
	}
	path, err := ticketPath(ticketID)
	if err != nil {
		return nil, err
	}
	return s.client.Request(ctx, http.MethodPost, path+"/reply", nil, request)
}

// Close is intentionally not exposed: the backend now requires staff to close
// tickets so the one-active-ticket customer gate cannot be bypassed.

func (s *TicketService) PresignAttachment(
	ctx context.Context,
	request PresignTicketAttachmentRequest,
) (*Response, error) {
	contentType, err := ValidateTicketAttachment(request.Filename, request.MIMEType)
	if err != nil {
		return nil, err
	}
	request.MIMEType = contentType
	return s.client.Request(ctx, http.MethodPost, "/api/auth/ticket/upload-url", nil, request)
}

// UploadAttachment uploads at most 10 MiB and returns the backend payload. Use
// the returned "key" (not a public URL) in CreateTicketRequest or
// ReplyTicketRequest.
func (s *TicketService) UploadAttachment(
	ctx context.Context,
	filename string,
	contentType string,
	source io.Reader,
) (*Response, error) {
	if source == nil {
		return nil, fmt.Errorf("drite: attachment source is required")
	}
	normalizedType, err := ValidateTicketAttachment(filename, contentType)
	if err != nil {
		return nil, err
	}
	content, err := io.ReadAll(io.LimitReader(source, MaxTicketAttachmentBytes+1))
	if err != nil {
		return nil, fmt.Errorf("drite: read attachment: %w", err)
	}
	if int64(len(content)) > MaxTicketAttachmentBytes {
		return nil, fmt.Errorf("drite: ticket attachment exceeds 10 MiB")
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreatePart(filePartHeader(filename, normalizedType))
	if err != nil {
		return nil, fmt.Errorf("drite: create multipart attachment: %w", err)
	}
	if _, err := part.Write(content); err != nil {
		return nil, fmt.Errorf("drite: write multipart attachment: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("drite: close multipart body: %w", err)
	}
	return s.client.request(
		ctx,
		http.MethodPost,
		"/api/auth/ticket/upload",
		nil,
		nil,
		true,
		writer.FormDataContentType(),
		&body,
	)
}

func ValidateTicketAttachment(filename string, contentType string) (string, error) {
	extension := strings.ToLower(filepath.Ext(filename))
	if extension == ".txt" || extension == ".log" {
		return "text/plain; charset=utf-8", nil
	}
	normalizedType := strings.ToLower(strings.TrimSpace(contentType))
	if normalizedType == "" {
		normalizedType = strings.ToLower(mime.TypeByExtension(extension))
	}
	if strings.HasPrefix(normalizedType, "image/") {
		return normalizedType, nil
	}
	return "", fmt.Errorf("drite: only TXT, LOG, and image/* ticket attachments are supported")
}

func filePartHeader(filename string, contentType string) textproto.MIMEHeader {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(
		`form-data; name="file"; filename="%s"`,
		escapeMultipartFilename(filepath.Base(filename)),
	))
	header.Set("Content-Type", contentType)
	return header
}

func escapeMultipartFilename(filename string) string {
	return strings.NewReplacer("\\", "_", `"`, "_", "\r", "_", "\n", "_").Replace(filename)
}

func validateAttachmentCount(attachments []string) error {
	if len(attachments) > MaxTicketAttachments {
		return fmt.Errorf("drite: a ticket can contain at most %d attachments", MaxTicketAttachments)
	}
	return nil
}

func ticketPath(ticketID string) (string, error) {
	id, err := pathSegment(ticketID, "ticket ID")
	if err != nil {
		return "", err
	}
	return "/api/auth/ticket/" + id, nil
}
