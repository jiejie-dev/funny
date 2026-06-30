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

// execute runs the top frame's instructions until HALT.
func (v *VM) execute() (bytecode.Value, error) {
	frame := v.frames[len(v.frames)-1]
	for {
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
		case bytecode.HALT:
			if len(v.stack) > 0 {
				return v.stack[len(v.stack)-1], nil
			}
			return nil, nil
		default:
			return nil, fmt.Errorf("vm: unsupported op %s at ip=%d (not yet implemented in this task)", instr.Op, frame.ip-1)
		}
	}
}