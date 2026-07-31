package scpcommon

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	lowerToUpperReg        = regexp.MustCompile(`([a-z])[A-Z]`)
	unsupportedCharsReg    = regexp.MustCompile(`[^a-zA-Z0-9_]+`)
	leadingNumbersReg      = regexp.MustCompile(`^(\d+)`)
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
func DecodeJSONResponse(body []byte) (interface{}, error) {
	var v interface{}
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

// JSONToTfValue converts a JSON-decoded value into a tftypes.Value that matches t.
func JSONToTfValue(ctx context.Context, t tftypes.Type, v interface{}) (tftypes.Value, error) {
	if v == nil {
		return tftypes.NewValue(t, nil), nil
	}

	if t.Is(tftypes.String) {
		s, err := toString(v)
		if err != nil {
			return tftypes.Value{}, err
		}
		return tftypes.NewValue(tftypes.String, s), nil
	}
	if t.Is(tftypes.Number) {
		n, err := toBigFloat(v)
		if err != nil {
			return tftypes.Value{}, err
		}
		return tftypes.NewValue(tftypes.Number, n), nil
	}
	if t.Is(tftypes.Bool) {
		b, ok := v.(bool)
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected bool, got %T", v)
		}
		return tftypes.NewValue(tftypes.Bool, b), nil
	}
	if t.Is(tftypes.DynamicPseudoType) {
		return dynamicJSONToTfValue(ctx, v)
	}

	switch ty := t.(type) {
	case tftypes.Object:
		m, ok := v.(map[string]interface{})
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected object for type %s, got %T", t, v)
		}
		// The generated Terraform schemas use snake_case attribute names, while
		// the SCP API returns camelCase keys. Build a lookup that preserves the
		// original values (including map keys) but normalizes only this object's
		// attribute names.
		normalized := make(map[string]interface{}, len(m))
		for k, val := range m {
			normalized[TerraformIdentifier(k)] = val
		}
		vals := make(map[string]tftypes.Value, len(ty.AttributeTypes))
		for attr, attrType := range ty.AttributeTypes {
			attrVal, ok := normalized[attr]
			if !ok {
				vals[attr] = tftypes.NewValue(attrType, nil)
				continue
			}
			converted, err := JSONToTfValue(ctx, attrType, attrVal)
			if err != nil {
				return tftypes.Value{}, err
			}
			vals[attr] = converted
		}
		return tftypes.NewValue(t, vals), nil
	case tftypes.List:
		s, ok := v.([]interface{})
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected list for type %s, got %T", t, v)
		}
		vals := make([]tftypes.Value, 0, len(s))
		for _, elem := range s {
			converted, err := JSONToTfValue(ctx, ty.ElementType, elem)
			if err != nil {
				return tftypes.Value{}, err
			}
			vals = append(vals, converted)
		}
		return tftypes.NewValue(t, vals), nil
	case tftypes.Set:
		s, ok := v.([]interface{})
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected set for type %s, got %T", t, v)
		}
		vals := make([]tftypes.Value, 0, len(s))
		for _, elem := range s {
			converted, err := JSONToTfValue(ctx, ty.ElementType, elem)
			if err != nil {
				return tftypes.Value{}, err
			}
			vals = append(vals, converted)
		}
		return tftypes.NewValue(t, vals), nil
	case tftypes.Map:
		m, ok := v.(map[string]interface{})
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected map for type %s, got %T", t, v)
		}
		vals := make(map[string]tftypes.Value, len(m))
		for k, elem := range m {
			converted, err := JSONToTfValue(ctx, ty.ElementType, elem)
			if err != nil {
				return tftypes.Value{}, err
			}
			vals[k] = converted
		}
		return tftypes.NewValue(t, vals), nil
	case tftypes.Tuple:
		s, ok := v.([]interface{})
		if !ok {
			return tftypes.Value{}, fmt.Errorf("expected tuple for type %s, got %T", t, v)
		}
		if len(s) != len(ty.ElementTypes) {
			return tftypes.Value{}, fmt.Errorf("expected tuple of length %d, got %d", len(ty.ElementTypes), len(s))
		}
		vals := make([]tftypes.Value, 0, len(s))
		for i, elem := range s {
			converted, err := JSONToTfValue(ctx, ty.ElementTypes[i], elem)
			if err != nil {
				return tftypes.Value{}, err
			}
			vals = append(vals, converted)
		}
		return tftypes.NewValue(t, vals), nil
	default:
		return tftypes.Value{}, fmt.Errorf("unsupported tftypes.Type %T", t)
	}
}

func toString(v interface{}) (string, error) {
	switch val := v.(type) {
	case string:
		return val, nil
	case json.Number:
		return val.String(), nil
	case nil:
		return "", nil
	default:
		return fmt.Sprintf("%v", val), nil
	}
}

