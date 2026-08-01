package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccScpServerInterface_basic(t *testing.T) {
	server := newScpServerInterfaceMockServer(t)
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
				Config: scpServerInterfaceConfig(providerConfig, 777, "virtio"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_server_interface.test", "server_id", "12345"),
					resource.TestCheckResourceAttr("netcup_scp_server_interface.test", attrVlanID, "777"),
					resource.TestCheckResourceAttr("netcup_scp_server_interface.test", "network_driver", "virtio"),
					resource.TestCheckResourceAttr("netcup_scp_server_interface.test", "mac", "00:11:22:33:44:55"),
					resource.TestCheckResourceAttr("netcup_scp_server_interface.test", "id", "00:11:22:33:44:55"),
					resource.TestCheckResourceAttr("netcup_scp_server_interface.test", "speed_in_mbits", "1000"),
				),
			},
			{
				Config: scpServerInterfaceConfig(providerConfig, 777, "e1000"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_server_interface.test", "network_driver", "e1000"),
					resource.TestCheckResourceAttr("netcup_scp_server_interface.test", "mac", "00:11:22:33:44:55"),
					resource.TestCheckResourceAttr("netcup_scp_server_interface.test", "speed_in_mbits", "1000"),
				),
			},
		},
	})
}

func scpServerInterfaceConfig(providerConfig string, vlanID int, driver string) string {
	return providerConfig + fmt.Sprintf(`
resource "netcup_scp_server_interface" "test" {
  server_id      = 12345
  vlan_id        = %d
  network_driver = %q
}
`, vlanID, driver)
}

type scpServerInterfaceMock struct {
	mu     sync.Mutex
	mac    string
	driver string
	speed  int64
	vlanID int64
}

func (m *scpServerInterfaceMock) handle(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/scp-core/api/v1/servers/12345/interfaces"), "/")
	mac := ""
	if len(parts) == 2 && parts[1] != "" {
		mac = parts[1]
	}

	switch r.Method {
	case http.MethodPost:
		m.handleCreate(w, r)
	case http.MethodGet:
		m.handleGet(w, r, mac)
	case http.MethodPatch:
		m.handleUpdate(w, r, mac)
	case http.MethodDelete:
		m.handleDelete(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *scpServerInterfaceMock) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		VlanID        int64  `json:"vlanId"`
		NetworkDriver string `json:"networkDriver"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	m.mac = "00:11:22:33:44:55"
	m.driver = req.NetworkDriver
	m.vlanID = req.VlanID
	m.speed = 1000
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"uuid":  "task-create",
		"state": taskStateFinished,
		"result": map[string]any{
			"mac":          m.mac,
			"driver":       m.driver,
			"speedInMBits": m.speed,
			"vlanId":       m.vlanID,
		},
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *scpServerInterfaceMock) handleGet(w http.ResponseWriter, _ *http.Request, mac string) {
	m.mu.Lock()
	if m.mac == "" || mac != m.mac {
		m.mu.Unlock()
		http.NotFound(w, nil)
		return
	}
	resp := map[string]any{
		"mac":          m.mac,
		"driver":       m.driver,
		"speedInMBits": m.speed,
	}
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *scpServerInterfaceMock) handleUpdate(w http.ResponseWriter, r *http.Request, mac string) {
	var req struct {
		Driver string `json:"driver"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	if m.mac == "" || mac != m.mac {
		m.mu.Unlock()
		http.NotFound(w, nil)
		return
	}
	m.driver = req.Driver
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"uuid":  "task-update",
		"state": taskStateFinished,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *scpServerInterfaceMock) handleDelete(w http.ResponseWriter, _ *http.Request) {
	m.mu.Lock()
	m.mac = ""
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"uuid":  "task-delete",
		"state": taskStateFinished,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func newScpServerInterfaceMockServer(t *testing.T) *httptest.Server {
	t.Helper()

	mock := &scpServerInterfaceMock{}
	mux := http.NewServeMux()
	mux.HandleFunc("/scp-core/api/v1/servers/12345/interfaces", mock.handle)
	mux.HandleFunc("/scp-core/api/v1/servers/12345/interfaces/", mock.handle)

	return httptest.NewServer(mux)
}
