package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccScpServerInterfaceFirewall_basic(t *testing.T) {
	server := newFirewallMockServer(t)
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: firewallTestConfig(server.URL, true, "[1]", "[10, 11]"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_server_interface_firewall.test", "server_id", "12345"),
					resource.TestCheckResourceAttr("netcup_scp_server_interface_firewall.test", "mac", "00:50:56:00:00:01"),
					resource.TestCheckResourceAttr("netcup_scp_server_interface_firewall.test", "active", "true"),
					resource.TestCheckResourceAttr("netcup_scp_server_interface_firewall.test", "copied_policy_ids.#", "1"),
					resource.TestCheckResourceAttr("netcup_scp_server_interface_firewall.test", "copied_policy_ids.0", "1"),
					resource.TestCheckResourceAttr("netcup_scp_server_interface_firewall.test", "user_policy_ids.#", "2"),
				),
			},
			{
				Config: firewallTestConfig(server.URL, false, "[]", "[]"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_server_interface_firewall.test", "active", "false"),
					resource.TestCheckResourceAttr("netcup_scp_server_interface_firewall.test", "copied_policy_ids.#", "0"),
					resource.TestCheckResourceAttr("netcup_scp_server_interface_firewall.test", "user_policy_ids.#", "0"),
				),
			},
		},
	})
}

func firewallTestConfig(serverURL string, active bool, copied, user string) string {
	return fmt.Sprintf(`
provider "netcup" {
  scp_access_token = "mock-token"
  scp_base_url     = "%s/scp-core"
}

resource "netcup_scp_server_interface_firewall" "test" {
  server_id = 12345
  mac       = "00:50:56:00:00:01"
  active    = %t

  copied_policy_ids = %s
  user_policy_ids   = %s
}
`, serverURL, active, copied, user)
}

type firewallMockState struct {
	Active         bool                 `json:"active"`
	CopiedPolicies []mockFirewallPolicy `json:"copiedPolicies"`
	UserPolicies   []mockFirewallPolicy `json:"userPolicies"`
}

type mockFirewallPolicy struct {
	ID int64 `json:"id"`
}

func newFirewallMockServer(t *testing.T) *httptest.Server {
	t.Helper()

	state := &firewallMockState{Active: false}

	mux := http.NewServeMux()

	mux.HandleFunc("/scp-core/api/v1/servers/12345/interfaces/00:50:56:00:00:01/firewall", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"active":              state.Active,
				"consistent":          true,
				"copiedPolicies":      state.CopiedPolicies,
				"userPolicies":        state.UserPolicies,
				"ingressImplicitRule": "DROP_ALL",
				"egressImplicitRule":  "ACCEPT_ALL",
			})
		case http.MethodPut:
			var payload firewallMockState
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer func() { _ = r.Body.Close() }()

			state.Active = payload.Active
			state.CopiedPolicies = payload.CopiedPolicies
			state.UserPolicies = payload.UserPolicies

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{"uuid": "task-1", "state": taskStateFinished})
		default:
			http.Error(w, "not allowed", http.StatusMethodNotAllowed)
		}
	})

	return httptest.NewServer(mux)
}
