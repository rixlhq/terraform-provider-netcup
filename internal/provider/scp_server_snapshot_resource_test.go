package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccScpServerSnapshot_basic(t *testing.T) {
	server := newSnapshotMockServer(t)
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
resource "netcup_scp_server_snapshot" "test" {
  server_id       = 12345
  name            = "test-snap"
  description     = "first"
  disk_name       = "vda"
  online_snapshot = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_server_snapshot.test", "server_id", "12345"),
					resource.TestCheckResourceAttr("netcup_scp_server_snapshot.test", "name", "test-snap"),
					resource.TestCheckResourceAttr("netcup_scp_server_snapshot.test", "description", "first"),
					resource.TestCheckResourceAttr("netcup_scp_server_snapshot.test", "uuid", "snap-1"),
					resource.TestCheckResourceAttr("netcup_scp_server_snapshot.test", "state", "AVAILABLE"),
				),
			},
			{
				Config: providerConfig + `
resource "netcup_scp_server_snapshot" "test" {
  server_id       = 12345
  name            = "test-snap"
  description     = "second"
  disk_name       = "vda"
  online_snapshot = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_server_snapshot.test", "description", "second"),
				),
			},
		},
	})
}

type snapshotSnapshot struct {
	ServerID       int64           `json:"serverId"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	DiskName       string          `json:"diskName"`
	OnlineSnapshot bool            `json:"onlineSnapshot"`
	UUID           string          `json:"uuid"`
	Disks          []string        `json:"disks"`
	CreationTime   string          `json:"creationTime"`
	State          string          `json:"state"`
	Online         bool            `json:"online"`
	Exported       bool            `json:"exported"`
	ExportedSize   int             `json:"exportedSizeInKiB"`
	DownloadInfos  json.RawMessage `json:"downloadInfos"`
}

type snapshotTaskResponse struct {
	UUID  string `json:"uuid"`
	State string `json:"state"`
}

type snapshotMock struct {
	snap snapshotSnapshot
}

func (m *snapshotMock) handleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload snapshotSnapshot
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	m.snap.Description = payload.Description
	m.snap.DiskName = payload.DiskName
	m.snap.OnlineSnapshot = payload.OnlineSnapshot

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(snapshotTaskResponse{UUID: "task-snapshot-1", State: taskStateFinished}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *snapshotMock) handleReadOrDelete(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(m.snap); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	case http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
	}
}

func newSnapshotMockServer(t *testing.T) *httptest.Server {
	t.Helper()

	mock := &snapshotMock{snap: snapshotSnapshot{
		ServerID:       12345,
		Name:           "test-snap",
		Description:    "first",
		DiskName:       "vda",
		OnlineSnapshot: true,
		UUID:           "snap-1",
		Disks:          []string{"vda"},
		CreationTime:   "2026-01-01T00:00:00Z",
		State:          "AVAILABLE",
		Online:         true,
		Exported:       false,
		ExportedSize:   0,
		DownloadInfos:  nil,
	}}

	mux := http.NewServeMux()
	mux.HandleFunc("/scp-core/api/v1/servers/12345/snapshots", mock.handleCreate)
	mux.HandleFunc("/scp-core/api/v1/servers/12345/snapshots/test-snap", mock.handleReadOrDelete)

	return httptest.NewServer(mux)
}
