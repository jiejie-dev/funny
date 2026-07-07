package vm

import (
	"fmt"

	"github.com/jiejie-dev/funny/v2/internal/bytecode"
)

// acquireLocals returns a locals slice with length n, reusing pooled storage.
func (v *VM) acquireLocals(n int) []bytecode.Value {
	if n == 0 {
		return nil
	}
	if lp := len(v.localsPool); lp > 0 {
		loc := v.localsPool[lp-1]
		v.localsPool = v.localsPool[:lp-1]
		if cap(loc) >= n {
			loc = loc[:n]
			clear(loc)
			return loc
		}
	}
	return make([]bytecode.Value, n)
}

func (v *VM) releaseLocals(loc []bytecode.Value) {
	if loc == nil {
		return
	}
	v.localsPool = append(v.localsPool, loc[:0])
}

// reset clears stack and frames between runs, recycling local slots.
func (v *VM) reset() {
	for i := range v.frames {
		v.releaseLocals(v.frames[i].locals)
	}
	v.stack = v.stack[:0]
	v.frames = v.frames[:0]
}

// execCallFast handles CALL with pooled locals (no per-call args slice).
func (v *VM) execCallFast(fnIdx int) error {
	if fnIdx < 0 || fnIdx >= len(v.mod.Functions) {
		return fmt.Errorf("vm: CALL invalid function index %d", fnIdx)
	}
	callee := v.mod.Functions[fnIdx]
	n := callee.Arity
	if len(v.stack) < n {
		return fmt.Errorf("vm: CALL %s expects %d args, got %d", callee.Name, n, len(v.stack))
	}
	locals := v.acquireLocals(callee.NumLocals)
	top := len(v.stack)
	base := top - n
	for i := 0; i < n; i++ {
		locals[i] = v.stack[base+i]
	}
	v.stack = v.stack[:base]
	v.frames = append(v.frames, Frame{fn: callee, locals: locals})
	return nil
}

// execReturnFast handles RETURN with locals pooling.
func (v *VM) execReturnFast() error {
	if len(v.frames) == 0 {
		return fmt.Errorf("vm: RETURN with no frames")
	}
	fi := len(v.frames) - 1
	var retVal bytecode.Value
	if len(v.stack) > 0 {
		retVal = v.stack[len(v.stack)-1]
		v.stack = v.stack[:len(v.stack)-1]
	}
	v.releaseLocals(v.frames[fi].locals)
	v.frames = v.frames[:fi]
	if retVal != nil {
		v.stack = append(v.stack, retVal)
	}
	return nil
}

// step executes one instruction for frame fi. Separated to keep execute readable.
func (v *VM) step(fi int, instr bytecode.Instruction) error {
	frame := &v.frames[fi]
	stack := &v.stack
	consts := v.mod.Constants

	switch instr.Op {
	case bytecode.PUSH_INT, bytecode.PUSH_FLOAT, bytecode.PUSH_STR, bytecode.PUSH_BOOL:
		*stack = append(*stack, consts[instr.Arg])
	case bytecode.PUSH_NIL:
		*stack = append(*stack, nil)
	case bytecode.POP:
		if len(*stack) == 0 {
			return fmt.Errorf("vm: POP on empty stack")
		}
		*stack = (*stack)[:len(*stack)-1]
	case bytecode.DUP:
		if len(*stack) == 0 {
			return fmt.Errorf("vm: DUP on empty stack")
		}
		*stack = append(*stack, (*stack)[len(*stack)-1])
	case bytecode.LOAD_LOCAL:
		if instr.Arg >= len(frame.locals) {
			return fmt.Errorf("vm: LOAD_LOCAL %d out of range", instr.Arg)
		}
		*stack = append(*stack, frame.locals[instr.Arg])
	case bytecode.STORE_LOCAL:
		if len(*stack) == 0 {
			return fmt.Errorf("vm: STORE_LOCAL on empty stack")
		}
		if instr.Arg >= len(frame.locals) {
			return fmt.Errorf("vm: STORE_LOCAL %d out of range", instr.Arg)
		}
		frame.locals[instr.Arg] = (*stack)[len(*stack)-1]
	case bytecode.ADD_INT:
		if len(*stack) < 2 {
			return fmt.Errorf("vm: ADD_INT underflow")
		}
		b := (*stack)[len(*stack)-1].(int)
		a := (*stack)[len(*stack)-2].(int)
		*stack = (*stack)[:len(*stack)-2]
		*stack = append(*stack, a+b)
	case bytecode.SUB_INT:
		if len(*stack) < 2 {
			return fmt.Errorf("vm: SUB_INT underflow")
		}
		b := (*stack)[len(*stack)-1].(int)
		a := (*stack)[len(*stack)-2].(int)
		*stack = (*stack)[:len(*stack)-2]
		*stack = append(*stack, a-b)
	case bytecode.LT_INT:
		if len(*stack) < 2 {
			return fmt.Errorf("vm: LT_INT underflow")
		}
		b := (*stack)[len(*stack)-1].(int)
		a := (*stack)[len(*stack)-2].(int)
		*stack = (*stack)[:len(*stack)-2]
		*stack = append(*stack, a < b)
	case bytecode.JUMP:
		frame.ip = instr.Arg
	case bytecode.JUMP_IF_FALSE:
		if len(*stack) == 0 {
			return fmt.Errorf("vm: JUMP_IF_FALSE on empty stack")
		}
		cond := (*stack)[len(*stack)-1]
		*stack = (*stack)[:len(*stack)-1]
		if b, ok := cond.(bool); ok && !b {
			frame.ip = instr.Arg
		}
	case bytecode.CALL:
		return v.execCallFast(instr.Arg)
	case bytecode.RETURN:
		return v.execReturnFast()
	case bytecode.HALT:
		return errHalt
	default:
		return v.stepSlow(fi, instr)
	}
	return nil
}

