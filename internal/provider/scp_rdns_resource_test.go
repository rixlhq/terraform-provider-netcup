package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/rixlhq/terraform-provider-netcup/internal/provider"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"netcup": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func TestAccScpRdns_basic(t *testing.T) {
	server := newSCPMockServer(t)
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
resource "netcup_scp_rdns" "test" {
  ip_version = "ipv4"
  ip         = "1.2.3.4"
  rdns       = "test.example.com"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_rdns.test", "ip", "1.2.3.4"),
					resource.TestCheckResourceAttr("netcup_scp_rdns.test", "rdns", "test.example.com"),
				),
			},
			{
				Config: providerConfig + `
resource "netcup_scp_rdns" "test" {
  ip_version = "ipv4"
  ip         = "1.2.3.4"
  rdns       = "updated.example.com"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_rdns.test", "rdns", "updated.example.com"),
				),
			},
		},
	})
}

type rdnsState struct {
	IP   string `json:"ip"`
	RDNS string `json:"rdns"`
}

func newSCPMockServer(t *testing.T) *httptest.Server {
	t.Helper()

	state := &rdnsState{IP: "1.2.3.4", RDNS: "test.example.com"}

	mux := http.NewServeMux()

	mux.HandleFunc("/scp-core/api/v1/rdns/ipv4", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var payload rdnsState
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer func() { _ = r.Body.Close() }()

			state.RDNS = payload.RDNS
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(state)
		default:
			http.Error(w, "not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/scp-core/api/v1/rdns/ipv4/1.2.3.4", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(state)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "not allowed", http.StatusMethodNotAllowed)
		}
	})

	return httptest.NewServer(mux)
}
