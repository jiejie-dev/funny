// v2/internal/bytecode/code.go
package bytecode

import (
	"fmt"
	"strings"
)

// Value is a runtime value passed on the operand stack.
// Uses interface{} (any) since Go values can be int/float64/string/bool/nil/[]any/map[string]any.
type Value = any

// BuiltinInfo identifies a builtin function callable via CALL_BUILTIN.
// The compiler emits one BuiltinInfo per call so the VM knows how many
// arguments to pop from the operand stack.
type BuiltinInfo struct {
	Name  string
	Arity int
}

// Instruction is a single bytecode instruction.
type Instruction struct {
	Op  OpCode
	Arg int // operand (constant index, local index, jump target, etc.)
}

// String renders an instruction for disassembly.
func (i Instruction) String() string {
	if i.Arg == 0 {
		return string(i.Op)
	}
	return fmt.Sprintf("%s %d", i.Op, i.Arg)
}

// Function is a compiled function body.
type Function struct {
	Name      string
	Arity     int
	NumLocals int
	Code      []Instruction
}

// Emit appends a single instruction to the function body.
func (f *Function) Emit(op OpCode, arg int) {
	f.Code = append(f.Code, Instruction{Op: op, Arg: arg})
}

// Module is a compilation unit (one .fn file produces one Module).
type Module struct {
	Name      string
	Constants []Value
	Functions []*Function
}

// NewModule creates an empty Module with the given name.
func NewModule(name string) *Module {
	return &Module{Name: name}
}

// AddConstant adds a constant to the pool, de-duplicating by value.
// Returns the index of the constant.
func (m *Module) AddConstant(v Value) int {
	for i, c := range m.Constants {
		if valueEqual(c, v) {
			return i
		}
	}
	m.Constants = append(m.Constants, v)
	return len(m.Constants) - 1
}

// AddFunction registers a function and returns its index.
func (m *Module) AddFunction(f *Function) int {
	m.Functions = append(m.Functions, f)
	return len(m.Functions) - 1
}

// valueEqual compares two runtime values for constant-pool deduplication.
// Note: uses == for primitives; for slices/maps would need deep comparison (not needed for M2-B constants).
func valueEqual(a, b Value) bool {
	return a == b
}

// Disassemble returns a human-readable form of the module for debugging.
func (m *Module) Disassemble() string {
	var b strings.Builder
	fmt.Fprintf(&b, "module %s\n", m.Name)
	for i, fn := range m.Functions {
		fmt.Fprintf(&b, "  fn %d %s arity=%d locals=%d\n", i, fn.Name, fn.Arity, fn.NumLocals)
		for j, instr := range fn.Code {
			fmt.Fprintf(&b, "    %4d %s\n", j, instr.String())
		}
	}
	return b.String()
}
