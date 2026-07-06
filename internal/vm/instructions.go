// v2/internal/vm/instructions.go
package vm

import (
	"fmt"

	"github.com/jiejie-dev/funny/v2/internal/bytecode"
	"github.com/jiejie-dev/funny/v2/internal/strfmt"
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

// execCmp handles comparison and logical operations on the top two stack
// values. Pops b first, then a, pushes bool result.
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
	case bytecode.EQ_FLOAT:
		return a.(float64) == b.(float64), nil
	case bytecode.LT_INT:
		return a.(int) < b.(int), nil
	case bytecode.GT_INT:
		return a.(int) > b.(int), nil
	case bytecode.LTE_INT:
		return a.(int) <= b.(int), nil
	case bytecode.GTE_INT:
		return a.(int) >= b.(int), nil
	case bytecode.LT_FLOAT:
		return a.(float64) < b.(float64), nil
	case bytecode.GT_FLOAT:
		return a.(float64) > b.(float64), nil
	case bytecode.LTE_FLOAT:
		return a.(float64) <= b.(float64), nil
	case bytecode.GTE_FLOAT:
		return a.(float64) >= b.(float64), nil
	case bytecode.AND_BOOL:
		return a.(bool) && b.(bool), nil
	case bytecode.OR_BOOL:
		return a.(bool) || b.(bool), nil
	}
	return false, fmt.Errorf("vm: unsupported cmp op %s", op)
}

// execInList implements `elem in list`. Pops list (top) then elem.
func (v *VM) execInList(elem, list bytecode.Value) bool {
	items, ok := list.([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		if item == elem {
			return true
		}
	}
	return false
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

// execBuildList handles BUILD_LIST n. Pops n values from stack (in reverse), pushes a []Value.
func (v *VM) execBuildList(n int) {
	items := make([]bytecode.Value, n)
	for i := n - 1; i >= 0; i-- {
		items[i] = v.stack[len(v.stack)-1]
		v.stack = v.stack[:len(v.stack)-1]
	}
	v.stack = append(v.stack, items)
}

// execIndex handles INDEX. Pops index then object, pushes element.
func (v *VM) execIndex() error {
	if len(v.stack) < 2 {
		return fmt.Errorf("vm: INDEX requires 2 stack values")
	}
	idx := v.stack[len(v.stack)-1]
	obj := v.stack[len(v.stack)-2]
	v.stack = v.stack[:len(v.stack)-2]
	if m, ok := obj.(map[string]bytecode.Value); ok {
		ks, ok := idx.(string)
		if !ok {
			ks = fmt.Sprintf("%v", idx)
		}
		val, ok := m[ks]
		if !ok {
			return fmt.Errorf("vm: INDEX map has no key %q", ks)
		}
		v.stack = append(v.stack, val)
		return nil
	}
	i, ok := idx.(int)
	if !ok {
		return fmt.Errorf("vm: INDEX index not int")
	}
	switch val := obj.(type) {
	case []bytecode.Value:
		if i < 0 || i >= len(val) {
			return fmt.Errorf("vm: INDEX list out of range")
		}
		v.stack = append(v.stack, val[i])
	case string:
		runes := []rune(val)
		if i < 0 || i >= len(runes) {
			return fmt.Errorf("vm: INDEX string out of range")
		}
		v.stack = append(v.stack, string(runes[i]))
	default:
		return fmt.Errorf("vm: INDEX on non-list/string")
	}
	return nil
}

// execSetIndex handles SET_INDEX for `obj[idx] = value`. Stack layout on
// entry (bottom to top): value, object, index. Pops index and object, and
// leaves value on top of the stack (mirroring STORE_LOCAL's peek-and-store
// semantics), so the compiler can emit a trailing POP for the statement form.
func (v *VM) execSetIndex() error {
	if len(v.stack) < 3 {
		return fmt.Errorf("vm: SET_INDEX requires 3 stack values")
	}
	idx := v.stack[len(v.stack)-1]
	obj := v.stack[len(v.stack)-2]
	val := v.stack[len(v.stack)-3]
	v.stack = v.stack[:len(v.stack)-2]
	switch o := obj.(type) {
	case []bytecode.Value:
		i, ok := idx.(int)
		if !ok {
			return fmt.Errorf("vm: SET_INDEX list index not int")
		}
		if i < 0 || i >= len(o) {
			return fmt.Errorf("vm: SET_INDEX list out of range")
		}
		o[i] = val
		return nil
	case map[string]bytecode.Value:
		ks, ok := idx.(string)
		if !ok {
			ks = fmt.Sprintf("%v", idx)
		}
		o[ks] = val
		return nil
	}
	return fmt.Errorf("vm: SET_INDEX on non-list/map")
}

// execBuildMap handles BUILD_MAP n. Pops 2n values (alternating key, value), pushes map.
func (v *VM) execBuildMap(n int) {
	m := make(map[string]bytecode.Value, n)
	for i := 0; i < n; i++ {
		val := v.stack[len(v.stack)-1]
		v.stack = v.stack[:len(v.stack)-1]
		key := v.stack[len(v.stack)-1]
		v.stack = v.stack[:len(v.stack)-1]
		ks, ok := key.(string)
		if !ok {
			ks = fmt.Sprintf("%v", key)
		}
		m[ks] = val
	}
	v.stack = append(v.stack, m)
}

// execGetField handles GET_FIELD. Pops field name then object, pushes value.
func (v *VM) execGetField() error {
	if len(v.stack) < 2 {
		return fmt.Errorf("vm: GET_FIELD requires 2 stack values")
	}
	fname := v.stack[len(v.stack)-1]
	obj := v.stack[len(v.stack)-2]
	v.stack = v.stack[:len(v.stack)-2]
	fs, ok := fname.(string)
	if !ok {
		return fmt.Errorf("vm: GET_FIELD field name not string")
	}
	switch o := obj.(type) {
	case map[string]bytecode.Value:
		if val, ok := o[fs]; ok {
			v.stack = append(v.stack, val)
		} else {
			v.stack = append(v.stack, nil)
		}
	default:
		return fmt.Errorf("vm: GET_FIELD on non-map/struct")
	}
	return nil
}

// execNewStruct handles NEW_STRUCT. For M2-B.5, this is a no-op tag.
// The map is already on the stack; we just leave it as-is.
func (v *VM) execNewStruct() {
	// no-op for M2-B.5; structs are just maps.
}

// execFormatValue handles FORMAT_VALUE specIdx. Pops a value, formats it
// using the format spec string at the given constant-pool index (used for
// f-string interpolation), pushes the resulting string.
func (v *VM) execFormatValue(specIdx int) error {
	if len(v.stack) < 1 {
		return fmt.Errorf("vm: FORMAT_VALUE on empty stack")
	}
	spec, ok := v.mod.Constants[specIdx].(string)
	if !ok {
		return fmt.Errorf("vm: FORMAT_VALUE spec is not a string")
	}
	val := v.stack[len(v.stack)-1]
	s, err := strfmt.Format(val, spec)
	if err != nil {
		return fmt.Errorf("vm: %v", err)
	}
	v.stack[len(v.stack)-1] = s
	return nil
}
