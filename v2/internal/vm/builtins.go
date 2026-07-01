// v2/internal/vm/builtins.go
package vm

import (
	"fmt"
	"reflect"

	"github.com/jerloo/funny/v2/internal/bytecode"
)

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
	default:
		return fmt.Errorf("vm: unknown builtin %q", name)
	}
	return nil
}
