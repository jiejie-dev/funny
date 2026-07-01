package bytecode

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstruction_String(t *testing.T) {
	instr := Instruction{Op: PUSH_INT, Arg: 42}
	s := instr.String()
	assert.Contains(t, s, "PUSH_INT")
	assert.Contains(t, s, "42")
}

func TestInstruction_StringNoArg(t *testing.T) {
	instr := Instruction{Op: HALT}
	assert.Equal(t, "HALT", instr.String())
}

func TestModule_AddConstant_Dedup(t *testing.T) {
	m := NewModule("test")
	i1 := m.AddConstant("hello")
	i2 := m.AddConstant("hello")
	i3 := m.AddConstant("world")
	assert.Equal(t, 0, i1)
	assert.Equal(t, 0, i2) // dedup
	assert.Equal(t, 1, i3)
}

func TestModule_AddConstant_Int(t *testing.T) {
	m := NewModule("test")
	assert.Equal(t, 0, m.AddConstant(42))
	assert.Equal(t, 1, m.AddConstant(99))
	assert.Equal(t, 0, m.AddConstant(42)) // dedup
}

func TestFunction_Emit(t *testing.T) {
	f := &Function{Name: "main", Arity: 0, NumLocals: 0}
	f.Emit(PUSH_INT, 1)
	f.Emit(PUSH_INT, 2)
	f.Emit(ADD_INT, 0)
	f.Emit(HALT, 0)
	assert.Len(t, f.Code, 4)
}

func TestModule_AddFunction(t *testing.T) {
	m := NewModule("test")
	fn1 := &Function{Name: "f1"}
	fn2 := &Function{Name: "f2"}
	i1 := m.AddFunction(fn1)
	i2 := m.AddFunction(fn2)
	assert.Equal(t, 0, i1)
	assert.Equal(t, 1, i2)
	assert.Len(t, m.Functions, 2)
}

func TestModule_Disassemble(t *testing.T) {
	m := NewModule("hello")
	fn := &Function{Name: "main", Arity: 0, NumLocals: 0}
	m.AddFunction(fn)
	fn.Emit(PUSH_INT, m.AddConstant(1))
	fn.Emit(HALT, 0)
	out := m.Disassemble()
	assert.Contains(t, out, "module hello")
	assert.Contains(t, out, "fn 0 main")
	assert.Contains(t, out, "PUSH_INT")
	assert.Contains(t, out, "HALT")
}
