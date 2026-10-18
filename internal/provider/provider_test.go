package provider_test

import (
	"context"
	"testing"

	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/rixlhq/terraform-provider-netcup/internal/provider"
)

func TestProviderCredentials(t *testing.T) {
	tests := map[string]struct {
		cloudKey     any
		accessToken  any
		refreshToken any
		wantErr      bool
	}{
		"CloudDNS key only configures provider": {cloudKey: "cloud-key"},
		"refresh token only configures SCP": {
			refreshToken: "refresh-token",
		},
		"access token only configures SCP": {
			accessToken: "access-token",
		},
		"no credentials": {wantErr: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			p := provider.New("test")()
			var schemaResponse frameworkprovider.SchemaResponse
			p.Schema(ctx, frameworkprovider.SchemaRequest{}, &schemaResponse)

			raw := tftypes.NewValue(schemaResponse.Schema.Type().TerraformType(ctx), map[string]tftypes.Value{
				"clouddns_api_key":  stringValue(test.cloudKey),
				"clouddns_base_url": stringValue(nil),
				"api_key":           stringValue(nil),
				"api_password":      stringValue(nil),
				"customer_number":   stringValue(nil),
				"endpoint":          stringValue(nil),
				"scp_access_token":  stringValue(test.accessToken),
				"scp_refresh_token": stringValue(test.refreshToken),
				"scp_base_url":      stringValue(nil),
			})

			var response frameworkprovider.ConfigureResponse
			p.Configure(ctx, frameworkprovider.ConfigureRequest{
				Config: tfsdk.Config{
					Raw:    raw,
					Schema: schemaResponse.Schema,
				},
			}, &response)

			if response.Diagnostics.HasError() != test.wantErr {
				t.Fatalf("provider configuration error = %v, want %v: %v", response.Diagnostics.HasError(), test.wantErr, response.Diagnostics)
			}
		})
	}
}

func stringValue(value any) tftypes.Value {
	if value == nil {
		return tftypes.NewValue(tftypes.String, nil)
	}
	return tftypes.NewValue(tftypes.String, value)
}
