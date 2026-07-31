package scpserver_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/rixlhq/terraform-provider-netcup/internal/provider/scp/scpserver"
	"github.com/rixlhq/terraform-provider-netcup/internal/provider/scpcommon"
)

var scpServerJSONBody = []byte(`{
	"id": 123,
	"name": "myserver",
	"hostname": "srv.example.com",
	"disabled": false,
	"serverId": 123,
	"maxCpuCount": 8,
	"snapshotCount": 2,
	"gpuDriverAvailable": true,
	"ipv4Addresses": [
		{"id": 1, "ip": "1.2.3.4", "gateway": "1.2.3.1", "netmask": "255.255.255.0", "broadcast": "1.2.3.255"}
	],
	"serverLiveInfo": {
		"state": "RUNNING",
		"disks": [],
		"cpuUsageInPercent": 12.5
	}
}`)

func TestJSONToTfValueServer(t *testing.T) {
	ctx := context.Background()
	obj := decodeServerBody(t, ctx)

	assertServerString(t, obj, "name", "myserver")
	assertServerString(t, obj, "hostname", "srv.example.com")
	assertServerNumber(t, obj, "max_cpu_count", "8")
	assertServerNumber(t, obj, "snapshot_count", "2")
	assertServerBool(t, obj, "gpu_driver_available", true)

	var liveInfo map[string]tftypes.Value
	if err := obj["server_live_info"].As(&liveInfo); err != nil {
		t.Fatalf("server_live_info as object: %v", err)
	}
	assertServerString(t, liveInfo, "state", "RUNNING")

	var ipv4 []tftypes.Value
	if err := obj["ipv4addresses"].As(&ipv4); err != nil || len(ipv4) != 1 {
		t.Fatalf("expected one ipv4address, got %d (%v)", len(ipv4), err)
	}
}

func decodeServerBody(t *testing.T, ctx context.Context) map[string]tftypes.Value {
	t.Helper()
	schema := scpserver.ScpServerDataSourceSchema(ctx)
	tfType := schema.Type().TerraformType(ctx)

	jsonVal, err := scpcommon.DecodeJSONResponse(scpServerJSONBody)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	tfVal, err := scpcommon.JSONToTfValue(ctx, tfType, jsonVal)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}

	var obj map[string]tftypes.Value
	if err := tfVal.As(&obj); err != nil {
		t.Fatalf("as object: %v", err)
	}
	return obj
}

func assertServerString(t *testing.T, obj map[string]tftypes.Value, attr, want string) {
	t.Helper()
	var got string
	if err := obj[attr].As(&got); err != nil || got != want {
		t.Fatalf("expected %s=%q, got %q (%v)", attr, want, got, err)
	}
}

func assertServerNumber(t *testing.T, obj map[string]tftypes.Value, attr, want string) {
	t.Helper()
	var got big.Float
	if err := obj[attr].As(&got); err != nil || got.String() != want {
		t.Fatalf("expected %s=%s, got %s (%v)", attr, want, got.String(), err)
	}
}

func assertServerBool(t *testing.T, obj map[string]tftypes.Value, attr string, want bool) {
	t.Helper()
	var got bool
	if err := obj[attr].As(&got); err != nil || got != want {
		t.Fatalf("expected %s=%v, got %v (%v)", attr, want, got, err)
	}
}
