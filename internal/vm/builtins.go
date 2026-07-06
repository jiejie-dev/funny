// v2/internal/vm/builtins.go
package vm

import (
	"fmt"

	"github.com/jiejie-dev/funny/v2/internal/bytecode"
	"github.com/jiejie-dev/funny/v2/internal/stdlib"
)

// execCallBuiltin handles CALL_BUILTIN nameIdx.
func (v *VM) execCallBuiltin(nameIdx int) error {
	info, ok := v.mod.Constants[nameIdx].(bytecode.BuiltinInfo)
	if !ok {
		return fmt.Errorf("vm: CALL_BUILTIN name is not a BuiltinInfo")
	}
	arity := info.Arity
	if len(v.stack) < arity {
		return fmt.Errorf("vm: %s() requires %d arguments", info.Name, arity)
	}
	start := len(v.stack) - arity
	args := make([]any, arity)
	for i := 0; i < arity; i++ {
		args[i] = v.stack[start+i]
	}
	v.stack = v.stack[:start]

	ret, err := stdlib.Call(info.Name, args)
	if err != nil {
		return fmt.Errorf("vm: %v", err)
	}
	if stdlib.SideEffectOnly(info.Name) {
		return nil
	}
	v.stack = append(v.stack, ret)
	return nil
}
