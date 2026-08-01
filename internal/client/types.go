package client

import "encoding/json"

// DNSRecord represents a single DNS record in the netcup API.
type DNSRecord struct {
	ID           string `json:"id"`
	Hostname     string `json:"hostname"`
	Type         string `json:"type"`
	Priority     *int64 `json:"priority,string,omitempty"`
	Destination  string `json:"destination"`
	DeleteRecord bool   `json:"deleterecord"`
}

// DNSRecordSet wraps a list of DNS records as used by the netcup API.
type DNSRecordSet struct {
	DNSRecords []DNSRecord `json:"dnsrecords"`
}

// DNSZone represents the DNS zone metadata returned by the netcup API.
type DNSZone struct {
	Name         string `json:"name"`
	TTL          int64  `json:"ttl,string"`
	Serial       string `json:"serial"`
	Refresh      string `json:"refresh"`
	Retry        string `json:"retry"`
	Expire       string `json:"expire"`
	DnsSecStatus bool   `json:"dnssecstatus"`
}

// apiRequest is the top-level request envelope sent to the JSON endpoint.
type apiRequest struct {
	Action string       `json:"action"`
	Param  requestParam `json:"param"`
}

type requestParam struct {
	CustomerNumber  int           `json:"customernumber"`
	APIKey          string        `json:"apikey"`
	APIPassword     string        `json:"apipassword,omitempty"`
	APISessionID    string        `json:"apisessionid,omitempty"`
	DomainName      string        `json:"domainname,omitempty"`
	DNSRecordSet    *DNSRecordSet `json:"dnsrecordset,omitempty"`
	DNSZone         *DNSZone      `json:"dnszone,omitempty"`
	ClientRequestID string        `json:"clientrequestid,omitempty"`
}

// Response is the common response envelope for netcup API calls.
type Response struct {
	ServerRequestID string          `json:"serverrequestid"`
	ClientRequestID string          `json:"clientrequestid"`
	Action          string          `json:"action"`
	Status          string          `json:"status"`
	StatusCode      int             `json:"statuscode"`
	ShortMessage    string          `json:"shortmessage"`
	LongMessage     string          `json:"longmessage"`
	ResponseData    json.RawMessage `json:"responsedata"`
}

type apiSessionData struct {
	APISessionID string `json:"apisessionid"`
}
