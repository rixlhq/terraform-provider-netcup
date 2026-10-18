//nolint:testpackage
package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestApplyEnvOverrides(t *testing.T) {
	t.Setenv("NETCUP_CLOUDDNS_API_KEY", "env-cloud-key")
	t.Setenv("NETCUP_CLOUDDNS_BASE_URL", "env-cloud-url")
	t.Setenv("NETCUP_SCP_ACCESS_TOKEN", "env-access-token")
	t.Setenv("NETCUP_SCP_REFRESH_TOKEN", "env-refresh-token")
	t.Setenv("NETCUP_SCP_BASE_URL", "env-base-url")

	data := NetcupProviderModel{
		CloudDNSAPIKey:  types.StringValue("config-cloud-key"),
		CloudDNSBaseURL: types.StringNull(),
		SCPAccessToken:  types.StringNull(),
		SCPRefreshToken: types.StringValue("config-refresh-token"),
		SCPBaseURL:      types.StringNull(),
	}

	got := applyEnvOverrides(data)

	if got.CloudDNSAPIKey.ValueString() != "config-cloud-key" {
		t.Errorf("clouddns_api_key: expected config value to be retained, got %q", got.CloudDNSAPIKey.ValueString())
	}
	if got.CloudDNSBaseURL.ValueString() != "env-cloud-url" {
		t.Errorf("clouddns_base_url: expected env value, got %q", got.CloudDNSBaseURL.ValueString())
	}
	if got.SCPAccessToken.ValueString() != "env-access-token" {
		t.Errorf("scp_access_token: expected env value, got %q", got.SCPAccessToken.ValueString())
	}
	if got.SCPRefreshToken.ValueString() != "config-refresh-token" {
		t.Errorf("scp_refresh_token: expected config value to be retained, got %q", got.SCPRefreshToken.ValueString())
	}
	if got.SCPBaseURL.ValueString() != "env-base-url" {
		t.Errorf("scp_base_url: expected env value, got %q", got.SCPBaseURL.ValueString())
	}
}

func TestApplyEnvOverrides_noEnv(t *testing.T) {
	data := NetcupProviderModel{
		CloudDNSAPIKey: types.StringValue("config-cloud-key"),
	}

	got := applyEnvOverrides(data)

	if got.CloudDNSAPIKey.ValueString() != "config-cloud-key" {
		t.Errorf("clouddns_api_key: expected config value to be retained, got %q", got.CloudDNSAPIKey.ValueString())
	}
}

func TestNewClients_rejectsEmptyStrings(t *testing.T) {
	empty := NetcupProviderModel{
		CloudDNSAPIKey:  types.StringValue(""),
		SCPAccessToken:  types.StringValue(""),
		SCPRefreshToken: types.StringValue(""),
	}

	if _, hasCloud, _ := newCloudDNSClient(empty); hasCloud {
		t.Errorf("newCloudDNSClient: empty strings must not count as configured")
	}
	if _, hasSCP := newSCPClient(empty); hasSCP {
		t.Errorf("newSCPClient: empty strings must not count as configured")
	}

	configured := NetcupProviderModel{
		CloudDNSAPIKey:  types.StringValue("cloud-key"),
		SCPRefreshToken: types.StringValue("refresh"),
	}
	if _, hasCloud, _ := newCloudDNSClient(configured); !hasCloud {
		t.Errorf("newCloudDNSClient: non-empty cloud key must count as configured")
	}
	if _, hasSCP := newSCPClient(configured); !hasSCP {
		t.Errorf("newSCPClient: non-empty refresh token must count as configured")
	}
}
