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

func TestAccScpServerAction_basic(t *testing.T) {
	server := newServerActionMockServer(t)
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
resource "netcup_scp_server_action" "test" {
  server_id = 12345
  action    = "start"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_server_action.test", "server_id", "12345"),
					resource.TestCheckResourceAttr("netcup_scp_server_action.test", "action", "start"),
					resource.TestCheckResourceAttrSet("netcup_scp_server_action.test", "id"),
				),
			},
			{
				Config: providerConfig + `
resource "netcup_scp_server_action" "test" {
  server_id = 12345
  action    = "start"
  triggers  = { run = "second" }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_server_action.test", "triggers.run", "second"),
					resource.TestCheckResourceAttrSet("netcup_scp_server_action.test", "id"),
				),
			},
		},
	})
}

// TestAccScpServerAction_rescueActivate covers an action whose request has no
// body. The real SCP API rejects such requests with HTTP 400
// error.content.type.invalid when the Content-Type header is missing, so the
// mock does the same.
func TestAccScpServerAction_rescueActivate(t *testing.T) {
	server := newServerActionMockServer(t)
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
resource "netcup_scp_server_action" "test" {
  server_id = 12345
  action    = "rescue_activate"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_server_action.test", "action", "rescue_activate"),
					resource.TestCheckResourceAttrSet("netcup_scp_server_action.test", "id"),
				),
			},
		},
	})
}

type serverActionMock struct {
	mu      sync.Mutex
	actions []string
}

func (m *serverActionMock) handleAction(w http.ResponseWriter, r *http.Request) {
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

	m.mu.Lock()
	m.actions = append(m.actions, fmt.Sprintf("%s: %v", strings.ToLower(r.URL.Query().Get("stateOption")), payload["state"]))
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(map[string]any{"uuid": "task-action-1", "state": taskStateFinished}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleRescue mirrors POST /api/v1/servers/{id}/rescuesystem, which takes no
// request body but still requires a Content-Type header.
func (m *serverActionMock) handleRescue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Header.Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"error.content.type.invalid","message":"The HTTP header Content-Type must be 'type/subtype'"}`))
		return
	}

	m.mu.Lock()
	m.actions = append(m.actions, "rescue_activate")
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(map[string]any{"uuid": "task-rescue-1", "state": taskStateFinished}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func newServerActionMockServer(t *testing.T) *httptest.Server {
	t.Helper()

	mock := &serverActionMock{}
	mux := http.NewServeMux()
	mux.HandleFunc("/scp-core/api/v1/servers/12345", mock.handleAction)
	mux.HandleFunc("/scp-core/api/v1/servers/12345/rescuesystem", mock.handleRescue)

	return httptest.NewServer(mux)
}
