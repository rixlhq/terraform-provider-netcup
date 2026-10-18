package provider

import (
	"os"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rixlhq/terraform-provider-netcup/internal/clouddnsclient"
	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

func applyEnvOverrides(data NetcupProviderModel) NetcupProviderModel {
	data.CloudDNSAPIKey = envOrString(data.CloudDNSAPIKey, "NETCUP_CLOUDDNS_API_KEY")
	data.CloudDNSBaseURL = envOrString(data.CloudDNSBaseURL, "NETCUP_CLOUDDNS_BASE_URL")
	data.SCPAccessToken = envOrString(data.SCPAccessToken, "NETCUP_SCP_ACCESS_TOKEN")
	data.SCPRefreshToken = envOrString(data.SCPRefreshToken, "NETCUP_SCP_REFRESH_TOKEN")
	data.SCPBaseURL = envOrString(data.SCPBaseURL, "NETCUP_SCP_BASE_URL")
	return data
}

func envOrString(v types.String, env string) types.String {
	if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		return v
	}
	if val := os.Getenv(env); val != "" {
		return types.StringValue(val)
	}
	return v
}

// isConfigured reports whether a provider credential holds a usable value.
// Explicit empty strings must not count as configured: otherwise
// `scp_access_token = ""` would create an SCP client with an empty bearer
// token instead of falling through to the missing-credentials error.
func isConfigured(v types.String) bool {
	return !v.IsNull() && !v.IsUnknown() && v.ValueString() != ""
}

func newSCPClient(data NetcupProviderModel) (*scpclient.Client, bool) {
	hasSCP := isConfigured(data.SCPAccessToken) || isConfigured(data.SCPRefreshToken)
	if !hasSCP {
		return nil, false
	}

	baseURL := ""
	if !data.SCPBaseURL.IsNull() && !data.SCPBaseURL.IsUnknown() {
		baseURL = data.SCPBaseURL.ValueString()
	}
	return scpclient.New(
		data.SCPAccessToken.ValueString(),
		data.SCPRefreshToken.ValueString(),
		baseURL,
		nil,
	), true
}

func newCloudDNSClient(data NetcupProviderModel) (*clouddnsclient.Client, bool, error) {
	if !isConfigured(data.CloudDNSAPIKey) {
		return nil, false, nil
	}
	c, err := clouddnsclient.New(data.CloudDNSAPIKey.ValueString(), data.CloudDNSBaseURL.ValueString(), nil)
	return c, true, err
}
