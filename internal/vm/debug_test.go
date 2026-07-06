package vm

import (
	"strings"
	"testing"

	"github.com/jiejie-dev/funny/v2/internal/bytecode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDebugger_BreakpointAndStep(t *testing.T) {
	fn := &bytecode.Function{Name: "main", NumLocals: 1, LocalNames: []string{"x"}}
	fn.EmitAt(bytecode.PUSH_INT, 0, bytecode.SourceLoc{File: "t.fn", Line: 0, Col: 0})
	fn.EmitAt(bytecode.STORE_LOCAL, 0, bytecode.SourceLoc{File: "t.fn", Line: 0, Col: 4})
	fn.EmitAt(bytecode.HALT, 0, bytecode.SourceLoc{File: "t.fn", Line: 1, Col: 0})
	mod := bytecode.NewModule("t.fn")
	mod.AddConstant(7)
	mod.AddFunction(fn)

	var log []string
	dbg := NewDebugger(func(ev DebugEvent) (DebugAction, error) {
		log = append(log, ev.Location.Display()+":"+string(ev.Instruction.Op))
		if len(log) >= 3 {
			return ActionQuit, nil
		}
		return ActionStep, nil
	})
	dbg.SetBreakpoint("t.fn", 1)

	m := New(mod)
	_, err := m.RunDebug(dbg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "debug: stopped")
	assert.GreaterOrEqual(t, len(log), 2)
}

func TestDebugger_ContinueToBreakpoint(t *testing.T) {
	fn := &bytecode.Function{Name: "main"}
	fn.EmitAt(bytecode.PUSH_INT, 0, bytecode.SourceLoc{File: "t.fn", Line: 0, Col: 0})
	fn.EmitAt(bytecode.PUSH_INT, 1, bytecode.SourceLoc{File: "t.fn", Line: 1, Col: 0})
	fn.EmitAt(bytecode.HALT, 0, bytecode.SourceLoc{File: "t.fn", Line: 2, Col: 0})
	mod := bytecode.NewModule("t.fn")
	mod.AddConstant(1)
	mod.AddConstant(2)
	mod.AddFunction(fn)

	steps := 0
	dbg := NewDebugger(func(ev DebugEvent) (DebugAction, error) {
		steps++
		if steps == 1 {
			return ActionContinue, nil
		}
		return ActionQuit, nil
	})
	dbg.SetBreakpoint("t.fn", 2)

	m := New(mod)
	_, err := m.RunDebug(dbg)
	require.Error(t, err)
	assert.Equal(t, 2, steps)
}

func TestFormatValue(t *testing.T) {
	assert.Equal(t, "nil", FormatValue(nil))
	assert.Equal(t, `"hi"`, FormatValue("hi"))
	assert.Equal(t, "42", FormatValue(42))
}

func TestInstructionStringInDebug(t *testing.T) {
	instr := bytecode.Instruction{Op: bytecode.PUSH_INT, Arg: 3}
	assert.True(t, strings.Contains(instr.String(), "PUSH_INT"))
}
