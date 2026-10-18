package scpclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

func TestFirewallPolicyTaskFailure(t *testing.T) {
	t.Parallel()
	for _, state := range []string{"ERROR", "ROLLBACK", "CANCELED"} {
		t.Run(state, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"firewallPolicy": map[string]any{"id": 1},
					"taskInfo":       map[string]any{"uuid": "failed-task", "state": state, "message": "update failed"},
				})
			}))
			defer server.Close()
			c := scpclient.New("token", "", server.URL, server.Client())
			_, err := c.Put(context.Background(), "/api/v1/users/123/firewall-policies/1", []byte(`{"name":"test"}`))
			if err == nil || !strings.Contains(err.Error(), "failed-task") {
				t.Fatalf("expected failed wrapped task, got %v", err)
			}
		})
	}
}

func TestFirewallPolicyTaskPolling(t *testing.T) {
	t.Parallel()
	polled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/users/123/firewall-policies/1":
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"firewallPolicy":{"id":1},"taskInfo":{"uuid":"pending-task","state":"PENDING"}}`))
		case "/api/v1/tasks/pending-task":
			polled = true
			_, _ = w.Write([]byte(`{"uuid":"pending-task","state":"FINISHED","result":{"id":1}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	c := scpclient.New("token", "", server.URL, server.Client())
	result, err := c.Put(context.Background(), "/api/v1/users/123/firewall-policies/1", []byte(`{"name":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !polled || string(result) != `{"id":1}` {
		t.Fatalf("wrapped task was not awaited: polled=%v result=%s", polled, result)
	}
}

func TestMalformedAcceptedTaskIsNotSuccess(t *testing.T) {
	t.Parallel()
	for _, body := range []string{`not-json`, `{}`, `null`, `{"taskInfo":null}`, `{"firewallPolicy":{"id":1}}`} {
		t.Run(body, func(t *testing.T) {
			t.Parallel()
			c := scpclient.New("token", "", "https://scp.invalid", &http.Client{Transport: retryTransport(func(_ *http.Request) (*http.Response, error) { return retryResponse(http.StatusAccepted, body), nil })})
			if _, err := c.Put(t.Context(), "/api/v1/users/123/firewall-policies/1", nil); err == nil {
				t.Fatal("malformed accepted response reported success")
			}
		})
	}
}

func TestAcceptedVLANWithoutTaskBody(t *testing.T) {
	t.Parallel()
	c := scpclient.New("token", "", "https://scp.invalid", &http.Client{Transport: retryTransport(func(_ *http.Request) (*http.Response, error) { return retryResponse(http.StatusAccepted, ""), nil })})
	// The OpenAPI documents a bodyless 202 for VLAN updates.
	if _, err := c.Put(t.Context(), "/api/v1/users/123/vlans/456", nil); err != nil {
		t.Fatal(err)
	}
}

func TestCompletedAcceptedTaskWithoutUUID(t *testing.T) {
	t.Parallel()
	// TaskInfo.uuid is optional in the published schema. An explicitly finished
	// task does not need polling, so its result remains usable without a UUID.
	c := scpclient.New("token", "", "https://scp.invalid", &http.Client{Transport: retryTransport(func(_ *http.Request) (*http.Response, error) {
		return retryResponse(http.StatusAccepted, `{"state":"FINISHED","result":{"id":1}}`), nil
	})})
	result, err := c.Put(t.Context(), "/api/v1/users/123/firewall-policies/1", nil)
	if err != nil || string(result) != `{"id":1}` {
		t.Fatalf("completed task result=%s, error=%v", result, err)
	}
}
