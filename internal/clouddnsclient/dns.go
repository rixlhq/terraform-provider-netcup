package clouddnsclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// Domain identifies a domain visible to the CloudDNS API key.
type Domain struct {
	ID           int64  `json:"id"`
	FQDN         string `json:"fqdn"`
	IsDNSManaged bool   `json:"isDnsManaged"`
}

// Revision describes a zone revision. Drafts may have a null serial.
type Revision struct {
	Identifier string `json:"identifier"`
	State      string `json:"state"`
	Serial     *int64 `json:"serial"`
}

// RecordInput uses the API's DNS presentation format; MX priority belongs in RData.
type RecordInput struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	RData string `json:"rdata"`
	TTL   int64  `json:"ttl"`
}

// Record describes a record in a revision. System records may have no TTL.
type Record struct {
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	RData      string `json:"rdata"`
	TTL        *int64 `json:"ttl"`
	Immutable  bool   `json:"immutable"`
}

// Domains returns all pages of the domain list.
func (c *Client) Domains(ctx context.Context) ([]Domain, error) {
	return list[Domain](ctx, c, "/v1/domain")
}

// Revisions returns all revisions without assuming server ordering.
func (c *Client) Revisions(ctx context.Context, domainID int64) ([]Revision, error) {
	return list[Revision](ctx, c, domainPath(domainID)+"/dns/revision")
}

// Records returns the records in a specific revision.
func (c *Client) Records(ctx context.Context, domainID int64, revisionID string) ([]Record, error) {
	return list[Record](ctx, c, domainPath(domainID)+"/dns/revision/"+url.PathEscape(revisionID)+"/record")
}

// ChangeRecords atomically changes only the supplied records and commits the revision.
// Strict mode prevents silently ignoring a missing delete or an existing create.
func (c *Client) ChangeRecords(ctx context.Context, domainID int64, create, remove []RecordInput) (Revision, error) {
	body := struct {
		AutoCommit bool          `json:"autoCommit"`
		Strict     bool          `json:"strict"`
		Create     []RecordInput `json:"create,omitempty"`
		Delete     []RecordInput `json:"delete,omitempty"`
	}{AutoCommit: true, Strict: true, Create: create, Delete: remove}
	result, err := c.request(ctx, http.MethodPost, domainPath(domainID)+"/changeset", body)
	if err != nil {
		return Revision{}, err
	}
	var revision Revision
	if err = json.Unmarshal(result.Result, &revision); err != nil {
		return revision, fmt.Errorf("decode CloudDNS revision: %w", err)
	}
	if revision.Identifier == "" {
		return revision, errors.New("CloudDNS changeset returned no revision identifier")
	}
	return revision, nil
}

func domainPath(id int64) string { return "/v1/domain/" + strconv.FormatInt(id, 10) }

func list[T any](ctx context.Context, c *Client, path string) ([]T, error) {
	var all []T
	for page := 1; ; {
		requestPath := path
		if page > 1 {
			requestPath += "?page=" + strconv.Itoa(page)
		}
		response, err := c.request(ctx, http.MethodGet, requestPath, nil)
		if err != nil {
			var apiErr *APIError
			if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound && page == 1 {
				return nil, nil
			}
			return nil, err
		}
		var entries []T
		if err = json.Unmarshal(response.Result, &entries); err != nil {
			return nil, fmt.Errorf("decode CloudDNS list: %w", err)
		}
		all = append(all, entries...)
		if response.Meta == nil || response.Meta.Pagination == nil || response.Meta.Pagination.NextPage == 0 {
			return all, nil
		}
		next := response.Meta.Pagination.NextPage
		if next <= page || next > 100000 {
			return nil, fmt.Errorf("invalid CloudDNS pagination: next page %d after %d", next, page)
		}
		page = next
	}
}
