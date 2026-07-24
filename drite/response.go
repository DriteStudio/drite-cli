package drite

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Response contains the complete API response. Body is kept as JSON so the
// SDK remains forward-compatible when the backend adds response fields.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       json.RawMessage
}

// Decode unmarshals the response body into dst.
func (r *Response) Decode(dst any) error {
	if r == nil {
		return fmt.Errorf("drite: nil response")
	}
	if dst == nil || len(r.Body) == 0 {
		return nil
	}
	if err := json.Unmarshal(r.Body, dst); err != nil {
		return fmt.Errorf("drite: decode response: %w", err)
	}
	return nil
}

// APIError is returned for every non-2xx response.
type APIError struct {
	StatusCode int
	Message    string
	Code       string
	RequestID  string
	Body       json.RawMessage
}

func (e *APIError) Error() string {
	if e == nil {
		return "drite: unknown API error"
	}
	switch {
	case e.Code != "" && e.Message != "":
		return fmt.Sprintf("drite API: %s (%s, HTTP %d)", e.Message, e.Code, e.StatusCode)
	case e.Message != "":
		return fmt.Sprintf("drite API: %s (HTTP %d)", e.Message, e.StatusCode)
	default:
		return fmt.Sprintf("drite API: HTTP %d", e.StatusCode)
	}
}

func newAPIError(response *Response) *APIError {
	apiErr := &APIError{
		StatusCode: response.StatusCode,
		RequestID:  response.Header.Get("X-Request-ID"),
		Body:       response.Body,
	}
	var payload struct {
		Message string `json:"message"`
		Error   string `json:"error"`
		Code    string `json:"code"`
	}
	if json.Unmarshal(response.Body, &payload) == nil {
		apiErr.Message = payload.Message
		if apiErr.Message == "" {
			apiErr.Message = payload.Error
		}
		apiErr.Code = payload.Code
	}
	if apiErr.Message == "" {
		message := []rune(strings.TrimSpace(string(response.Body)))
		const maxMessageRunes = 512
		if len(message) > maxMessageRunes {
			message = append(message[:maxMessageRunes], '…')
		}
		apiErr.Message = string(message)
	}
	return apiErr
}
