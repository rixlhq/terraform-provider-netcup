package scpcommon_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/rixlhq/terraform-provider-netcup/internal/provider/scpcommon"
)

// TestJSONToTfValue_MapFromEmptyArray pins the SCP API's empty-collection
// spelling: an EMPTY JSON array is accepted for a map-typed attribute (the live
// case is scp_server_interfaces[*].ipv6addresses[*].rdns, which the API returns
// as [] for a server with no reverse-DNS entries), while a NON-empty array stays
// a loud error so entries are never silently dropped.
func TestJSONToTfValue_MapFromEmptyArray(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	mapType := tftypes.Map{ElementType: tftypes.String}

	empty, err := scpcommon.JSONToTfValue(ctx, mapType, []any{})
	if err != nil {
		t.Fatalf("empty array must convert to an empty map, got error: %v", err)
	}
	var emptyMap map[string]tftypes.Value
	if err := empty.As(&emptyMap); err != nil {
		t.Fatalf("empty map is not readable: %v", err)
	}
	if len(emptyMap) != 0 {
		t.Fatalf("expected an empty map, got %d entries", len(emptyMap))
	}

	if _, err := scpcommon.JSONToTfValue(ctx, mapType, []any{"not-a-map-entry"}); err == nil {
		t.Fatal("a NON-empty array must stay an error (no silent data loss)")
	}

	// A normal object still converts, and null stays null.
	obj, err := scpcommon.JSONToTfValue(ctx, mapType, map[string]any{"k": "v"})
	if err != nil {
		t.Fatalf("object conversion regressed: %v", err)
	}
	var objMap map[string]tftypes.Value
	if err := obj.As(&objMap); err != nil {
		t.Fatalf("object is not readable: %v", err)
	}
	var got string
	if err := objMap["k"].As(&got); err != nil || got != "v" {
		t.Fatalf("expected k=v, got %q (err %v)", got, err)
	}

	null, err := scpcommon.JSONToTfValue(ctx, mapType, nil)
	if err != nil {
		t.Fatalf("null must convert to a null value: %v", err)
	}
	if !null.IsNull() {
		t.Fatal("expected a null value")
	}
}
