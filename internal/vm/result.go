// v2/internal/vm/result.go
package vm

import "github.com/jiejie-dev/funny/v2/internal/bytecode"

// makeResult constructs a Result runtime value: map{tag, val}.
func makeResult(tag string, val bytecode.Value) bytecode.Value {
	return map[string]bytecode.Value{
		"tag": tag,
		"val": val,
	}
}

// isResult reports whether v is a Result runtime value.
func isResult(v bytecode.Value) bool {
	m, ok := v.(map[string]bytecode.Value)
	if !ok {
		return false
	}
	_, hasTag := m["tag"]
	_, hasVal := m["val"]
	return hasTag && hasVal
}

// resultTag returns "ok" or "err" (or "" if not a Result).
func resultTag(v bytecode.Value) string {
	m, ok := v.(map[string]bytecode.Value)
	if !ok {
		return ""
	}
	tag, _ := m["tag"].(string)
	return tag
}

// resultVal returns the inner value of a Result.
func resultVal(v bytecode.Value) bytecode.Value {
	m, _ := v.(map[string]bytecode.Value)
	return m["val"]
}