package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccScpUserSshKey_basic(t *testing.T) {
	server := newSshKeyMockServer(t)
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
resource "netcup_scp_user_ssh_key" "test" {
  user_id = 12345
  name    = "test-key"
  key     = "ssh-rsa AAAAB3NzaC1 test@example.com"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_user_ssh_key.test", "user_id", "12345"),
					resource.TestCheckResourceAttr("netcup_scp_user_ssh_key.test", "name", "test-key"),
					resource.TestCheckResourceAttr("netcup_scp_user_ssh_key.test", "id", "1"),
				),
			},
		},
	})
}

type sshKey struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"userId"`
	Name      string `json:"name"`
	Key       string `json:"key"`
	CreatedAt string `json:"createdAt"`
}

type sshKeyMock struct {
	keys []sshKey
}

func (m *sshKeyMock) handleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload sshKey
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	payload.ID = 1
	payload.UserID = 12345
	payload.CreatedAt = "2026-01-01T00:00:00Z"
	m.keys = []sshKey{payload}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *sshKeyMock) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(m.keys); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *sshKeyMock) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.keys = nil
	w.WriteHeader(http.StatusNoContent)
}

func newSshKeyMockServer(t *testing.T) *httptest.Server {
	t.Helper()

	mock := &sshKeyMock{}
	mux := http.NewServeMux()
	mux.HandleFunc("/scp-core/api/v1/users/12345/ssh-keys", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			mock.handleCreate(w, r)
			return
		}
		mock.handleList(w, r)
	})
	mux.HandleFunc("/scp-core/api/v1/users/12345/ssh-keys/1", mock.handleDelete)

	return httptest.NewServer(mux)
}
