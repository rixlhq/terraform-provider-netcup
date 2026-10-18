package provider_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCloudDNSRecord(t *testing.T) {
	mock := newCloudDNSMock(t)
	server := httptest.NewServer(mock)
	defer server.Close()
	config := func(address string) string {
		return fmt.Sprintf(`
provider "netcup" {
 clouddns_api_key = "cloud-key"
 clouddns_base_url = %q
}
resource "netcup_clouddns_record" "test" {
 domain_id = 42
 name = "www"
 type = "A"
 rdata = %q
 ttl = 300
}
data "netcup_clouddns_domains" "all" {}
data "netcup_clouddns_records" "all" {
 domain_id = 42
 depends_on = [netcup_clouddns_record.test]
}
`, server.URL, address)
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config("192.0.2.1"), Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("netcup_clouddns_record.test", "rdata", "192.0.2.1"),
				resource.TestCheckResourceAttr("data.netcup_clouddns_domains.all", "domains.0.fqdn", "example.com"),
				resource.TestCheckResourceAttr("data.netcup_clouddns_records.all", "records.#", "2"),
			)},
			{ResourceName: "netcup_clouddns_record.test", ImportState: true, ImportStateVerify: true},
			{ResourceName: "netcup_clouddns_record.test", ImportState: true, ImportStateId: "42/www/A/192.0.2.1", ImportStateVerify: true},
			{Config: config("192.0.2.2"), Check: resource.TestCheckResourceAttr("netcup_clouddns_record.test", "rdata", "192.0.2.2")},
			{Config: config("192.0.2.2"), PlanOnly: true},
			{Config: config("192.0.2.2"), PreConfig: mock.removeManaged, ExpectNonEmptyPlan: true, PlanOnly: true},
		},
	})
	mock.assertUnmanaged(t)
}

func TestAccCloudDNSCredentials(t *testing.T) {
	for _, config := range []string{
		`resource "netcup_clouddns_record" "test" {
 domain_id = 42
 name = "www"
 type = "A"
 rdata = "192.0.2.1"
 ttl = 300
}`,
		`data "netcup_clouddns_domains" "test" {}`,
		`data "netcup_clouddns_records" "test" {
 domain_id = 42
}`,
	} {
		t.Run(config[:12], func(t *testing.T) {
			resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{
				Config: `provider "netcup" { scp_access_token = "scp-token" }
` + config,
				PlanOnly: true, ExpectError: regexp.MustCompile("Missing CloudDNS Credentials"),
			}}})
		})
	}
}

func TestAccCloudDNSRecordValidation(t *testing.T) {
	for _, test := range []struct{ name, field, value, pattern string }{
		{"TTL", "ttl", "0", "at least 1"},
		{"domain", "domain_id", "0", "at least 1"},
		{"type", "type", `"SOA"`, "must be one of"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fields := map[string]string{"domain_id": "42", "name": `"www"`, "type": `"A"`, "rdata": `"192.0.2.1"`, "ttl": "300"}
			fields[test.field] = test.value
			config := `provider "netcup" {
 clouddns_api_key = "cloud-key"
}
resource "netcup_clouddns_record" "test" {`
			var configSb81 strings.Builder
			for _, key := range []string{"domain_id", "name", "type", "rdata", "ttl"} {
				fmt.Fprintf(&configSb81, "\n %s = %s", key, fields[key])
			}
			config += configSb81.String()
			config += "\n}"
			resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{Config: config, PlanOnly: true, ExpectError: regexp.MustCompile(test.pattern)}}})
		})
	}
}

func TestAccCloudDNSUnknownRevisionState(t *testing.T) {
	mock := newCloudDNSMock(t)
	mock.revisionState = "not-documented"
	server := httptest.NewServer(mock)
	defer server.Close()
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{
		Config: fmt.Sprintf(`provider "netcup" {
 clouddns_api_key = "cloud-key"
 clouddns_base_url = %q
}
data "netcup_clouddns_records" "test" {
 domain_id = 42
}`, server.URL),
		PlanOnly: true, ExpectError: regexp.MustCompile("active DNS revision"),
	}}})
	if mock.writes != 0 {
		t.Fatal("read performed a write")
	}
}

// Ensure the mock cannot fall through to any live netcup endpoint.
var _ http.Handler = (*cloudDNSMock)(nil)
