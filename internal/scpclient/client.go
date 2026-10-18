package scpclient

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
	"sync"
	"time"
)

// APIError is returned when the SCP API responds with a non-2xx status code.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	if len(e.Body) == 0 {
		return fmt.Sprintf("scp API error %d", e.StatusCode)
	}

	var parsed struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Errors  []struct {
			Field   string `json:"field"`
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal([]byte(e.Body), &parsed); err == nil {
		var details []string
		if parsed.Message != "" {
			details = append(details, parsed.Message)
		}
		for _, fe := range parsed.Errors {
			switch {
			case fe.Field != "" && fe.Message != "":
				details = append(details, fmt.Sprintf("%s: %s", fe.Field, fe.Message))
			case fe.Message != "":
				details = append(details, fe.Message)
			case fe.Field != "":
				details = append(details, fe.Field)
			}
		}
		if len(details) > 0 {
			msg := strings.Join(details, "; ")
			if parsed.Code != "" && !strings.Contains(msg, parsed.Code) {
				return fmt.Sprintf("scp API error %d (%s): %s", e.StatusCode, parsed.Code, msg)
			}
			return fmt.Sprintf("scp API error %d: %s", e.StatusCode, msg)
		}
	}

	return fmt.Sprintf("scp API error %d: %s", e.StatusCode, e.Body)
}

// IsNotFound reports whether err is an APIError with HTTP 404 status.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

const (
	defaultBaseURL   = "https://www.servercontrolpanel.de/scp-core"
	tokenURL         = "https://www.servercontrolpanel.de/realms/scp/protocol/openid-connect/token"
	maxRetries       = 3
	initialBackoff   = 500 * time.Millisecond
	taskTimeout      = 5 * time.Minute
	taskPollInterval = 2 * time.Second
)

// Client interacts with the netcup SCP REST API.
type Client struct {
	baseURL      string
	accessToken  string
	refreshToken string
	httpClient   *http.Client
	mu           sync.Mutex
}

// New creates an SCP API client.
func New(accessToken, refreshToken, baseURL string, httpClient *http.Client) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		baseURL:      strings.TrimRight(baseURL, "/"),
		accessToken:  accessToken,
		refreshToken: refreshToken,
		httpClient:   httpClient,
	}
}

// Get performs a GET request against the SCP API and returns the response body.
func (c *Client) Get(ctx context.Context, path string, query url.Values) ([]byte, error) {
	return c.doWithRetry(ctx, http.MethodGet, path, query, nil)
}

// Patch performs a PATCH request against the SCP API and returns the response body.
func (c *Client) Patch(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.doWithRetry(ctx, http.MethodPatch, path, nil, body)
}

// Post performs a POST request against the SCP API and returns the response body.
func (c *Client) Post(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.doWithRetry(ctx, http.MethodPost, path, nil, body)
}

// Put performs a PUT request against the SCP API and returns the response body.
func (c *Client) Put(ctx context.Context, path string, body []byte) ([]byte, error) {
	return c.doWithRetry(ctx, http.MethodPut, path, nil, body)
}

// Delete performs a DELETE request against the SCP API and returns the response body.
func (c *Client) Delete(ctx context.Context, path string) ([]byte, error) {
	return c.doWithRetry(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body []byte, accessToken string) ([]byte, int, error) {

	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, 0, err
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, 0, err
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	// The SCP API requires Content-Type on every write, including requests
	// without a body (e.g. POST .../rescuesystem), so set it by method.
	switch method {
	case http.MethodPatch:
		req.Header.Set("Content-Type", "application/merge-patch+json")
	case http.MethodPost, http.MethodPut, http.MethodDelete:
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return respBody, resp.StatusCode, nil
}
