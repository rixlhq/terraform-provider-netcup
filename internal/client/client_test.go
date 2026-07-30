package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientLogin(t *testing.T) {
	var lastBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		lastBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("reading body: %v", err)
		}
		resp := Response{
			Status:       "success",
			ShortMessage: "login successful",
			ResponseData: json.RawMessage(`{"apisessionid":"session-123"}`),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c, err := New("12345", "key", "secret", srv.URL+"?JSON", srv.Client())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	session, err := c.Login(context.Background())
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if session != "session-123" {
		t.Fatalf("expected session-123, got %s", session)
	}

	var req apiRequest
	if err := json.Unmarshal(lastBody, &req); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	if req.Action != "login" {
		t.Fatalf("expected action login, got %s", req.Action)
	}
	if req.Param.APIPassword != "secret" {
		t.Fatalf("expected api password to be sent")
	}
}

func TestClientInfoDnsRecords(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := Response{
			Status:       "success",
			ShortMessage: "ok",
			ResponseData: json.RawMessage(`{"dnsrecords":[{"id":"1","hostname":"@","type":"A","destination":"1.2.3.4"}]}`),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c, err := New("12345", "key", "secret", srv.URL+"?JSON", srv.Client())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	set, err := c.InfoDnsRecords(context.Background(), "session", "example.com")
	if err != nil {
		t.Fatalf("info dns records: %v", err)
	}
	if len(set.DNSRecords) != 1 {
		t.Fatalf("expected 1 record, got %d", len(set.DNSRecords))
	}
	if set.DNSRecords[0].Destination != "1.2.3.4" {
		t.Fatalf("expected destination 1.2.3.4, got %s", set.DNSRecords[0].Destination)
	}
}

func TestClientUpdateDnsRecords(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := Response{
			Status:       "success",
			ShortMessage: "ok",
			ResponseData: json.RawMessage(`{"dnsrecords":[{"id":"42","hostname":"www","type":"CNAME","destination":"example.com"}]}`),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c, err := New("12345", "key", "secret", srv.URL+"?JSON", srv.Client())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	p := int64(10)
	set, err := c.UpdateDnsRecords(context.Background(), "session", "example.com", &DNSRecordSet{
		DNSRecords: []DNSRecord{{Hostname: "www", Type: "CNAME", Destination: "example.com", Priority: &p}},
	})
	if err != nil {
		t.Fatalf("update dns records: %v", err)
	}
	if set.DNSRecords[0].ID != "42" {
		t.Fatalf("expected id 42, got %s", set.DNSRecords[0].ID)
	}
}
