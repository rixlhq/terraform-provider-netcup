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

func TestAccScpFailoverIpV4_basic(t *testing.T) {
	server := newFailoverIPv4MockServer(t)
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
resource "netcup_scp_failover_ip_v4" "test" {
  user_id        = 12345
  failover_ip_id = 67890
  server_id      = 11111
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_failover_ip_v4.test", "user_id", "12345"),
					resource.TestCheckResourceAttr("netcup_scp_failover_ip_v4.test", "failover_ip_id", "67890"),
					resource.TestCheckResourceAttr("netcup_scp_failover_ip_v4.test", "server_id", "11111"),
					resource.TestCheckResourceAttr("netcup_scp_failover_ip_v4.test", "ip", "1.2.3.4"),
					resource.TestCheckResourceAttr("netcup_scp_failover_ip_v4.test", "id", "67890"),
				),
			},
			{
				Config: providerConfig + `
resource "netcup_scp_failover_ip_v4" "test" {
  user_id        = 12345
  failover_ip_id = 67890
  server_id      = 22222
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_failover_ip_v4.test", "server_id", "22222"),
				),
			},
		},
	})
}

type failoverIPv4Mock struct {
	mu       sync.Mutex
	serverID int64
}

func (m *failoverIPv4Mock) handleUpdate(w http.ResponseWriter, r *http.Request) {
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
	if err := json.NewEncoder(w).Encode(map[string]any{"uuid": "task-failover-v4", "state": taskStateFinished}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *failoverIPv4Mock) handleList(w http.ResponseWriter, r *http.Request) {
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
			"id":         67890,
			"ip":         "1.2.3.4",
			"cidrSuffix": 24,
			"editable":   true,
			"server":     map[string]any{"id": serverID, "name": "server-name"},
			"site":       map[string]any{"id": 1, "name": "nbg"},
		},
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func newFailoverIPv4MockServer(t *testing.T) *httptest.Server {
	t.Helper()

	mock := &failoverIPv4Mock{serverID: 11111}
	mux := http.NewServeMux()
	mux.HandleFunc("/scp-core/api/v1/users/12345/failoverips/v4/67890", mock.handleUpdate)
	mux.HandleFunc("/scp-core/api/v1/users/12345/failoverips/v4", mock.handleList)

	return httptest.NewServer(mux)
}
