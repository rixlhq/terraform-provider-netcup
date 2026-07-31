package provider_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-netcup/internal/provider"
)

func TestIPVersionFunction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"ipv4", "1.2.3.4", "ipv4", false},
		{"ipv6", "2001:db8::1", "ipv6", false},
		{"ipv4 mapped", "::ffff:192.0.2.1", "ipv4", false},
		{"invalid", "not-an-ip", "", true},
	}

	f := provider.NewIPVersionFunction()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{types.StringValue(tt.input)}),
			}
			resp := function.RunResponse{
				Result: function.NewResultData(types.StringNull()),
			}

			f.Run(context.Background(), req, &resp)

			if tt.wantErr {
				if resp.Error == nil {
					t.Fatalf("expected an error for %q", tt.input)
				}
				return
			}

			if resp.Error != nil {
				t.Fatalf("unexpected error: %s", resp.Error.Error())
			}

			got, ok := resp.Result.Value().(types.String)
			if !ok {
				t.Fatalf("expected result to be types.String, got %T", resp.Result.Value())
			}

			if got.ValueString() != tt.want {
				t.Errorf("ip_version(%q) = %q, want %q", tt.input, got.ValueString(), tt.want)
			}
		})
	}
}
