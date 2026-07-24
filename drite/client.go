package drite

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultBaseURL          = "https://dritestudio.co.th"
	defaultUserAgent        = "drite-go/1.0"
	defaultMaxResponseBytes = int64(16 << 20)
)

var ErrMissingToken = errors.New("drite: API token is required")

type authMode uint8

const (
	authBearer authMode = iota
	authAPIKeyHeader
)

// Client is safe for concurrent use. Service fields group customer APIs into
// small, discoverable functions.
type Client struct {
	baseURL         *url.URL
	token           string
	authMode        authMode
	httpClient      *http.Client
	userAgent       string
	maxResponseSize int64

	Account    *AccountService
	VPS        *VPSService
	Hosting    *HostingService
	Billing    *BillingService
	Tickets    *TicketService
	Containers *ContainerService
	Reseller   *ResellerService
	Public     *PublicService
}

type Option func(*Client) error

// WithBaseURL points the client at production, staging, or a local backend.
func WithBaseURL(rawURL string) Option {
	return func(client *Client) error {
		parsed, err := url.Parse(strings.TrimSpace(rawURL))
		if err != nil {
			return fmt.Errorf("drite: parse base URL: %w", err)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("drite: base URL must use http or https")
		}
		if parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return fmt.Errorf("drite: invalid base URL")
		}
		parsed.Path = strings.TrimRight(parsed.Path, "/")
		client.baseURL = parsed
		return nil
	}
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) error {
		if httpClient == nil {
			return fmt.Errorf("drite: HTTP client cannot be nil")
		}
		client.httpClient = httpClient
		return nil
	}
}

func WithUserAgent(userAgent string) Option {
	return func(client *Client) error {
		if strings.TrimSpace(userAgent) == "" {
			return fmt.Errorf("drite: user agent cannot be empty")
		}
		client.userAgent = strings.TrimSpace(userAgent)
		return nil
	}
}

// WithAPIKeyHeader sends the website token through X-API-Key. The default and
// recommended mode is Authorization: Bearer <token>; both are accepted by the
// legacy backend.
func WithAPIKeyHeader() Option {
	return func(client *Client) error {
		client.authMode = authAPIKeyHeader
		return nil
	}
}

func WithMaxResponseBytes(maxBytes int64) Option {
	return func(client *Client) error {
		if maxBytes <= 0 {
			return fmt.Errorf("drite: max response bytes must be positive")
		}
		client.maxResponseSize = maxBytes
		return nil
	}
}

// NewClient creates a customer API client. An empty token is permitted only so
// Public methods can be used; authenticated methods return ErrMissingToken.
func NewClient(token string, options ...Option) (*Client, error) {
	baseURL, _ := url.Parse(DefaultBaseURL)
	client := &Client{
		baseURL:         baseURL,
		token:           strings.TrimSpace(token),
		authMode:        authBearer,
		httpClient:      &http.Client{Timeout: 30 * time.Second},
		userAgent:       defaultUserAgent,
		maxResponseSize: defaultMaxResponseBytes,
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(client); err != nil {
			return nil, err
		}
	}
	client.Account = &AccountService{client: client}
	client.VPS = &VPSService{client: client}
	client.Hosting = &HostingService{client: client}
	client.Billing = &BillingService{client: client}
	client.Tickets = &TicketService{client: client}
	client.Containers = &ContainerService{client: client}
	client.Reseller = &ResellerService{client: client}
	client.Public = &PublicService{client: client}
	return client, nil
}

// Request performs an authenticated request. endpoint must be an API path,
// never an absolute URL, so callers cannot accidentally send a token elsewhere.
func (c *Client) Request(
	ctx context.Context,
	method string,
	endpoint string,
	query url.Values,
	body any,
) (*Response, error) {
	return c.request(ctx, method, endpoint, query, body, true, "", nil)
}

// PublicRequest performs a request without attaching the customer token.
func (c *Client) PublicRequest(
	ctx context.Context,
	method string,
	endpoint string,
	query url.Values,
) (*Response, error) {
	return c.request(ctx, method, endpoint, query, nil, false, "", nil)
}

func (c *Client) request(
	ctx context.Context,
	method string,
	endpoint string,
	query url.Values,
	body any,
	authenticated bool,
	contentType string,
	rawBody io.Reader,
) (*Response, error) {
	if c == nil {
		return nil, fmt.Errorf("drite: nil client")
	}
	if authenticated && c.token == "" {
		return nil, ErrMissingToken
	}
	requestURL, err := c.endpointURL(endpoint, query)
	if err != nil {
		return nil, err
	}

	var requestBody io.Reader
	if rawBody != nil {
		requestBody = rawBody
	} else if body != nil {
		encoded, marshalErr := json.Marshal(body)
		if marshalErr != nil {
			return nil, fmt.Errorf("drite: encode request: %w", marshalErr)
		}
		requestBody = bytes.NewReader(encoded)
		contentType = "application/json"
	}

	request, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), requestURL, requestBody)
	if err != nil {
		return nil, fmt.Errorf("drite: create request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", c.userAgent)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if authenticated {
		if c.authMode == authAPIKeyHeader {
			request.Header.Set("X-API-Key", c.token)
		} else {
			request.Header.Set("Authorization", "Bearer "+c.token)
		}
	}

	httpResponse, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("drite: request failed: %w", err)
	}
	defer httpResponse.Body.Close()

	limited := io.LimitReader(httpResponse.Body, c.maxResponseSize+1)
	responseBody, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("drite: read response: %w", err)
	}
	if int64(len(responseBody)) > c.maxResponseSize {
		return nil, fmt.Errorf("drite: response exceeds %d bytes", c.maxResponseSize)
	}
	response := &Response{
		StatusCode: httpResponse.StatusCode,
		Header:     httpResponse.Header.Clone(),
		Body:       append(json.RawMessage(nil), responseBody...),
	}
	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		return response, newAPIError(response)
	}
	return response, nil
}

func (c *Client) endpointURL(endpoint string, query url.Values) (string, error) {
	if !strings.HasPrefix(endpoint, "/") {
		return "", fmt.Errorf("drite: endpoint must start with /")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return "", fmt.Errorf("drite: endpoint must be a relative API path")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("drite: pass query parameters separately")
	}
	target := *c.baseURL
	target.Path = strings.TrimRight(c.baseURL.Path, "/") + parsed.Path
	target.RawPath = strings.TrimRight(c.baseURL.EscapedPath(), "/") + parsed.EscapedPath()
	if target.RawPath == target.Path {
		target.RawPath = ""
	}
	if len(query) > 0 {
		target.RawQuery = query.Encode()
	}
	return target.String(), nil
}

func pathSegment(value, name string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("drite: %s is required", name)
	}
	return url.PathEscape(trimmed), nil
}

func queryInt(values url.Values, name string, value int) {
	if value > 0 {
		values.Set(name, fmt.Sprintf("%d", value))
	}
}
