package scpmaintenance_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/rixlhq/terraform-provider-netcup/internal/provider/scp/scpmaintenance"
)

func TestScpMaintenanceDataSource_SchemaDeprecation(t *testing.T) {
	t.Parallel()

	d := scpmaintenance.NewScpMaintenanceDataSource()
	var resp datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	if resp.Schema.DeprecationMessage == "" {
		t.Fatal("expected DeprecationMessage to be set on scp_maintenance data source schema")
	}
	if !strings.Contains(resp.Schema.DeprecationMessage, "31.12.2026") {
		t.Fatalf("expected DeprecationMessage to mention 31.12.2026, got: %s", resp.Schema.DeprecationMessage)
	}
}
