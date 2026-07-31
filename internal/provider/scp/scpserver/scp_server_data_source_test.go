package scpserver

import (
	"context"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
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

	jsonVal, err := scpcommon.DecodeJSONResponse(body)
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

	var name string
	if err := obj["name"].As(&name); err != nil || name != "myserver" {
		t.Fatalf("expected name=myserver, got %q (%v)", name, err)
	}

	var hostname string
	if err := obj["hostname"].As(&hostname); err != nil || hostname != "srv.example.com" {
		t.Fatalf("expected hostname=srv.example.com, got %q (%v)", hostname, err)
	}

	var maxCPU big.Float
	if err := obj["max_cpu_count"].As(&maxCPU); err != nil || maxCPU.String() != "8" {
		t.Fatalf("expected max_cpu_count=8, got %s (%v)", maxCPU.String(), err)
	}

	var snapshotCount big.Float
	if err := obj["snapshot_count"].As(&snapshotCount); err != nil || snapshotCount.String() != "2" {
		t.Fatalf("expected snapshot_count=2, got %s (%v)", snapshotCount.String(), err)
	}

	var gpuAvailable bool
	if err := obj["gpu_driver_available"].As(&gpuAvailable); err != nil || !gpuAvailable {
		t.Fatalf("expected gpu_driver_available=true, got %v (%v)", gpuAvailable, err)
	}

	var liveInfo map[string]tftypes.Value
	if err := obj["server_live_info"].As(&liveInfo); err != nil {
		t.Fatalf("server_live_info as object: %v", err)
	}

	var state string
	if err := liveInfo["state"].As(&state); err != nil || state != "RUNNING" {
		t.Fatalf("expected server_live_info.state=RUNNING, got %q (%v)", state, err)
	}

	var ipv4 []tftypes.Value
	if err := obj["ipv4addresses"].As(&ipv4); err != nil || len(ipv4) != 1 {
		t.Fatalf("expected one ipv4address, got %d (%v)", len(ipv4), err)
	}
}
