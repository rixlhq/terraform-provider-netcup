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

func TestAccScpUser_basic(t *testing.T) {
	server := newScpUserMockServer(t)
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
				Config: scpUserConfig(providerConfig, "de", "Europe/Berlin", true, false, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_user.test", "user_id", "12345"),
					resource.TestCheckResourceAttr("netcup_scp_user.test", "language", "de"),
					resource.TestCheckResourceAttr("netcup_scp_user.test", "time_zone", "Europe/Berlin"),
					resource.TestCheckResourceAttr("netcup_scp_user.test", "show_nickname", "true"),
					resource.TestCheckResourceAttr("netcup_scp_user.test", "username", "max"),
					resource.TestCheckResourceAttr("netcup_scp_user.test", "id", "12345"),
				),
			},
			{
				Config: scpUserConfig(providerConfig, "en", "UTC", false, true, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_user.test", "language", "en"),
					resource.TestCheckResourceAttr("netcup_scp_user.test", "time_zone", "UTC"),
					resource.TestCheckResourceAttr("netcup_scp_user.test", "show_nickname", "false"),
					resource.TestCheckResourceAttr("netcup_scp_user.test", "passwordless_mode", "true"),
				),
			},
		},
	})
}

func scpUserConfig(providerConfig, lang, tz string, show, passless, secure bool) string {
	return providerConfig + fmt.Sprintf(`
resource "netcup_scp_user" "test" {
  user_id           = 12345
  language          = %q
  time_zone         = %q
  show_nickname     = %t
  passwordless_mode = %t
  secure_mode       = %t
}
`, lang, tz, show, passless, secure)
}

type scpUserMock struct {
	mu       sync.Mutex
	language string
	timeZone string
	showNick bool
	passless bool
	secure   bool
}

type scpUserSavePayload struct {
	Language         string `json:"language"`
	TimeZone         string `json:"timeZone"`
	ShowNickname     bool   `json:"showNickname"`
	PasswordlessMode bool   `json:"passwordlessMode"`
	SecureMode       bool   `json:"secureMode"`
}

func (m *scpUserMock) updateState(p scpUserSavePayload) scpUserSavePayload {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.language = p.Language
	m.timeZone = p.TimeZone
	m.showNick = p.ShowNickname
	m.passless = p.PasswordlessMode
	m.secure = p.SecureMode
	return scpUserSavePayload{
		Language:         m.language,
		TimeZone:         m.timeZone,
		ShowNickname:     m.showNick,
		PasswordlessMode: m.passless,
		SecureMode:       m.secure,
	}
}

func (m *scpUserMock) current() scpUserSavePayload {
	m.mu.Lock()
	defer m.mu.Unlock()
	return scpUserSavePayload{
		Language:         m.language,
		TimeZone:         m.timeZone,
		ShowNickname:     m.showNick,
		PasswordlessMode: m.passless,
		SecureMode:       m.secure,
	}
}

func (m *scpUserMock) handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload scpUserSavePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	state := m.updateState(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"id":               12345,
		"username":         "max",
		"firstname":        "Max",
		"lastname":         "M",
		"email":            "max@example.com",
		"language":         state.Language,
		"timeZone":         state.TimeZone,
		"showNickname":     state.ShowNickname,
		"passwordlessMode": state.PasswordlessMode,
		"secureMode":       state.SecureMode,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (m *scpUserMock) handleRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	state := m.current()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"id":               12345,
		"username":         "max",
		"firstname":        "Max",
		"lastname":         "M",
		"email":            "max@example.com",
		"language":         state.Language,
		"timeZone":         state.TimeZone,
		"showNickname":     state.ShowNickname,
		"passwordlessMode": state.PasswordlessMode,
		"secureMode":       state.SecureMode,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func newScpUserMockServer(t *testing.T) *httptest.Server {
	t.Helper()

	mock := &scpUserMock{
		language: "de",
		timeZone: "Europe/Berlin",
		showNick: true,
		passless: false,
		secure:   false,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/scp-core/api/v1/users/12345", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			mock.handleUpdate(w, r)
			return
		}
		mock.handleRead(w, r)
	})

	return httptest.NewServer(mux)
}
