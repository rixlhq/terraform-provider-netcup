package scpserver

import (
	"context"
	"testing"

	"github.com/rixlhq/terraform-provider-netcup/internal/provider/scpcommon"
)

func TestJSONToTfValueServer(t *testing.T) {
	ctx := context.Background()
	schema := ScpServerDataSourceSchema(ctx)
	tfType := schema.Type().TerraformType(ctx)

	body := []byte(`{
		"id": 123,
		"name": "myserver",
		"hostname": "srv.example.com",
		"disabled": false,
		"ipv4Addresses": [
			{"id": 1, "ip": "1.2.3.4", "gateway": "1.2.3.1", "netmask": "255.255.255.0", "broadcast": "1.2.3.255"}
		]
	}`)

	jsonVal, err := scpcommon.DecodeJSONResponse(body)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	_, err = scpcommon.JSONToTfValue(ctx, tfType, jsonVal)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
}
