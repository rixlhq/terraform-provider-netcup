package scpcommon

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func tfValueToJSON(ctx context.Context, v tftypes.Value) (any, error) {
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
		return tfObjectToJSON(ctx, v)
	case tftypes.List, tftypes.Set, tftypes.Tuple:
		return tfCollectionToJSON(ctx, v)
	case tftypes.Map:
		return tfMapToJSON(ctx, v)
	default:
		return nil, fmt.Errorf("unsupported tftypes.Type %T for JSON conversion", t)
	}
}

func tfObjectToJSON(ctx context.Context, v tftypes.Value) (any, error) {
	var m map[string]tftypes.Value
	if err := v.As(&m); err != nil {
		return nil, err
	}
	out := make(map[string]any, len(m))
	for k, elem := range m {
		converted, err := tfValueToJSON(ctx, elem)
		if err != nil {
			return nil, err
		}
		if converted == nil {
			continue
		}
		out[SnakeToCamel(k)] = converted
	}
	return out, nil
}

func tfCollectionToJSON(ctx context.Context, v tftypes.Value) (any, error) {
	var l []tftypes.Value
	if err := v.As(&l); err != nil {
		return nil, err
	}
	out := make([]any, 0, len(l))
	for _, elem := range l {
		converted, err := tfValueToJSON(ctx, elem)
		if err != nil {
			return nil, err
		}
		if converted == nil {
			continue
		}
		out = append(out, converted)
	}
	return out, nil
}

func tfMapToJSON(ctx context.Context, v tftypes.Value) (any, error) {
	var m map[string]tftypes.Value
	if err := v.As(&m); err != nil {
		return nil, err
	}
	out := make(map[string]any, len(m))
	for k, elem := range m {
		converted, err := tfValueToJSON(ctx, elem)
		if err != nil {
			return nil, err
		}
		if converted == nil {
			continue
		}
		out[k] = converted
	}
	return out, nil
}
