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

func TestAccScpUserVlan_basic(t *testing.T) {
	server := newUserVlanMockServer(t)
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
resource "netcup_scp_user_vlan" "test" {
  user_id = 12345
  vlan_id = 777
  name    = "first-vlan"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_user_vlan.test", "user_id", "12345"),
					resource.TestCheckResourceAttr("netcup_scp_user_vlan.test", "vlan_id", "777"),
					resource.TestCheckResourceAttr("netcup_scp_user_vlan.test", "name", "first-vlan"),
					resource.TestCheckResourceAttr("netcup_scp_user_vlan.test", "id", "777"),
				),
			},
			{
				Config: providerConfig + `
resource "netcup_scp_user_vlan" "test" {
  user_id = 12345
  vlan_id = 777
  name    = "second-vlan"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_user_vlan.test", "name", "second-vlan"),
				),
			},
		},
	})
}

type userVlanMock struct {
	mu   sync.Mutex
	name string
}

func (m *userVlanMock) handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	if v, ok := payload["name"].(string); ok {
		m.mu.Lock()
		m.name = v
		m.mu.Unlock()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(map[string]any{"uuid": "task-vlan-1", "state": taskStateFinished}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *userVlanMock) handleRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mu.Lock()
	name := m.name
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"vlanId": 777,
		"name":   name,
		"site":   map[string]any{"id": 1, "name": "nbg"},
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func newUserVlanMockServer(t *testing.T) *httptest.Server {
	t.Helper()

	mock := &userVlanMock{name: "first-vlan"}
	mux := http.NewServeMux()
	mux.HandleFunc("/scp-core/api/v1/users/12345/vlans/777", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			mock.handleUpdate(w, r)
			return
		}
		mock.handleRead(w, r)
	})

	return httptest.NewServer(mux)
}
