package provider_test

import (
	"fmt"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCloudDNSCreateRejectsUnknownRevision(t *testing.T) {
	mock := newCloudDNSMock(t)
	mock.revisionState = "not-documented"
	server := httptest.NewServer(mock)
	defer server.Close()
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{
		Config: cloudDNSRecordConfig(server.URL, "192.0.2.1"), ExpectError: regexp.MustCompile("active DNS revision"),
	}}})
	if mock.writes != 0 {
		t.Fatalf("performed %d writes with unknown revision state", mock.writes)
	}
}

func TestAccCloudDNSCreateRecoversReadFailure(t *testing.T) {
	mock := newCloudDNSMock(t)
	mock.failReadAfterWrite = true
	server := httptest.NewServer(mock)
	defer server.Close()
	config := cloudDNSRecordConfig(server.URL, "192.0.2.1")
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config, ExpectError: regexp.MustCompile("read failed")},
		{Config: config, Check: resource.TestCheckResourceAttr("netcup_clouddns_record.test", "rdata", "192.0.2.1")},
	}})
	// Terraform taints the failed create, then safely deletes and recreates it.
	if mock.writes != 4 {
		t.Fatalf("changesets=%d, want initial create, replacement delete/create, and cleanup", mock.writes)
	}
	mock.assertUnmanaged(t)
}

func TestAccCloudDNSRecordIdentityAcrossRevisions(t *testing.T) {
	mock := newCloudDNSMock(t)
	server := httptest.NewServer(mock)
	defer server.Close()
	config := cloudDNSRecordConfig(server.URL, "192.0.2.1")
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: config},
		{Config: config, PreConfig: func() { mock.mu.Lock(); defer mock.mu.Unlock(); mock.records[1][cloudDNSIdentifier] = "copied-record" }, PlanOnly: true},
	}})
	mock.assertUnmanaged(t)
}

func TestAccCloudDNSImmutableRecord(t *testing.T) {
	mock := newCloudDNSMock(t)
	mock.records = append(mock.records, map[string]any{cloudDNSIdentifier: "immutable", "name": "www", "type": "A", "rdata": "192.0.2.1", "ttl": int64(300), "immutable": true})
	server := httptest.NewServer(mock)
	defer server.Close()
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{{
		Config: cloudDNSRecordConfig(server.URL, "192.0.2.1"), ExpectError: regexp.MustCompile("already exists"),
	}}})
	if mock.writes != 0 {
		t.Fatalf("wrote changeset for existing immutable record")
	}
}

func cloudDNSRecordConfig(baseURL, address string) string {
	return fmt.Sprintf(`provider "netcup" {
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
`, baseURL, address)
}

func TestAccCloudDNSUpdateRecoversReadFailure(t *testing.T) {
	mock := newCloudDNSMock(t)
	server := httptest.NewServer(mock)
	defer server.Close()
	next := cloudDNSRecordConfig(server.URL, "192.0.2.2")
	resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{
		{Config: cloudDNSRecordConfig(server.URL, "192.0.2.1")},
		{Config: next, PreConfig: func() {
			mock.mu.Lock()
			defer mock.mu.Unlock()
			mock.failReadAfterWrite = true
			mock.failAfterWrites = 2
		}, ExpectError: regexp.MustCompile("read failed")},
		{Config: next, PlanOnly: true},
	}})
	if mock.writes != 3 {
		t.Fatalf("changesets=%d, want create, update and cleanup", mock.writes)
	}
	mock.assertUnmanaged(t)
}
