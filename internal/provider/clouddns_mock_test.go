package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"testing"
)

const cloudDNSIdentifier = "identifier"

type cloudDNSInput struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	RData string `json:"rdata"`
	TTL   int64  `json:"ttl"`
}

type cloudDNSMock struct {
	mu                 sync.Mutex
	t                  *testing.T
	records            []map[string]any
	revisionState      string
	writes             int
	failReadAfterWrite bool
	failAfterWrites    int
	readFailed         bool
}

func newCloudDNSMock(t *testing.T) *cloudDNSMock {
	t.Helper()
	return &cloudDNSMock{t: t, revisionState: "active", failAfterWrites: 1, records: []map[string]any{
		{cloudDNSIdentifier: "unmanaged", "name": "@", "type": "NS", "rdata": "ns.example.", "ttl": nil, "immutable": true},
	}}
}

func (m *cloudDNSMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r.Header.Get("Authorization") != "Bearer cloud-key" {
		m.t.Error("CloudDNS bearer key missing")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var result any
	switch r.Method + " " + r.URL.Path {
	case "GET /v1/domain":
		result = []map[string]any{{"id": 42, "fqdn": "example.com", "isDnsManaged": true}}
	case "GET /v1/domain/42/dns/revision":
		result = []map[string]any{
			{cloudDNSIdentifier: "draft", "state": "draft", "serial": nil},
			{cloudDNSIdentifier: "revision", "state": m.revisionState, "serial": 2026101901},
		}
	case "GET /v1/domain/42/dns/revision/revision/record":
		if m.failReadAfterWrite && m.writes >= m.failAfterWrites && !m.readFailed {
			m.readFailed = true
			w.WriteHeader(http.StatusServiceUnavailable)
			if err := json.NewEncoder(w).Encode(map[string]any{"success": false, "errors": []any{map[string]string{"code": "temporarilyUnavailable", "message": "read failed"}}, "result": nil}); err != nil {
				m.t.Error(err)
			}
			return
		}
		result = m.records
	case "POST /v1/domain/42/changeset":
		if r.Header.Get("Content-Type") != "application/json" {
			m.t.Error("missing JSON content type")
		}
		result = m.changeset(w, r)
		if result == nil {
			return
		}
	default:
		m.t.Errorf("unexpected request %s %s", r.Method, r.URL)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"success": true, "errors": []any{}, "messages": []any{}, "meta": nil, "result": result}); err != nil {
		m.t.Error(err)
	}
}

func (m *cloudDNSMock) changeset(w http.ResponseWriter, r *http.Request) any {
	var body struct {
		AutoCommit bool            `json:"autoCommit"`
		Strict     bool            `json:"strict"`
		Create     []cloudDNSInput `json:"create"`
		Delete     []cloudDNSInput `json:"delete"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		m.t.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}
	if !body.AutoCommit || !body.Strict {
		m.t.Error("changeset must commit atomically and strictly")
	}
	for _, old := range body.Delete {
		index := slices.IndexFunc(m.records, func(record map[string]any) bool { return matchesCloudDNS(record, old) })
		if index < 0 {
			w.WriteHeader(http.StatusUnprocessableEntity)
			if err := json.NewEncoder(w).Encode(map[string]any{"success": false, "errors": []any{map[string]any{"code": "validationError", "message": "record missing"}}, "result": nil}); err != nil {
				m.t.Error(err)
			}
			return nil
		}
		if m.records[index]["immutable"] == true {
			m.t.Fatal("attempt to delete immutable record")
		}
		m.records = slices.Delete(m.records, index, index+1)
	}
	for _, next := range body.Create {
		m.records = append(m.records, map[string]any{cloudDNSIdentifier: fmt.Sprintf("record-%d", m.writes), "name": next.Name, "type": next.Type, "rdata": next.RData, "ttl": next.TTL, "immutable": false})
	}
	m.writes++
	return map[string]any{cloudDNSIdentifier: "revision", "state": "active", "serial": 2026101901}
}

func matchesCloudDNS(record map[string]any, input cloudDNSInput) bool {
	return record["name"] == input.Name && record["type"] == input.Type && record["rdata"] == input.RData && record["ttl"] == input.TTL
}

func (m *cloudDNSMock) removeManaged() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = m.records[:1]
}

func (m *cloudDNSMock) assertUnmanaged(t *testing.T) {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.records) != 1 || m.records[0][cloudDNSIdentifier] != "unmanaged" {
		t.Fatalf("unmanaged record changed: %+v", m.records)
	}
}