var errHalt = fmt.Errorf("halt")

func (v *VM) stepSlow(fi int, instr bytecode.Instruction) error {
	frame := &v.frames[fi]
	switch instr.Op {
	case bytecode.ADD_FLOAT, bytecode.SUB_FLOAT, bytecode.MUL_FLOAT, bytecode.DIV_FLOAT,
		bytecode.MUL_INT, bytecode.DIV_INT, bytecode.MOD_INT, bytecode.ADD_STR:
		a, b := v.pop2()
		res, err := v.execArith(instr.Op, a, b)
		if err != nil {
			return err
		}
		v.stack = append(v.stack, res)
	case bytecode.EQ_INT, bytecode.EQ_STR, bytecode.EQ_BOOL, bytecode.EQ_NIL, bytecode.EQ_FLOAT,
		bytecode.GT_INT, bytecode.LTE_INT, bytecode.GTE_INT,
		bytecode.LT_FLOAT, bytecode.GT_FLOAT, bytecode.LTE_FLOAT, bytecode.GTE_FLOAT,
		bytecode.AND_BOOL, bytecode.OR_BOOL:
		a, b := v.pop2()
		res, err := v.execCmp(instr.Op, a, b)
		if err != nil {
			return err
		}
		v.stack = append(v.stack, res)
	case bytecode.IN_LIST:
		elem, list := v.pop2()
		v.stack = append(v.stack, v.execInList(elem, list))
	case bytecode.NEG_INT, bytecode.NEG_FLOAT, bytecode.NOT_BOOL:
		a := v.pop()
		res, err := v.execUnary(instr.Op, a)
		if err != nil {
			return err
		}
		v.stack = append(v.stack, res)
	case bytecode.JUMP_IF_TRUE:
		if len(v.stack) == 0 {
			return fmt.Errorf("vm: JUMP_IF_TRUE on empty stack")
		}
		cond := v.stack[len(v.stack)-1]
		v.stack = v.stack[:len(v.stack)-1]
		if b, ok := cond.(bool); ok && b {
			frame.ip = instr.Arg
		}
	case bytecode.TRY_OR_RETURN:
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: TRY_OR_RETURN on empty stack")
		}
		top := v.stack[len(v.stack)-1]
		if isResult(top) && resultTag(top) == "err" && len(v.frames) > 1 {
			if err := v.execReturnFast(); err != nil {
				return err
			}
		}
	case bytecode.CALL_BUILTIN:
		if err := v.execCallBuiltin(instr.Arg); err != nil {
			return err
		}
	case bytecode.BUILD_LIST:
		v.execBuildList(instr.Arg)
	case bytecode.INDEX:
		if err := v.execIndex(); err != nil {
			return err
		}
	case bytecode.SET_INDEX:
		if err := v.execSetIndex(); err != nil {
			return err
		}
	case bytecode.BUILD_MAP:
		v.execBuildMap(instr.Arg)
	case bytecode.GET_FIELD:
		if err := v.execGetField(); err != nil {
			return err
		}
	case bytecode.SET_FIELD:
		if err := v.execSetField(); err != nil {
			return err
		}
	case bytecode.NEW_STRUCT:
		v.execNewStruct(instr.Arg)
	case bytecode.FORMAT_VALUE:
		if err := v.execFormatValue(instr.Arg); err != nil {
			return err
		}
	case bytecode.LOAD_GLOBAL, bytecode.STORE_GLOBAL:
		return fmt.Errorf("vm: %s not implemented", instr.Op)
	default:
		return fmt.Errorf("vm: unsupported op %s at ip=%d", instr.Op, frame.ip-1)
	}
	return nil
}
