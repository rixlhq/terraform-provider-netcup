package scpservers_test

import (
	"context"
	"testing"

	"github.com/rixlhq/terraform-provider-netcup/internal/provider/scp/scpservers"
	"github.com/rixlhq/terraform-provider-netcup/internal/provider/scpcommon"
)

func TestJSONToTfValueServers(t *testing.T) {
	ctx := context.Background()
	schema := scpservers.ScpServersDataSourceSchema(ctx)
	tfType := schema.Type().TerraformType(ctx)

	body := []byte(`[
		{"id": 123, "name": "server1", "hostname": "srv1.example.com", "disabled": false},
		{"id": 456, "name": "server2", "hostname": null, "disabled": true}
	]`)

	jsonVal, err := scpcommon.DecodeJSONResponse(body)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if arr, ok := jsonVal.([]any); ok {
		jsonVal = map[string]any{"scp_servers": arr}
	}

	_, err = scpcommon.JSONToTfValue(ctx, tfType, jsonVal)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
}
