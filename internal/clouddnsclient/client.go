// Package clouddnsclient implements netcup's CloudDNS REST API.
package clouddnsclient

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

// Client authenticates CloudDNS requests independently of the CCP and SCP APIs.
type Client struct {
	key     string
	baseURL string
	http    *http.Client
}

// New creates a client. Empty baseURL selects https://api.netcup.com.
// Redirects are rejected and writes are never automatically retried.
func New(key, baseURL string, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(key) == "" {
		return nil, errors.New("CloudDNS API key is required")
	}
	if baseURL == "" {
		baseURL = "https://api.netcup.com"
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("CloudDNS base URL must be an HTTP(S) URL without credentials, query or fragment")
	}
	transport := http.Client{Timeout: 30 * time.Second}
	if httpClient != nil {
		transport = *httpClient
	}
	transport.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{key: key, baseURL: strings.TrimRight(baseURL, "/"), http: &transport}, nil
}

type envelope struct {
	Success *bool     `json:"success"`
	Errors  []Message `json:"errors"`
	Meta    *struct {
		Pagination *struct {
			NextPage int `json:"nextPage"`
		} `json:"pagination"`
	} `json:"meta"`
	Result json.RawMessage `json:"result"`
}

// Message is a structured diagnostic returned by netcup.
type Message struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// APIError preserves HTTP status and netcup error codes, including HTTP 400 authentication errors.
type APIError struct {
	StatusCode int
	Messages   []Message
}

func (e *APIError) Error() string {
	details := make([]string, 0, len(e.Messages))
	for _, message := range e.Messages {
		details = append(details, message.Code+": "+message.Message)
	}
	return fmt.Sprintf("CloudDNS API HTTP %d: %s", e.StatusCode, strings.Join(details, "; "))
}

func (c *Client) request(ctx context.Context, method, path string, body any) (envelope, error) {
	var encoded []byte
	var err error
	if body != nil {
		encoded, err = json.Marshal(body)
		if err != nil {
			return envelope{}, fmt.Errorf("encode CloudDNS request: %w", err)
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(encoded))
	if err != nil {
		return envelope{}, fmt.Errorf("create CloudDNS request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := c.http.Do(req)
	if err != nil {
		return envelope{}, fmt.Errorf("CloudDNS request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	return c.decode(response)
}

func (c *Client) decode(response *http.Response) (envelope, error) {
	var result envelope
	if err := json.NewDecoder(io.LimitReader(response.Body, 16<<20)).Decode(&result); err != nil {
		return result, fmt.Errorf("invalid CloudDNS response (HTTP %d)", response.StatusCode)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || result.Success == nil || !*result.Success {
		messages := make([]Message, len(result.Errors))
		for i, message := range result.Errors {
			messages[i] = Message{Code: strings.ReplaceAll(message.Code, c.key, "[redacted]"), Message: strings.ReplaceAll(message.Message, c.key, "[redacted]")}
		}
		return result, &APIError{StatusCode: response.StatusCode, Messages: messages}
	}
	if len(result.Result) == 0 || string(result.Result) == "null" {
		return result, errors.New("CloudDNS response has no result")
	}
	return result, nil
}
