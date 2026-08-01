package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccScpTaskAction_basic(t *testing.T) {
	server := newTaskActionMockServer(t)
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
resource "netcup_scp_task_action" "test" {
  task_uuid = "task-1"
  action    = "cancel"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_task_action.test", "task_uuid", "task-1"),
					resource.TestCheckResourceAttr("netcup_scp_task_action.test", "action", "cancel"),
					resource.TestCheckResourceAttrSet("netcup_scp_task_action.test", "id"),
				),
			},
			{
				Config: providerConfig + `
resource "netcup_scp_task_action" "test" {
  task_uuid = "task-1"
  action    = "cancel"
  triggers  = { run = "second" }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_task_action.test", "triggers.run", "second"),
					resource.TestCheckResourceAttrSet("netcup_scp_task_action.test", "id"),
				),
			},
		},
	})
}

type taskActionMock struct{}

func (m *taskActionMock) handleCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(map[string]any{"uuid": "task-1", "state": taskStateFinished}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func newTaskActionMockServer(t *testing.T) *httptest.Server {
	t.Helper()

	mock := &taskActionMock{}
	mux := http.NewServeMux()
	mux.HandleFunc("/scp-core/api/v1/tasks/task-1:cancel", mock.handleCancel)

	return httptest.NewServer(mux)
}
