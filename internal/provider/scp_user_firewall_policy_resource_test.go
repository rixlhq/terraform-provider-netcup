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

func TestAccScpUserFirewallPolicy_basic(t *testing.T) {
	server := newFirewallPolicyMockServer(t)
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
				Config: firewallPolicyConfig(providerConfig, "first", "ACCEPT", "INGRESS", "TCP", "0.0.0.0/0", "", "80"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_user_firewall_policy.test", "user_id", "12345"),
					resource.TestCheckResourceAttr("netcup_scp_user_firewall_policy.test", "name", "test-policy"),
					resource.TestCheckResourceAttr("netcup_scp_user_firewall_policy.test", "description", "first"),
					resource.TestCheckResourceAttr("netcup_scp_user_firewall_policy.test", "id", "1"),
					resource.TestCheckResourceAttr("netcup_scp_user_firewall_policy.test", "rules.0.action", "ACCEPT"),
				),
			},
			{
				Config: firewallPolicyConfig(providerConfig, "second", "DROP", "EGRESS", "UDP", "", "::/0", "53"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_user_firewall_policy.test", "description", "second"),
					resource.TestCheckResourceAttr("netcup_scp_user_firewall_policy.test", "rules.0.action", "DROP"),
				),
			},
		},
	})
}

func firewallPolicyConfig(providerConfig, desc, action, direction, protocol, sources, destinations, ports string) string {
	sourcesAttr := ""
	if sources != "" {
		sourcesAttr = fmt.Sprintf("      sources = [%q]\n", sources)
	}
	destinationsAttr := ""
	if destinations != "" {
		destinationsAttr = fmt.Sprintf("      destinations = [%q]\n", destinations)
	}
	portsAttr := ""
	if ports != "" {
		portsAttr = fmt.Sprintf("      %s_ports = %q\n", mapPortsAttr(sources != ""), ports)
	}

	return providerConfig + fmt.Sprintf(`
resource "netcup_scp_user_firewall_policy" "test" {
  user_id     = 12345
  name        = "test-policy"
  description = %q
  rules = [
    {
      action      = %q
      direction   = %q
      protocol    = %q
%s%s%s    }
  ]
}
`, desc, action, direction, protocol, sourcesAttr, destinationsAttr, portsAttr)
}

func mapPortsAttr(ingress bool) string {
	if ingress {
		return "destination"
	}
	return "source"
}

type firewallPolicyMock struct {
	mu          sync.Mutex
	description string
	rules       []map[string]any
}

func (m *firewallPolicyMock) normalizeRules(in []any) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, r := range in {
		rule, ok := r.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, rule)
	}
	return out
}

func (m *firewallPolicyMock) store(description string, rules []any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.description = description
	m.rules = m.normalizeRules(rules)
}

func (m *firewallPolicyMock) load() (string, []map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.description, m.rules
}

func (m *firewallPolicyMock) handleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	desc, _ := payload["description"].(string)
	rules, _ := payload["rules"].([]any)
	m.store(desc, rules)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(m.policyResponse()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *firewallPolicyMock) handleReadOrUpdate(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(m.policyResponse()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	case http.MethodPut:
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer func() { _ = r.Body.Close() }()

		desc, _ := payload["description"].(string)
		rules, _ := payload["rules"].([]any)
		m.store(desc, rules)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		if err := json.NewEncoder(w).Encode(map[string]any{"uuid": "task-fw-1", "state": taskStateFinished}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	case http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *firewallPolicyMock) policyResponse() map[string]any {
	desc, rules := m.load()
	return map[string]any{
		"id":          1,
		"name":        "test-policy",
		"description": desc,
		"rules":       rules,
	}
}

func newFirewallPolicyMockServer(t *testing.T) *httptest.Server {
	t.Helper()

	mock := &firewallPolicyMock{
		description: "first",
		rules: []map[string]any{
			{
				"action":           "ACCEPT",
				"direction":        "INGRESS",
				"protocol":         "TCP",
				"sources":          []any{"0.0.0.0/0"},
				"destinationPorts": "80",
			},
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/scp-core/api/v1/users/12345/firewall-policies", mock.handleCreate)
	mux.HandleFunc("/scp-core/api/v1/users/12345/firewall-policies/1", mock.handleReadOrUpdate)

	return httptest.NewServer(mux)
}
