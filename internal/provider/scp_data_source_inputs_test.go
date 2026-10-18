package provider_test

import (
	"math/big"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/rixlhq/terraform-provider-netcup/internal/provider"
	"github.com/rixlhq/terraform-provider-netcup/internal/providerdata"
	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

func TestSCPDataSourcesPreserveInputs(t *testing.T) {
	t.Parallel()
	p := provider.New("test")()
	for _, factory := range p.DataSources(t.Context()) {
		source := factory()
		var metadata datasource.MetadataResponse
		source.Metadata(t.Context(), datasource.MetadataRequest{ProviderTypeName: "netcup"}, &metadata)
		if !strings.HasPrefix(metadata.TypeName, "netcup_scp_") {
			continue
		}
		t.Run(metadata.TypeName, func(t *testing.T) { t.Parallel(); checkSCPDataSourceInputs(t, source) })
	}
}

func checkSCPDataSourceInputs(t *testing.T, source datasource.DataSource) {
	t.Helper()
	var schemaResponse datasource.SchemaResponse
	source.Schema(t.Context(), datasource.SchemaRequest{}, &schemaResponse)
	tfType := schemaResponse.Schema.Type().TerraformType(t.Context()).(tftypes.Object)
	values := make(map[string]tftypes.Value, len(tfType.AttributeTypes))
	inputs := make(map[string]tftypes.Value)
	for name, typ := range tfType.AttributeTypes {
		values[name] = tftypes.NewValue(typ, nil)
		attribute := schemaResponse.Schema.GetAttributes()[name]
		if attribute.IsRequired() || attribute.IsOptional() {
			values[name] = dataSourceInputValue(t, name, typ)
			inputs[name] = values[name]
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("data source made a write: %s", r.Method)
		}
		if r.URL.Path == "/api/v1/servers" && !reflect.DeepEqual(r.URL.Query()["sort"], []string{"name", "-nickname"}) {
			t.Errorf("sort parameters were not forwarded: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hostname":"api.example.com"}`))
	}))
	defer server.Close()
	var configure datasource.ConfigureResponse
	source.(datasource.DataSourceWithConfigure).Configure(t.Context(), datasource.ConfigureRequest{ProviderData: &providerdata.Data{SCP: scpclient.New("test-token", "", server.URL, server.Client())}}, &configure)
	if configure.Diagnostics.HasError() {
		t.Fatal(configure.Diagnostics)
	}
	config := tfsdk.Config{Schema: schemaResponse.Schema, Raw: tftypes.NewValue(tfType, values)}
	response := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
	source.Read(t.Context(), datasource.ReadRequest{Config: config}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	var got map[string]tftypes.Value
	if err := response.State.Raw.As(&got); err != nil {
		t.Fatal(err)
	}
	for name, want := range inputs {
		if !got[name].Equal(want) {
			t.Errorf("input %s lost: got %s, want %s", name, got[name], want)
		}
	}
}

func dataSourceInputValue(t *testing.T, name string, typ tftypes.Type) tftypes.Value {
	t.Helper()
	switch {
	case typ.Is(tftypes.String):
		value := "input-value"
		if name == "metric" {
			value = "cpu"
		}
		return tftypes.NewValue(typ, value)
	case typ.Is(tftypes.Number):
		return tftypes.NewValue(typ, big.NewFloat(123))
	case typ.Is(tftypes.Bool):
		return tftypes.NewValue(typ, false)
	case typ.Is(tftypes.List{ElementType: tftypes.String}):
		return tftypes.NewValue(typ, []tftypes.Value{tftypes.NewValue(tftypes.String, "name"), tftypes.NewValue(tftypes.String, "-nickname")})
	default:
		t.Fatalf("unsupported input %s: %s", name, typ)
		return tftypes.Value{}
	}
}
