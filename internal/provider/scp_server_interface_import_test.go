//nolint:testpackage
package provider

import (
	"testing"
)

func TestParseServerMACImportID(t *testing.T) {
	validMAC := "00:11:22:33:44:55"

	tests := []struct {
		name      string
		id        string
		wantID    int64
		wantMAC   string
		wantError bool
	}{
		{name: "valid", id: "12345/" + validMAC, wantID: 12345, wantMAC: validMAC},
		{name: "missing slash", id: "12345", wantError: true},
		{name: "empty mac", id: "12345/", wantError: true},
		{name: "empty server id", id: "/" + validMAC, wantError: true},
		{name: "non-numeric server id", id: "abc/" + validMAC, wantError: true},
		{name: "extra slash", id: "12345/" + validMAC + "/extra", wantError: true},
		{name: "empty", id: "", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverID, mac, err := parseServerMACImportID(tt.id)
			if tt.wantError {
				if err == nil {
					t.Errorf("parseServerMACImportID(%q): expected error, got (%d, %q)", tt.id, serverID, mac)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseServerMACImportID(%q): unexpected error: %v", tt.id, err)
			}
			if serverID != tt.wantID || mac != tt.wantMAC {
				t.Errorf("parseServerMACImportID(%q) = (%d, %q), want (%d, %q)", tt.id, serverID, mac, tt.wantID, tt.wantMAC)
			}
		})
	}
}