func toBigFloat(v interface{}) (*big.Float, error) {
	switch val := v.(type) {
	case json.Number:
		f, _, err := big.NewFloat(0).SetPrec(128).Parse(val.String(), 10)
		if err != nil {
			return nil, err
		}
		return f, nil
	case float64:
		return big.NewFloat(val).SetPrec(128), nil
	case int:
		return big.NewFloat(float64(val)).SetPrec(128), nil
	case int64:
		return big.NewFloat(float64(val)).SetPrec(128), nil
	case string:
		f, _, err := big.NewFloat(0).SetPrec(128).Parse(val, 10)
		if err != nil {
			return nil, err
		}
		return f, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to number", v)
	}
}

func dynamicJSONToTfValue(ctx context.Context, v interface{}) (tftypes.Value, error) {
	switch val := v.(type) {
	case nil:
		return tftypes.NewValue(tftypes.DynamicPseudoType, nil), nil
	case bool:
		return tftypes.NewValue(tftypes.DynamicPseudoType, val), nil
	case json.Number:
		n, err := toBigFloat(val)
		if err != nil {
			return tftypes.Value{}, err
		}
		return tftypes.NewValue(tftypes.DynamicPseudoType, n), nil
	case float64:
		return tftypes.NewValue(tftypes.DynamicPseudoType, big.NewFloat(val).SetPrec(128)), nil
	case int:
		return tftypes.NewValue(tftypes.DynamicPseudoType, big.NewFloat(float64(val)).SetPrec(128)), nil
	case string:
		return tftypes.NewValue(tftypes.DynamicPseudoType, val), nil
	case []interface{}:
		elems := make([]tftypes.Value, 0, len(val))
		for _, e := range val {
			el, err := dynamicJSONToTfValue(ctx, e)
			if err != nil {
				return tftypes.Value{}, err
			}
			elems = append(elems, el)
		}
		return tftypes.NewValue(tftypes.DynamicPseudoType, elems), nil
	case map[string]interface{}:
		vals := make(map[string]tftypes.Value, len(val))
		for k, e := range val {
			el, err := dynamicJSONToTfValue(ctx, e)
			if err != nil {
				return tftypes.Value{}, err
			}
			vals[k] = el
		}
		return tftypes.NewValue(tftypes.DynamicPseudoType, vals), nil
	default:
		return tftypes.Value{}, fmt.Errorf("unsupported dynamic value type %T", val)
	}
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

// TfValueToJSON converts a tftypes.Value into a Go value that can be JSON
// marshaled. Object attribute names are converted from snake_case to camelCase
// using SnakeToCamel. Unknown and null values are omitted so they are not sent
// in request bodies. Numbers are returned as json.Number to preserve precision.
func TfValueToJSON(ctx context.Context, v tftypes.Value) (interface{}, error) {
	if v.IsNull() || !v.IsKnown() {
		return nil, nil
	}

	t := v.Type()
	if t.Is(tftypes.String) {
		var s string
		if err := v.As(&s); err != nil {
			return nil, err
		}
		return s, nil
	}
	if t.Is(tftypes.Bool) {
		var b bool
		if err := v.As(&b); err != nil {
			return nil, err
		}
		return b, nil
	}
	if t.Is(tftypes.Number) {
		var n big.Float
		if err := v.As(&n); err != nil {
			return nil, err
		}
		return json.Number(n.Text('f', -1)), nil
	}

	switch t.(type) {
	case tftypes.Object:
		var m map[string]tftypes.Value
		if err := v.As(&m); err != nil {
			return nil, err
		}
		out := make(map[string]interface{}, len(m))
		for k, elem := range m {
			converted, err := TfValueToJSON(ctx, elem)
			if err != nil {
				return nil, err
			}
			if converted == nil {
				continue
			}
			out[SnakeToCamel(k)] = converted
		}
		return out, nil
	case tftypes.List, tftypes.Set, tftypes.Tuple:
		var l []tftypes.Value
		if err := v.As(&l); err != nil {
			return nil, err
		}
		out := make([]interface{}, 0, len(l))
		for _, elem := range l {
			converted, err := TfValueToJSON(ctx, elem)
			if err != nil {
				return nil, err
			}
			if converted == nil {
				continue
			}
			out = append(out, converted)
		}
		return out, nil
	case tftypes.Map:
		var m map[string]tftypes.Value
		if err := v.As(&m); err != nil {
			return nil, err
		}
		out := make(map[string]interface{}, len(m))
		for k, elem := range m {
			converted, err := TfValueToJSON(ctx, elem)
			if err != nil {
				return nil, err
			}
			if converted == nil {
				continue
			}
			out[k] = converted
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported tftypes.Type %T for JSON conversion", t)
	}
}
