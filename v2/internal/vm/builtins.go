// v2/internal/vm/builtins.go
package vm

import (
	"fmt"
	"reflect"

	"github.com/jerloo/funny/v2/internal/bytecode"
)

// execCallBuiltin handles CALL_BUILTIN nameIdx.
// The constant pool at nameIdx must be a string identifying the builtin.
// Pops arguments from stack (depending on the builtin), pushes result.
func (v *VM) execCallBuiltin(nameIdx int) error {
	name, ok := v.mod.Constants[nameIdx].(string)
	if !ok {
		return fmt.Errorf("vm: CALL_BUILTIN name is not a string")
	}
	switch name {
	case "print":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: print() requires at least 1 argument")
		}
		fmt.Print(v.stack[len(v.stack)-1])
		v.stack = v.stack[:len(v.stack)-1]
	case "println":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: println() requires at least 1 argument")
		}
		fmt.Println(v.stack[len(v.stack)-1])
		v.stack = v.stack[:len(v.stack)-1]
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
