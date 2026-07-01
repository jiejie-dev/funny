// v2/internal/vm/builtins.go
package vm

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/jerloo/funny/v2/internal/bytecode"
)

// makeResult builds an Ok/Err tagged-result map for builtins that return
// a value-or-error outcome instead of raising a VM error.
func makeResult(tag string, value bytecode.Value) map[string]bytecode.Value {
	return map[string]bytecode.Value{"tag": tag, "value": value}
}

// convertJSON converts a generic Go value (from json.Unmarshal) into a funny
// Value, using []any and map[string]any instead of []interface{} and
// map[string]interface{}.
func convertJSON(x any) bytecode.Value {
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
		out := make([]bytecode.Value, len(v))
		for i, e := range v {
			out[i] = convertJSON(e)
		}
		return out
	case map[string]any:
		out := make(map[string]bytecode.Value, len(v))
		for k, e := range v {
			out[k] = convertJSON(e)
		}
		return out
	default:
		return fmt.Sprintf("%v", v)
	}
}

// execCallBuiltin handles CALL_BUILTIN nameIdx.
// The constant pool at nameIdx must be a struct{ Name string; Arity int }
// identifying the builtin and its argument count. Pops Arity args from stack
// (in source order, bottom-of-stack first) and consumes them.
func (v *VM) execCallBuiltin(nameIdx int) error {
	info, ok := v.mod.Constants[nameIdx].(bytecode.BuiltinInfo)
	if !ok {
		return fmt.Errorf("vm: CALL_BUILTIN name is not a BuiltinInfo")
	}
	name := info.Name
	arity := info.Arity
	if len(v.stack) < arity {
		return fmt.Errorf("vm: %s() requires %d arguments", name, arity)
	}
	start := len(v.stack) - arity
	switch name {
	case "print":
		for i := start; i < len(v.stack); i++ {
			if i > start {
				fmt.Print(" ")
			}
			fmt.Print(v.stack[i])
		}
		v.stack = v.stack[:start]
	case "println":
		for i := start; i < len(v.stack); i++ {
			if i > start {
				fmt.Print(" ")
			}
			fmt.Print(v.stack[i])
		}
		fmt.Println()
		v.stack = v.stack[:start]
	case "len":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: len() requires 1 argument")
		}
		x := v.stack[len(v.stack)-1]
		v.stack = v.stack[:len(v.stack)-1]
		switch val := x.(type) {
		case string:
			v.stack = append(v.stack, len(val))
		case []bytecode.Value:
			v.stack = append(v.stack, len(val))
		default:
			v.stack = append(v.stack, reflect.ValueOf(val).Len())
		}
	case "to_str":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: to_str() requires 1 argument")
		}
		v.stack[len(v.stack)-1] = fmt.Sprintf("%v", v.stack[len(v.stack)-1])
	case "to_int":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: to_int() requires 1 argument")
		}
		switch x := v.stack[len(v.stack)-1].(type) {
		case int:
			_ = x
		case float64:
			v.stack[len(v.stack)-1] = int(x)
		case string:
			var n int
			for _, c := range x {
				if c >= '0' && c <= '9' {
					n = n*10 + int(c-'0')
				}
			}
			v.stack[len(v.stack)-1] = n
		}
	case "type_of":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: type_of() requires 1 argument")
		}
		switch v.stack[len(v.stack)-1].(type) {
		case nil:
			v.stack[len(v.stack)-1] = "nil"
		case bool:
			v.stack[len(v.stack)-1] = "bool"
		case int:
			v.stack[len(v.stack)-1] = "int"
		case float64:
			v.stack[len(v.stack)-1] = "float"
		case string:
			v.stack[len(v.stack)-1] = "str"
		case []bytecode.Value:
			v.stack[len(v.stack)-1] = "list"
		case map[string]bytecode.Value:
			v.stack[len(v.stack)-1] = "map"
		default:
			v.stack[len(v.stack)-1] = "unknown"
		}
	case "to_json":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: to_json() requires 1 argument")
		}
		s, ok := v.stack[len(v.stack)-1].(string)
		if !ok {
			return fmt.Errorf("vm: to_json() requires a string argument")
		}
		v.stack = v.stack[:len(v.stack)-1]
		var x any
		if err := json.Unmarshal([]byte(s), &x); err != nil {
			return fmt.Errorf("vm: to_json: invalid JSON: %v", err)
		}
		v.stack = append(v.stack, convertJSON(x))
	case "parse_json":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: parse_json() requires 1 argument")
		}
		s, ok := v.stack[len(v.stack)-1].(string)
		if !ok {
			return fmt.Errorf("vm: parse_json() requires a string argument")
		}
		v.stack = v.stack[:len(v.stack)-1]
		var x any
		if err := json.Unmarshal([]byte(s), &x); err != nil {
			v.stack = append(v.stack, makeResult("err", fmt.Sprintf("parse_json: %v", err)))
			return nil
		}
		v.stack = append(v.stack, convertJSON(x))
	case "now":
		v.stack = append(v.stack, int(time.Now().Unix()))
	case "time_format":
		if len(v.stack) < 2 {
			return fmt.Errorf("vm: time_format() requires 2 arguments")
		}
		layout := v.stack[len(v.stack)-1].(string)
		ts := v.stack[len(v.stack)-2].(int)
		v.stack = v.stack[:len(v.stack)-2]
		t := time.Unix(int64(ts), 0)
		v.stack = append(v.stack, t.Format(layout))
	default:
		return fmt.Errorf("vm: unknown builtin %q", name)
	}
	return nil
}
