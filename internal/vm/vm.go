package vm

import (
	"fmt"

	"github.com/jiejie-dev/funny/v2/internal/bytecode"
)

// Frame is a function call frame.
type Frame struct {
	fn     *bytecode.Function
	ip     int // instruction pointer within fn.Code
	locals []bytecode.Value
}

// VM is a stack-based bytecode interpreter.
type VM struct {
	mod        *bytecode.Module
	stack      []bytecode.Value
	frames     []Frame
	localsPool [][]bytecode.Value
	dbg        *Debugger
}

// New creates a VM ready to run the given module.
func New(mod *bytecode.Module) *VM {
	return &VM{
		mod:        mod,
		stack:      make([]bytecode.Value, 0, 512),
		frames:     make([]Frame, 0, 64),
		localsPool: make([][]bytecode.Value, 0, 64),
	}
}

// SetDebugger attaches a debugger for RunDebug or breakpoint stepping.
func (v *VM) SetDebugger(d *Debugger) {
	v.dbg = d
}

// Run executes the module's first function (main) and returns the top of stack.
func (v *VM) Run() (bytecode.Value, error) {
	return v.runFrom(0)
}

// RunDebug executes with the attached debugger, pausing on breakpoints/steps.
func (v *VM) RunDebug(d *Debugger) (bytecode.Value, error) {
	v.dbg = d
	d.StepOnce()
	return v.runFrom(0)
}

func (v *VM) runFrom(fnIdx int) (bytecode.Value, error) {
	if fnIdx < 0 || fnIdx >= len(v.mod.Functions) {
		return nil, fmt.Errorf("vm: module has no functions")
	}
	v.reset()
	main := v.mod.Functions[fnIdx]
	locals := v.acquireLocals(main.NumLocals)
	v.frames = append(v.frames, Frame{fn: main, locals: locals})
	return v.execute()
}

func (v *VM) execute() (bytecode.Value, error) {
	for {
		if len(v.frames) == 0 {
			if len(v.stack) > 0 {
				return v.stack[len(v.stack)-1], nil
			}
			return nil, nil
		}
		fi := len(v.frames) - 1
		frame := &v.frames[fi]
		if frame.ip >= len(frame.fn.Code) {
			return nil, fmt.Errorf("vm: ip out of bounds at %d", frame.ip)
		}
		if v.dbg != nil {
			action, err := v.dbg.beforeInstr(v, frame)
			if err != nil {
				return nil, err
			}
			if action == ActionQuit {
				return nil, fmt.Errorf("debug: stopped")
			}
		}
		instr := frame.fn.Code[frame.ip]
		frame.ip++
		if err := v.step(fi, instr); err != nil {
			if err == errHalt {
				if len(v.stack) > 0 {
					return v.stack[len(v.stack)-1], nil
				}
				return nil, nil
			}
			return nil, err
		}
	}
}
