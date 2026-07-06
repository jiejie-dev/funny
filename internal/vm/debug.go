package vm

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jiejie-dev/funny/v2/internal/bytecode"
)

// DebugAction is the user's response after the VM pauses.
type DebugAction int

const (
	ActionContinue DebugAction = iota
	ActionStep
	ActionQuit
)

// DebugEvent describes the VM state at a pause point.
type DebugEvent struct {
	FnIndex    int
	FnName     string
	IP         int
	Instruction bytecode.Instruction
	Location   bytecode.SourceLoc
	Stack      []bytecode.Value
	Locals     []NamedValue
}

// NamedValue pairs a local slot name with its runtime value.
type NamedValue struct {
	Name  string
	Value bytecode.Value
}

// Debugger controls breakpoints and single-stepping for a VM run.
type Debugger struct {
	breakpoints map[string]map[int]struct{} // normalized file → 0-based line
	stepOnce    bool
	handler     func(DebugEvent) (DebugAction, error)
}

// NewDebugger creates a debugger that pauses on handler invocation.
func NewDebugger(handler func(DebugEvent) (DebugAction, error)) *Debugger {
	return &Debugger{
		breakpoints: map[string]map[int]struct{}{},
		handler:     handler,
	}
}

// SetBreakpoint registers a breakpoint at line (1-based for CLI convenience).
func (d *Debugger) SetBreakpoint(file string, line1 int) {
	file = normalizeDebugFile(file)
	if d.breakpoints[file] == nil {
		d.breakpoints[file] = map[int]struct{}{}
	}
	d.breakpoints[file][line1-1] = struct{}{}
}

// StepOnce requests a pause before the next instruction executes.
func (d *Debugger) StepOnce() {
	d.stepOnce = true
}

func (d *Debugger) shouldPause(loc bytecode.SourceLoc) bool {
	if d.stepOnce {
		return true
	}
	if loc.IsZero() {
		return false
	}
	file := normalizeDebugFile(loc.File)
	if lines, ok := d.breakpoints[file]; ok {
		if _, hit := lines[loc.Line]; hit {
			return true
		}
	}
	// Also match basename when the user set a breakpoint without a path.
	base := filepath.Base(loc.File)
	if base != file {
		if lines, ok := d.breakpoints[base]; ok {
			if _, hit := lines[loc.Line]; hit {
				return true
			}
		}
	}
	return false
}

func (d *Debugger) beforeInstr(v *VM, frame *Frame) (DebugAction, error) {
	loc := bytecode.SourceLoc{}
	if frame.ip < len(frame.fn.Locations) {
		loc = frame.fn.Locations[frame.ip]
	}
	if !d.shouldPause(loc) {
		return ActionContinue, nil
	}
	d.stepOnce = false
	if d.handler == nil {
		return ActionContinue, nil
	}
	ev := v.buildDebugEvent(frame, loc)
	action, err := d.handler(ev)
	if err != nil {
		return ActionQuit, err
	}
	switch action {
	case ActionStep:
		d.stepOnce = true
	case ActionQuit:
		return ActionQuit, nil
	}
	return ActionContinue, nil
}

func (v *VM) buildDebugEvent(frame *Frame, loc bytecode.SourceLoc) DebugEvent {
	instr := frame.fn.Code[frame.ip]
	locals := make([]NamedValue, 0, len(frame.locals))
	for i, val := range frame.locals {
		name := fmt.Sprintf("$%d", i)
		if i < len(frame.fn.LocalNames) && frame.fn.LocalNames[i] != "" {
			name = frame.fn.LocalNames[i]
		}
		locals = append(locals, NamedValue{Name: name, Value: val})
	}
	stack := make([]bytecode.Value, len(v.stack))
	copy(stack, v.stack)
	fnIdx := 0
	for i, f := range v.mod.Functions {
		if f == frame.fn {
			fnIdx = i
			break
		}
	}
	return DebugEvent{
		FnIndex:     fnIdx,
		FnName:      frame.fn.Name,
		IP:          frame.ip,
		Instruction: instr,
		Location:    loc,
		Stack:       stack,
		Locals:      locals,
	}
}

func normalizeDebugFile(file string) string {
	file = strings.TrimSpace(file)
	file = filepath.Clean(file)
	return file
}

// FormatValue renders a stack/local value for debugger output.
func FormatValue(v bytecode.Value) string {
	if v == nil {
		return "nil"
	}
	switch x := v.(type) {
	case string:
		return fmt.Sprintf("%q", x)
	default:
		return fmt.Sprintf("%v", x)
	}
}
