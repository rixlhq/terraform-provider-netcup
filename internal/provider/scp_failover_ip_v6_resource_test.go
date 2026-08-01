package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccScpFailoverIpV6_basic(t *testing.T) {
	server := newFailoverIPv6MockServer(t)
	defer server.Close()

	providerConfig := fmt.Sprintf(`
provider "netcup" {
  scp_access_token = "mock-token"
  scp_base_url     = "%s/scp-core"
}
`, server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "netcup_scp_failover_ip_v6" "test" {
  user_id        = 12345
  failover_ip_id = 67890
  server_id      = 11111
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_failover_ip_v6.test", "user_id", "12345"),
					resource.TestCheckResourceAttr("netcup_scp_failover_ip_v6.test", "failover_ip_id", "67890"),
					resource.TestCheckResourceAttr("netcup_scp_failover_ip_v6.test", "server_id", "11111"),
					resource.TestCheckResourceAttr("netcup_scp_failover_ip_v6.test", "network_prefix", "2001:db8::"),
					resource.TestCheckResourceAttr("netcup_scp_failover_ip_v6.test", "network_prefix_length", "64"),
					resource.TestCheckResourceAttr("netcup_scp_failover_ip_v6.test", "id", "67890"),
				),
			},
			{
				Config: providerConfig + `
resource "netcup_scp_failover_ip_v6" "test" {
  user_id        = 12345
  failover_ip_id = 67890
  server_id      = 22222
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_failover_ip_v6.test", "server_id", "22222"),
				),
			},
		},
	})
}

type failoverIPv6Mock struct {
	mu       sync.Mutex
	serverID int64
}

func (m *failoverIPv6Mock) handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	if v, ok := payload["serverId"].(float64); ok {
		m.mu.Lock()
		m.serverID = int64(v)
		m.mu.Unlock()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(map[string]any{"uuid": "task-failover-v6", "state": taskStateFinished}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *failoverIPv6Mock) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mu.Lock()
	serverID := m.serverID
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode([]map[string]any{
		{
			"id":                  67890,
			"networkPrefix":       "2001:db8::",
			"networkPrefixLength": 64,
			"editable":            true,
			"server":              map[string]any{"id": serverID, "name": "server-name"},
			"site":                map[string]any{"id": 1, "name": "nbg"},
		},
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func newFailoverIPv6MockServer(t *testing.T) *httptest.Server {
	t.Helper()

	mock := &failoverIPv6Mock{serverID: 11111}
	mux := http.NewServeMux()
	mux.HandleFunc("/scp-core/api/v1/users/12345/failoverips/v6/67890", mock.handleUpdate)
	mux.HandleFunc("/scp-core/api/v1/users/12345/failoverips/v6", mock.handleList)

	return httptest.NewServer(mux)
}
