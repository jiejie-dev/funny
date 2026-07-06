package stdlib

import "encoding/json"

// ConvertJSON converts a generic Go value (from json.Unmarshal) into funny
// runtime values using []any and map[string]any.
func ConvertJSON(x any) any {
	switch v := x.(type) {
	case nil:
		return nil
	case bool:
		return v
	case float64:
		return v
	case string:
		return v
	case []any:
		out := make([]any, len(v))
		for i, e := range v {
			out[i] = ConvertJSON(e)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, e := range v {
			out[k] = ConvertJSON(e)
		}
		return out
	default:
		return v
	}
}

func toGoForJSON(val any) any {
	switch v := val.(type) {
	case nil:
		return nil
	case bool, int, float64, string:
		return v
	case []any:
		out := make([]any, len(v))
		for i, e := range v {
			out[i] = toGoForJSON(e)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, e := range v {
			out[k] = toGoForJSON(e)
		}
		return out
	}
	return val
}

func marshalJSON(val any) (string, error) {
	canonical, err := json.Marshal(toGoForJSON(val))
	if err != nil {
		return "", err
	}
	return string(canonical), nil
}

func parseJSON(s string) (any, error) {
	var x any
	if err := json.Unmarshal([]byte(s), &x); err != nil {
		return nil, err
	}
	return ConvertJSON(x), nil
}
