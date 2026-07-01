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