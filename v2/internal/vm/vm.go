// v2/internal/vm/vm.go
package vm

import (
	"fmt"

	"github.com/jerloo/funny/v2/internal/bytecode"
)

// Frame is a function call frame.
type Frame struct {
	fn     *bytecode.Function
	ip     int // instruction pointer within fn.Code
	locals []bytecode.Value
}

// VM is a stack-based bytecode interpreter.
type VM struct {
	mod    *bytecode.Module
	stack  []bytecode.Value
	frames []*Frame
}

// New creates a VM ready to run the given module.
func New(mod *bytecode.Module) *VM {
	return &VM{mod: mod}
}

// Run executes the module's first function (main) and returns the top of stack.
// Returns nil if the stack is empty at HALT.
func (v *VM) Run() (bytecode.Value, error) {
	if len(v.mod.Functions) == 0 {
		return nil, fmt.Errorf("vm: module has no functions")
	}
	main := v.mod.Functions[0]
	v.frames = []*Frame{{fn: main, ip: 0, locals: make([]bytecode.Value, main.NumLocals)}}
	return v.execute()
}

// execute runs the top frame's instructions until HALT or until frames are empty (main RETURN).
func (v *VM) execute() (bytecode.Value, error) {
	for {
		if len(v.frames) == 0 {
			if len(v.stack) > 0 {
				return v.stack[len(v.stack)-1], nil
			}
			return nil, nil
		}
		frame := v.frames[len(v.frames)-1]
		if frame.ip >= len(frame.fn.Code) {
			return nil, fmt.Errorf("vm: ip out of bounds at %d", frame.ip)
		}
		instr := frame.fn.Code[frame.ip]
		frame.ip++
		switch instr.Op {
		case bytecode.PUSH_INT, bytecode.PUSH_FLOAT, bytecode.PUSH_STR, bytecode.PUSH_BOOL, bytecode.PUSH_NIL:
			if instr.Op == bytecode.PUSH_NIL {
				v.stack = append(v.stack, nil)
			} else {
				v.stack = append(v.stack, v.mod.Constants[instr.Arg])
			}
		case bytecode.POP:
			if len(v.stack) == 0 {
				return nil, fmt.Errorf("vm: POP on empty stack")
			}
			v.stack = v.stack[:len(v.stack)-1]
		case bytecode.DUP:
			if len(v.stack) == 0 {
				return nil, fmt.Errorf("vm: DUP on empty stack")
			}
			v.stack = append(v.stack, v.stack[len(v.stack)-1])
		case bytecode.LOAD_LOCAL:
			if instr.Arg >= len(frame.locals) {
				return nil, fmt.Errorf("vm: LOAD_LOCAL %d out of range", instr.Arg)
			}
			v.stack = append(v.stack, frame.locals[instr.Arg])
		case bytecode.STORE_LOCAL:
			if len(v.stack) == 0 {
				return nil, fmt.Errorf("vm: STORE_LOCAL on empty stack")
			}
			if instr.Arg >= len(frame.locals) {
				return nil, fmt.Errorf("vm: STORE_LOCAL %d out of range", instr.Arg)
			}
			frame.locals[instr.Arg] = v.stack[len(v.stack)-1]
		case bytecode.ADD_INT, bytecode.SUB_INT, bytecode.MUL_INT, bytecode.DIV_INT, bytecode.MOD_INT,
			bytecode.ADD_FLOAT, bytecode.SUB_FLOAT, bytecode.MUL_FLOAT, bytecode.DIV_FLOAT,
			bytecode.ADD_STR:
			a, b := v.pop2()
			res, err := v.execArith(instr.Op, a, b)
			if err != nil {
				return nil, err
			}
			v.stack = append(v.stack, res)
		case bytecode.EQ_INT, bytecode.EQ_STR, bytecode.EQ_BOOL, bytecode.EQ_NIL,
			bytecode.LT_INT, bytecode.GT_INT, bytecode.LTE_INT, bytecode.GTE_INT:
			a, b := v.pop2()
			res, err := v.execCmp(instr.Op, a, b)
			if err != nil {
				return nil, err
			}
			v.stack = append(v.stack, res)
		case bytecode.NEG_INT, bytecode.NEG_FLOAT, bytecode.NOT_BOOL:
			a := v.pop()
			res, err := v.execUnary(instr.Op, a)
			if err != nil {
				return nil, err
			}
			v.stack = append(v.stack, res)
		case bytecode.HALT:
			if len(v.stack) > 0 {
				return v.stack[len(v.stack)-1], nil
			}
			return nil, nil
		case bytecode.JUMP:
			frame.ip = instr.Arg
		case bytecode.JUMP_IF_FALSE:
			if len(v.stack) == 0 {
				return nil, fmt.Errorf("vm: JUMP_IF_FALSE on empty stack")
			}
			cond := v.stack[len(v.stack)-1]
			v.stack = v.stack[:len(v.stack)-1]
			b, isBool := cond.(bool)
			if isBool && !b {
				frame.ip = instr.Arg
			}
		case bytecode.JUMP_IF_TRUE:
			if len(v.stack) == 0 {
				return nil, fmt.Errorf("vm: JUMP_IF_TRUE on empty stack")
			}
			cond := v.stack[len(v.stack)-1]
			v.stack = v.stack[:len(v.stack)-1]
			b, isBool := cond.(bool)
			if isBool && b {
				frame.ip = instr.Arg
			}
		case bytecode.TRY_OR_RETURN:
			if len(v.stack) < 1 {
				return nil, fmt.Errorf("vm: TRY_OR_RETURN on empty stack")
			}
			top := v.stack[len(v.stack)-1]
			if !isResult(top) {
				// Non-Result operand: `?` is a no-op, leave the value as-is.
				break
			}
			if resultTag(top) == "err" && len(v.frames) > 1 {
				// Err inside a function: leave the Result on the stack; execReturn will
				// pop it, pop the current frame, and push it back as the caller's
				// return value.
				if err := v.execReturn(); err != nil {
					return nil, err
				}
			}
			// Ok, or Err at top level: leave the Result on the stack.
			// `?` propagates Err from a function but does NOT unwrap Ok.
			// At the top level there is no caller to return to, so the Result
			// stays on the stack for the program to inspect.
		case bytecode.CALL:
			if err := v.execCall(instr.Arg); err != nil {
				return nil, err
			}
		case bytecode.CALL_BUILTIN:
			if err := v.execCallBuiltin(instr.Arg); err != nil {
				return nil, err
			}
		case bytecode.RETURN:
			if err := v.execReturn(); err != nil {
				return nil, err
			}
		case bytecode.BUILD_LIST:
			v.execBuildList(instr.Arg)
		case bytecode.INDEX:
			if err := v.execIndex(); err != nil {
				return nil, err
			}
		case bytecode.BUILD_MAP:
			v.execBuildMap(instr.Arg)
		case bytecode.GET_FIELD:
			if err := v.execGetField(); err != nil {
				return nil, err
			}
		case bytecode.NEW_STRUCT:
			v.execNewStruct()
		default:
			return nil, fmt.Errorf("vm: unsupported op %s at ip=%d (not yet implemented in this task)", instr.Op, frame.ip-1)
		}
	}
}
