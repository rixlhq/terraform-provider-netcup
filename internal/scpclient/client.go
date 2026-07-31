package scpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const defaultBaseURL = "https://www.servercontrolpanel.de/scp-core"
const tokenURL = "https://www.servercontrolpanel.de/realms/scp/protocol/openid-connect/token"

// Client interacts with the netcup SCP REST API.
type Client struct {
	baseURL      string
	accessToken  string
	refreshToken string
	httpClient   *http.Client
	mu           sync.Mutex
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
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

// Delete performs a DELETE request against the SCP API and returns the response body.
func (c *Client) Delete(ctx context.Context, path string) ([]byte, error) {
	return c.doWithRetry(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) doWithRetry(ctx context.Context, method, path string, query url.Values, body []byte) ([]byte, error) {
	bodyBytes, status, err := c.do(ctx, method, path, query, body)
	if err != nil {
		return nil, err
	}

	if status == http.StatusUnauthorized && c.refreshToken != "" {
		if err := c.refresh(ctx); err != nil {
			return nil, fmt.Errorf("token refresh failed: %w", err)
		}
		bodyBytes, status, err = c.do(ctx, method, path, query, body)
		if err != nil {
			return nil, err
		}
	}

	if status >= 400 {
		return nil, fmt.Errorf("scp API error %d: %s", status, string(bodyBytes))
	}

	return bodyBytes, nil
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body []byte) ([]byte, int, error) {
	c.mu.Lock()
	accessToken := c.accessToken
	c.mu.Unlock()

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
	if body != nil {
		req.Header.Set("Content-Type", "application/merge-patch+json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return respBody, resp.StatusCode, nil
}

func (c *Client) refresh(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.refreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	data := url.Values{}
	data.Set("client_id", "scp")
	data.Set("refresh_token", c.refreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("token refresh returned %d: %s", resp.StatusCode, string(body))
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return err
	}

	c.accessToken = tr.AccessToken
	if tr.RefreshToken != "" {
		c.refreshToken = tr.RefreshToken
	}

	// Expire a little before actual expiry to avoid edge cases.
	_ = time.Duration(tr.ExpiresIn) * time.Second

	return nil
}
