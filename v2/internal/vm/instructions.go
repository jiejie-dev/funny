// v2/internal/vm/instructions.go
package vm

import (
	"fmt"

	"github.com/jerloo/funny/v2/internal/bytecode"
)

// execArith handles arithmetic operations on the top two stack values.
// Pops b first, then a, pushes result.
func (v *VM) execArith(op bytecode.OpCode, a, b bytecode.Value) (bytecode.Value, error) {
	switch op {
	case bytecode.ADD_INT:
		return a.(int) + b.(int), nil
	case bytecode.SUB_INT:
		return a.(int) - b.(int), nil
	case bytecode.MUL_INT:
		return a.(int) * b.(int), nil
	case bytecode.DIV_INT:
		av, ok := a.(int)
		if !ok {
			return nil, fmt.Errorf("vm: DIV_INT left operand not int")
		}
		bv, ok := b.(int)
		if !ok {
			return nil, fmt.Errorf("vm: DIV_INT right operand not int")
		}
		if bv == 0 {
			return nil, fmt.Errorf("vm: division by zero")
		}
		return av / bv, nil
	case bytecode.MOD_INT:
		return a.(int) % b.(int), nil
	case bytecode.ADD_FLOAT:
		return a.(float64) + b.(float64), nil
	case bytecode.SUB_FLOAT:
		return a.(float64) - b.(float64), nil
	case bytecode.MUL_FLOAT:
		return a.(float64) * b.(float64), nil
	case bytecode.DIV_FLOAT:
		return a.(float64) / b.(float64), nil
	case bytecode.ADD_STR:
		return a.(string) + b.(string), nil
	}
	return nil, fmt.Errorf("vm: unsupported arith op %s", op)
}

// execCmp handles comparison operations on the top two stack values.
// Pops b first, then a, pushes bool result.
func (v *VM) execCmp(op bytecode.OpCode, a, b bytecode.Value) (bool, error) {
	switch op {
	case bytecode.EQ_INT:
		return a.(int) == b.(int), nil
	case bytecode.EQ_STR:
		return a.(string) == b.(string), nil
	case bytecode.EQ_BOOL:
		return a.(bool) == b.(bool), nil
	case bytecode.EQ_NIL:
		return a == nil && b == nil, nil
	case bytecode.LT_INT:
		return a.(int) < b.(int), nil
	case bytecode.GT_INT:
		return a.(int) > b.(int), nil
	case bytecode.LTE_INT:
		return a.(int) <= b.(int), nil
	case bytecode.GTE_INT:
		return a.(int) >= b.(int), nil
	}
	return false, fmt.Errorf("vm: unsupported cmp op %s", op)
}

// execUnary handles unary operations on the top stack value.
func (v *VM) execUnary(op bytecode.OpCode, a bytecode.Value) (bytecode.Value, error) {
	switch op {
	case bytecode.NEG_INT:
		return -a.(int), nil
	case bytecode.NEG_FLOAT:
		return -a.(float64), nil
	case bytecode.NOT_BOOL:
		return !a.(bool), nil
	}
	return nil, fmt.Errorf("vm: unsupported unary op %s", op)
}

// pop2 pops two values from the stack, returning (a, b) where b was on top.
func (v *VM) pop2() (bytecode.Value, bytecode.Value) {
	n := len(v.stack)
	b := v.stack[n-1]
	a := v.stack[n-2]
	v.stack = v.stack[:n-2]
	return a, b
}

// pop pops the top of the stack.
func (v *VM) pop() bytecode.Value {
	n := len(v.stack)
	x := v.stack[n-1]
	v.stack = v.stack[:n-1]
	return x
}

// execCall handles CALL fnIdx.
// Pops argCount args from the stack (in reverse), pushes new frame with args as locals.
func (v *VM) execCall(fnIdx int) error {
	if fnIdx < 0 || fnIdx >= len(v.mod.Functions) {
		return fmt.Errorf("vm: CALL invalid function index %d", fnIdx)
	}
	callee := v.mod.Functions[fnIdx]
	n := callee.Arity
	if len(v.stack) < n {
		return fmt.Errorf("vm: CALL %s expects %d args, got %d", callee.Name, n, len(v.stack))
	}
	args := make([]bytecode.Value, n)
	for i := n - 1; i >= 0; i-- {
		args[i] = v.stack[len(v.stack)-1-(n-1-i)]
	}
	v.stack = v.stack[:len(v.stack)-n]
	newFrame := &Frame{
		fn:     callee,
		ip:     0,
		locals: make([]bytecode.Value, callee.NumLocals),
	}
	for i, a := range args {
		newFrame.locals[i] = a
	}
	v.frames = append(v.frames, newFrame)
	return nil
}

// execReturn handles RETURN.
// Pops the current frame, pushes top-of-stack as caller's return value (if any).
func (v *VM) execReturn() error {
	if len(v.frames) == 0 {
		return fmt.Errorf("vm: RETURN with no frames")
	}
	var retVal bytecode.Value
	if len(v.stack) > 0 {
		retVal = v.stack[len(v.stack)-1]
		v.stack = v.stack[:len(v.stack)-1]
	}
	v.frames = v.frames[:len(v.frames)-1]
	if retVal != nil {
		v.stack = append(v.stack, retVal)
	}
	return nil
}