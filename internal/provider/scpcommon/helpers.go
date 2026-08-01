package scpcommon

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	lowerToUpperReg     = regexp.MustCompile(`([a-z])[A-Z]`)
	unsupportedCharsReg = regexp.MustCompile(`[^a-zA-Z0-9_]+`)
	leadingNumbersReg   = regexp.MustCompile(`^(\d+)`)
)

// TerraformIdentifier converts a name to the snake_case form used by the Terraform code generator.
func TerraformIdentifier(original string) string {
	if len(original) == 0 {
		return original
	}
	removed := unsupportedCharsReg.ReplaceAllString(original, "")
	noLeading := leadingNumbersReg.ReplaceAllString(removed, "")
	inserted := lowerToUpperReg.ReplaceAllStringFunc(noLeading, func(s string) string {
		firstRune, size := utf8.DecodeRuneInString(s)
		return fmt.Sprintf("%s_%s", string(firstRune), strings.ToLower(s[size:]))
	})
	return strings.ToLower(inserted)
}

// DecodeJSONResponse decodes JSON using json.Number to preserve numeric precision.
func DecodeJSONResponse(body []byte) (any, error) {
	var v any
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

// knownAcronyms are short lowercase tokens that should remain lowercase when
// converting snake_case Terraform identifiers back to camelCase JSON keys.
// They are used by SnakeToCamel to split tokens like "ipv4addresses" into
// "ipv4" and "addresses" before title-casing the trailing part.
var knownAcronyms = []string{"ipv4", "ipv6", "rdns"}

// SnakeToCamel converts a snake_case Terraform attribute name to a lowerCamelCase
// JSON key. It understands the same acronym tokens that TerraformIdentifier
// does not split on its own, so "ipv4addresses" becomes "ipv4Addresses" and
// "source_ports" becomes "sourcePorts".
func SnakeToCamel(s string) string {
	if s == "" {
		return s
	}
	for _, ac := range knownAcronyms {
		if strings.HasPrefix(strings.ToLower(s), ac) && len(s) > len(ac) {
			next := s[len(ac)]
			if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') {
				s = ac + "_" + s[len(ac):]
			}
		}
	}
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if i == 0 {
			parts[i] = strings.ToLower(p)
			continue
		}
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
	}
	return strings.Join(parts, "")
}

// JSONToTfValue converts a JSON-decoded value into a tftypes.Value that matches t.
func JSONToTfValue(ctx context.Context, t tftypes.Type, v any) (tftypes.Value, error) {
	return jsonToTfValue(ctx, t, v)
}

// TfValueToJSON converts a tftypes.Value into a Go value that can be JSON
// marshaled. Object attribute names are converted from snake_case to camelCase
// using SnakeToCamel. Unknown and null values are omitted so they are not sent
// in request bodies. Numbers are returned as json.Number to preserve precision.
func TfValueToJSON(ctx context.Context, v tftypes.Value) (any, error) {
	return tfValueToJSON(ctx, v)
}
