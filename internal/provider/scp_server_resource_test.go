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

func TestAccScpServer_basic(t *testing.T) {
	server := newScpServerMockServer(t)
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
				Config: scpServerConfig(providerConfig, "server1", "server-one", "HDD"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_server.test", "server_id", "12345"),
					resource.TestCheckResourceAttr("netcup_scp_server.test", "hostname", "server1"),
					resource.TestCheckResourceAttr("netcup_scp_server.test", "nickname", "server-one"),
					resource.TestCheckResourceAttr("netcup_scp_server.test", "autostart", "true"),
					resource.TestCheckResourceAttr("netcup_scp_server.test", "uefi", "true"),
					resource.TestCheckResourceAttr("netcup_scp_server.test", "bootorder.0", "HDD"),
					resource.TestCheckResourceAttr("netcup_scp_server.test", "name", "test-server"),
					resource.TestCheckResourceAttr("netcup_scp_server.test", "cpu_topology.socket_count", "1"),
				),
			},
			{
				Config: scpServerConfig(providerConfig, "server2", "server-two", "CDROM"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_server.test", "hostname", "server2"),
					resource.TestCheckResourceAttr("netcup_scp_server.test", "nickname", "server-two"),
					resource.TestCheckResourceAttr("netcup_scp_server.test", "bootorder.0", "CDROM"),
				),
			},
		},
	})
}

func scpServerConfig(providerConfig, hostname, nickname, firstBoot string) string {
	return providerConfig + fmt.Sprintf(`
resource "netcup_scp_server" "test" {
  server_id        = 12345
  hostname         = %q
  nickname         = %q
  autostart        = true
  uefi             = true
  bootorder        = [%q, "NETWORK"]
  os_optimization  = "LINUX"
  keyboard_layout  = "de"
  cpu_topology = {
    socket_count           = 1
    cores_per_socket_count = 2
  }
}
`, hostname, nickname, firstBoot)
}

type scpServerMock struct {
	mu           sync.Mutex
	hostname     string
	nickname     string
	autostart    bool
	uefi         bool
	bootOrder    []any
	osOpt        string
	keyboard     string
	socketCount  int64
	coresPerSock int64
}

func (m *scpServerMock) applyPatch(payload map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if v, ok := payload["hostname"].(string); ok {
		m.hostname = v
	}
	if v, ok := payload["nickname"].(string); ok {
		m.nickname = v
	}
	if v, ok := payload["autostart"].(bool); ok {
		m.autostart = v
	}
	if v, ok := payload["uefi"].(bool); ok {
		m.uefi = v
	}
	if v, ok := payload["bootorder"].([]any); ok {
		m.bootOrder = v
	}
	if v, ok := payload["osOptimization"].(string); ok {
		m.osOpt = v
	}
	if v, ok := payload["keyboardLayout"].(string); ok {
		m.keyboard = v
	}
	if v, ok := payload["cpuTopology"].(map[string]any); ok {
		if n, ok := v["socketCount"].(json.Number); ok {
			m.socketCount, _ = n.Int64()
		}
		if n, ok := v["coresPerSocketCount"].(json.Number); ok {
			m.coresPerSock, _ = n.Int64()
		}
	}
}

func (m *scpServerMock) handlePatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload map[string]any
	dec := json.NewDecoder(r.Body)
	dec.UseNumber()
	if err := dec.Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	m.applyPatch(payload)
	w.WriteHeader(http.StatusOK)
}

func (m *scpServerMock) handleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mu.Lock()
	resp := map[string]any{
		"name":               "test-server",
		"hostname":           m.hostname,
		"nickname":           m.nickname,
		"autostart":          m.autostart,
		"uefi":               m.uefi,
		"bootorder":          m.bootOrder,
		"osOptimization":     m.osOpt,
		"keyboardLayout":     m.keyboard,
		"cpuTopology":        map[string]any{"socketCount": m.socketCount, "coresPerSocketCount": m.coresPerSock},
		"disabled":           false,
		"rescueSystemActive": false,
		"snapshotAllowed":    true,
	}
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func newScpServerMockServer(t *testing.T) *httptest.Server {
	t.Helper()

	mock := &scpServerMock{
		hostname:     "server1",
		nickname:     "server-one",
		autostart:    true,
		uefi:         true,
		bootOrder:    []any{"HDD", "NETWORK"},
		osOpt:        "LINUX",
		keyboard:     "de",
		socketCount:  1,
		coresPerSock: 2,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/scp-core/api/v1/servers/12345", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			mock.handlePatch(w, r)
			return
		}
		mock.handleGet(w, r)
	})

	return httptest.NewServer(mux)
}
