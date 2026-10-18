//nolint:testpackage // Exercise path construction shared by SCP CRUD resources.
package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/rixlhq/terraform-provider-netcup/internal/provider/scp/scpserversnapshot"
	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

const snapshotSpecialName = "backup ?#/\u96ea%"

func TestCRUDPathEscapesParameters(t *testing.T) {
	t.Parallel()
	r := &scpCrudResource{spec: scpServerSnapshotSpec}
	typ := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"server_id": tftypes.Number, "name": tftypes.String}}
	value := tftypes.NewValue(typ, map[string]tftypes.Value{"server_id": tftypes.NewValue(tftypes.Number, 12345), "name": tftypes.NewValue(tftypes.String, snapshotSpecialName)})
	got, err := r.buildPath(value, r.spec.readPath)
	if err != nil {
		t.Fatal(err)
	}
	if want := snapshotEncodedPath(); got != want {
		t.Fatalf("path=%q, want %q", got, want)
	}
}

func TestSnapshotDataSourceEscapesName(t *testing.T) {
	t.Parallel()
	captured := make(chan string, 1)
	server := snapshotPathServer(t, captured)
	defer server.Close()
	source := scpserversnapshot.NewScpServerSnapshotDataSource()
	var schemaResponse datasource.SchemaResponse
	source.Schema(t.Context(), datasource.SchemaRequest{}, &schemaResponse)
	typ := schemaResponse.Schema.Type().TerraformType(t.Context()).(tftypes.Object)
	values := make(map[string]tftypes.Value, len(typ.AttributeTypes))
	for name, attrType := range typ.AttributeTypes {
		values[name] = tftypes.NewValue(attrType, nil)
	}
	values["server_id"] = tftypes.NewValue(tftypes.Number, 12345)
	values["name"] = tftypes.NewValue(tftypes.String, snapshotSpecialName)
	var configured datasource.ConfigureResponse
	source.(datasource.DataSourceWithConfigure).Configure(t.Context(), datasource.ConfigureRequest{ProviderData: &providerData{SCP: scpclient.New("token", "", server.URL, server.Client())}}, &configured)
	response := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
	source.Read(t.Context(), datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResponse.Schema, Raw: tftypes.NewValue(typ, values)}}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	if got := <-captured; got != snapshotEncodedPath() {
		t.Fatalf("request URI=%q, want %q", got, snapshotEncodedPath())
	}
}

func TestSnapshotImportPreservesNameWithSlash(t *testing.T) {
	t.Parallel()
	captured := make(chan string, 1)
	server := snapshotPathServer(t, captured)
	defer server.Close()
	r := NewScpServerSnapshotResource().(*scpCrudResource)
	r.client = scpclient.New("token", "", server.URL, server.Client())
	response := resource.ImportStateResponse{State: tfsdk.State{Schema: r.schema}}
	r.ImportState(t.Context(), resource.ImportStateRequest{ID: "12345/" + snapshotSpecialName}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	if got := <-captured; got != snapshotEncodedPath() {
		t.Fatalf("request URI=%q, want %q", got, snapshotEncodedPath())
	}
	var name string
	if diags := response.State.GetAttribute(t.Context(), path.Root("name"), &name); diags.HasError() {
		t.Fatal(diags)
	}
	if name != snapshotSpecialName {
		t.Fatalf("imported name=%q", name)
	}
}

func snapshotEncodedPath() string {
	return "/api/v1/servers/12345/snapshots/" + url.PathEscape(snapshotSpecialName)
}

func snapshotPathServer(t *testing.T, captured chan<- string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured <- r.RequestURI
		if r.Method != http.MethodGet {
			t.Errorf("unexpected write: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"name": snapshotSpecialName, "uuid": "snapshot-uuid", "state": "OFF", "online": false, "exported": false}); err != nil {
			t.Error(err)
		}
	}))
}
