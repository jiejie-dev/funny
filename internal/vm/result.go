// v2/internal/vm/result.go
package vm

import "github.com/jiejie-dev/funny/v2/internal/bytecode"

// makeResult constructs a Result runtime value: map{tag, val}.
func makeResult(tag string, val bytecode.Value) bytecode.Value {
	return map[string]any{
		"tag": tag,
		"val": val,
	}
}

func resultFields(v bytecode.Value) (tag string, val bytecode.Value, ok bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return "", nil, false
	}
	t, _ := m["tag"].(string)
	inner, has := m["val"]
	return t, inner, has
}

// isResult reports whether v is a Result runtime value.
func isResult(v bytecode.Value) bool {
	tag, _, ok := resultFields(v)
	return ok && (tag == "ok" || tag == "err")
}

// resultTag returns "ok" or "err" (or "" if not a Result).
func resultTag(v bytecode.Value) string {
	tag, _, ok := resultFields(v)
	if !ok {
		return ""
	}
	return tag
}

// resultVal returns the inner value of a Result.
func resultVal(v bytecode.Value) bytecode.Value {
	_, val, ok := resultFields(v)
	if !ok {
		return nil
	}
	return val
}
