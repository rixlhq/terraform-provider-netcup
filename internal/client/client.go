package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

const defaultEndpoint = "https://ccp.netcup.net/run/webservice/servers/endpoint.php?JSON"

// Client communicates with the netcup CCP JSON API.
type Client struct {
	customerNumber int
	apiKey         string
	apiPassword    string
	endpoint       string
	httpClient     *http.Client
}

// New creates a netcup API client.
func New(customerNumber, apiKey, apiPassword, endpoint string, httpClient *http.Client) (*Client, error) {
	cn, err := strconv.Atoi(customerNumber)
	if err != nil {
		return nil, fmt.Errorf("customer_number must be an integer: %w", err)
	}

	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		customerNumber: cn,
		apiKey:         apiKey,
		apiPassword:    apiPassword,
		endpoint:       endpoint,
		httpClient:     httpClient,
	}, nil
}

func (c *Client) doRequest(ctx context.Context, action string, param requestParam) (*Response, error) {
	reqBody, err := json.Marshal(apiRequest{Action: action, Param: param})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = httpResp.Body.Close() }()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	if httpResp.StatusCode >= 400 {
		return nil, fmt.Errorf("http error %d: %s", httpResp.StatusCode, string(body))
	}

	var resp Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if resp.Status != "success" {
		return nil, fmt.Errorf("netcup API error %d: %s - %s", resp.StatusCode, resp.ShortMessage, resp.LongMessage)
	}

	return &resp, nil
}

// Login starts an API session and returns the session ID.
func (c *Client) Login(ctx context.Context) (string, error) {
	resp, err := c.doRequest(ctx, "login", requestParam{
		CustomerNumber: c.customerNumber,
		APIKey:         c.apiKey,
		APIPassword:    c.apiPassword,
	})
	if err != nil {
		return "", err
	}

	var session apiSessionData
	if err := json.Unmarshal(resp.ResponseData, &session); err != nil {
		return "", err
	}

	return session.APISessionID, nil
}

// Logout ends an API session.
func (c *Client) Logout(ctx context.Context, sessionID string) error {
	_, err := c.doRequest(ctx, "logout", requestParam{
		CustomerNumber: c.customerNumber,
		APIKey:         c.apiKey,
		APISessionID:   sessionID,
	})
	return err
}

// InfoDnsZone returns metadata for the given DNS zone.
func (c *Client) InfoDnsZone(ctx context.Context, sessionID, domainName string) (*DNSZone, error) {
	resp, err := c.doRequest(ctx, "infoDnsZone", requestParam{
		CustomerNumber: c.customerNumber,
		APIKey:         c.apiKey,
		APISessionID:   sessionID,
		DomainName:     domainName,
	})
	if err != nil {
		return nil, err
	}

	var zone DNSZone
	if err := json.Unmarshal(resp.ResponseData, &zone); err != nil {
		return nil, err
	}

	return &zone, nil
}

// InfoDnsRecords returns all DNS records for the given zone.
func (c *Client) InfoDnsRecords(ctx context.Context, sessionID, domainName string) (*DNSRecordSet, error) {
	resp, err := c.doRequest(ctx, "infoDnsRecords", requestParam{
		CustomerNumber: c.customerNumber,
		APIKey:         c.apiKey,
		APISessionID:   sessionID,
		DomainName:     domainName,
	})
	if err != nil {
		return nil, err
	}

	var set DNSRecordSet
	if err := json.Unmarshal(resp.ResponseData, &set); err != nil {
		return nil, err
	}

	return &set, nil
}

// UpdateDnsRecords updates DNS records in the given zone.
func (c *Client) UpdateDnsRecords(ctx context.Context, sessionID, domainName string, records *DNSRecordSet) (*DNSRecordSet, error) {
	resp, err := c.doRequest(ctx, "updateDnsRecords", requestParam{
		CustomerNumber: c.customerNumber,
		APIKey:         c.apiKey,
		APISessionID:   sessionID,
		DomainName:     domainName,
		DNSRecordSet:   records,
	})
	if err != nil {
		return nil, err
	}

	var set DNSRecordSet
	if err := json.Unmarshal(resp.ResponseData, &set); err != nil {
		return nil, err
	}

	return &set, nil
}

// UpdateDnsZone updates the DNS zone settings.
func (c *Client) UpdateDnsZone(ctx context.Context, sessionID, domainName string, zone *DNSZone) (*DNSZone, error) {
	resp, err := c.doRequest(ctx, "updateDnsZone", requestParam{
		CustomerNumber: c.customerNumber,
		APIKey:         c.apiKey,
		APISessionID:   sessionID,
		DomainName:     domainName,
		DNSZone:        zone,
	})
	if err != nil {
		return nil, err
	}

	var updated DNSZone
	if err := json.Unmarshal(resp.ResponseData, &updated); err != nil {
		return nil, err
	}

	return &updated, nil
}
