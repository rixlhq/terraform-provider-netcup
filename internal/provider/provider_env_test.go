//nolint:testpackage
package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestApplyEnvOverrides(t *testing.T) {
	t.Setenv("NETCUP_API_KEY", "env-api-key")
	t.Setenv("NETCUP_API_PASSWORD", "env-api-password")
	t.Setenv("NETCUP_CUSTOMER_NUMBER", "env-customer")
	t.Setenv("NETCUP_ENDPOINT", "env-endpoint")
	t.Setenv("NETCUP_SCP_ACCESS_TOKEN", "env-access-token")
	t.Setenv("NETCUP_SCP_REFRESH_TOKEN", "env-refresh-token")
	t.Setenv("NETCUP_SCP_BASE_URL", "env-base-url")

	data := NetcupProviderModel{
		APIKey:          types.StringValue("config-api-key"),
		APIPassword:     types.StringNull(),
		CustomerNumber:  types.StringNull(),
		Endpoint:        types.StringValue("config-endpoint"),
		SCPAccessToken:  types.StringNull(),
		SCPRefreshToken: types.StringValue("config-refresh-token"),
		SCPBaseURL:      types.StringNull(),
	}

	got := applyEnvOverrides(data)

	if got.APIKey.ValueString() != "config-api-key" {
		t.Errorf("api_key: expected config value to be retained, got %q", got.APIKey.ValueString())
	}
	if got.APIPassword.ValueString() != "env-api-password" {
		t.Errorf("api_password: expected env value, got %q", got.APIPassword.ValueString())
	}
	if got.CustomerNumber.ValueString() != "env-customer" {
		t.Errorf("customer_number: expected env value, got %q", got.CustomerNumber.ValueString())
	}
	if got.Endpoint.ValueString() != "config-endpoint" {
		t.Errorf("endpoint: expected config value to be retained, got %q", got.Endpoint.ValueString())
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
		APIKey: types.StringValue("config-api-key"),
	}

	got := applyEnvOverrides(data)

	if got.APIKey.ValueString() != "config-api-key" {
		t.Errorf("api_key: expected config value to be retained, got %q", got.APIKey.ValueString())
	}
}
