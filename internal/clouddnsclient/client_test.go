package clouddnsclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/rixlhq/terraform-provider-netcup/internal/clouddnsclient"
)

func TestCloudDNSLists(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer cloud-key" {
			t.Errorf("unexpected request: %s, auth present: %v", r.Method, r.Header.Get("Authorization") != "")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		switch r.URL.Path {
		case "/v1/domain":
			if r.URL.Query().Get("page") == "2" {
				reply(t, w, `{"success":true,"meta":{"pagination":{"nextPage":0}},"result":[{"id":42,"fqdn":"example.com","isDnsManaged":true}]}`)
				return
			}
			reply(t, w, `{"success":true,"meta":{"pagination":{"nextPage":2}},"result":[{"id":1,"fqdn":"legacy.example","isDnsManaged":false}]}`)
		case "/v1/domain/42/dns/revision":
			reply(t, w, `{"success":true,"meta":null,"result":[{"identifier":"draft","state":"draft","serial":null},{"identifier":"live","state":"active","serial":2026101901}]}`)
		case "/v1/domain/42/dns/revision/live/record":
			reply(t, w, `{"success":true,"meta":null,"result":[{"identifier":"record","name":"@","type":"NS","rdata":"ns.example.","ttl":null,"immutable":true}]}`)
		default:
			t.Errorf("unexpected path: %s", r.URL)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	c, err := clouddnsclient.New("cloud-key", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	domains, err := c.Domains(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(domains) != 2 || domains[1].ID != 42 || !domains[1].IsDNSManaged {
		t.Fatalf("domains: %+v", domains)
	}
	revisions, err := c.Revisions(t.Context(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(revisions) != 2 || revisions[0].Serial != nil || revisions[1].Serial == nil || *revisions[1].Serial != 2026101901 {
		t.Fatalf("revisions: %+v", revisions)
	}
	records, err := c.Records(t.Context(), 42, "live")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].TTL != nil || !records[0].Immutable || records[0].Identifier != "record" {
		t.Fatalf("records: %+v", records)
	}
}

func TestCloudDNSListEmpty404(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"success":false,"errors":[{"code":"resourceDoesNotExist","message":"No results found, the page number may be too high."}],"messages":[],"meta":null,"result":null}`))
	}))
	defer server.Close()
	c, err := clouddnsclient.New("cloud-key", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	domains, err := c.Domains(t.Context())
	if err != nil {
		t.Fatalf("expected nil error on 404 empty list, got: %v", err)
	}
	if len(domains) != 0 {
		t.Fatalf("expected 0 domains, got: %d", len(domains))
	}
}

func TestCloudDNSChangeset(t *testing.T) {
	t.Parallel()
	old := clouddnsclient.RecordInput{Name: "www", Type: "A", RData: "192.0.2.1", TTL: 300}
	next := clouddnsclient.RecordInput{Name: "www", Type: "A", RData: "192.0.2.2", TTL: 600}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/domain/42/changeset" || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Authorization") != "Bearer cloud-key" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		want := map[string]any{"autoCommit": true, "strict": true, "create": []clouddnsclient.RecordInput{next}, "delete": []clouddnsclient.RecordInput{old}}
		encoded, _ := json.Marshal(want)
		var expected map[string]json.RawMessage
		_ = json.Unmarshal(encoded, &expected)
		if !reflect.DeepEqual(body, expected) {
			t.Errorf("changeset = %s, want %s", body, expected)
		}
		reply(t, w, `{"success":true,"result":{"identifier":"new-revision","state":"active","serial":2026101902}}`)
	}))
	defer server.Close()
	c, err := clouddnsclient.New("cloud-key", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	revision, err := c.ChangeRecords(t.Context(), 42, []clouddnsclient.RecordInput{next}, []clouddnsclient.RecordInput{old})
	if err != nil || revision.Identifier != "new-revision" {
		t.Fatalf("revision=%+v error=%v", revision, err)
	}
}

func TestCloudDNSErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, body string
		status     int
	}{
		{"HTTP error", `{"success":false,"errors":[{"code":"authenticationError","message":"bad cloud-key"}],"result":null}`, 400},
		{"logical error", `{"success":false,"errors":[{"code":"validationError","message":"invalid"}],"result":null}`, 200},
		{"invalid JSON", `not json cloud-key`, 200},
		{"missing success", `{"result":[]}`, 200},
		{"missing result", `{"success":true}`, 200},
		{"HTTP success flag on error", `{"success":true,"result":[]}`, 500},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls++
				w.WriteHeader(test.status)
				reply(t, w, test.body)
			}))
			defer server.Close()
			c, err := clouddnsclient.New("cloud-key", server.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			_, err = c.Domains(t.Context())
			if err == nil {
				t.Fatal("expected error")
			}
			if strings.Contains(err.Error(), "cloud-key") {
				t.Fatalf("credential leaked in error: %v", err)
			}
			if calls != 1 {
				t.Fatalf("requests=%d, want one", calls)
			}
		})
	}
}

func TestCloudDNSSafety(t *testing.T) {
	t.Parallel()
	t.Run("invalid configuration", func(t *testing.T) {
		t.Parallel()
		for _, base := range []string{"https://user:pass@example.com", "file:///tmp/api", "https://example.com?key=secret"} {
			if _, err := clouddnsclient.New("key", base, nil); err == nil {
				t.Errorf("accepted %s", base)
			}
		}
		if _, err := clouddnsclient.New("", "", nil); err == nil {
			t.Error("accepted empty key")
		}
	})
	t.Run("cancelled context", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		c, err := clouddnsclient.New("key", "", nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.Domains(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("error=%v", err)
		}
	})
	t.Run("pagination loop", func(t *testing.T) {
		t.Parallel()
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			reply(t, w, `{"success":true,"result":[],"meta":{"pagination":{"nextPage":1}}}`)
		}))
		defer s.Close()
		c, err := clouddnsclient.New("key", s.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.Domains(t.Context()); err == nil {
			t.Fatal("accepted nonadvancing page")
		}
	})
	t.Run("no redirects", testNoRedirect)
}

func testNoRedirect(t *testing.T) {
	t.Helper()
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("redirect followed")
		reply(t, w, `{"success":true,"result":[]}`)
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	c, err := clouddnsclient.New("key", source.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Domains(t.Context()); err == nil {
		t.Fatal("accepted redirect")
	}
}

func reply(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	if _, err := w.Write([]byte(body)); err != nil {
		t.Error(err)
	}
}
