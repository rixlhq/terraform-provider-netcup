package scpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (c *Client) currentAccessToken() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.accessToken
}

// refresh serializes token rotation and reuses a refresh already completed for
// the access token rejected by this request. The boolean reports whether retrying
// with credentials is possible; absent refresh credentials preserve the API 401.
func (c *Client) refresh(ctx context.Context, rejectedToken string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.accessToken != rejectedToken {
		return true, nil
	}
	if c.refreshToken == "" {
		return false, nil
	}
	token, err := c.exchangeToken(ctx, c.refreshToken)
	if err != nil {
		return false, err
	}
	if token.AccessToken == "" {
		return false, errors.New("token response missing access_token")
	}
	c.accessToken = token.AccessToken
	if token.RefreshToken != "" {
		c.refreshToken = token.RefreshToken
	}
	return true, nil
}

func (c *Client) exchangeToken(ctx context.Context, refreshToken string) (tokenResponse, error) {
	data := url.Values{"client_id": {"scp"}, "refresh_token": {refreshToken}, "grant_type": {"refresh_token"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return tokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := c.httpClient.Do(req)
	if err != nil {
		return tokenResponse{}, err
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return tokenResponse{}, err
	}
	if response.StatusCode >= 400 {
		return tokenResponse{}, fmt.Errorf("token refresh returned %d: %s", response.StatusCode, string(body))
	}
	var token tokenResponse
	if err = json.Unmarshal(body, &token); err != nil {
		return tokenResponse{}, err
	}
	return token, nil
}
