// v2/internal/compiler/compiler.go
package compiler

import (
	"fmt"

	"github.com/jerloo/funny/v2/internal/ast"
	"github.com/jerloo/funny/v2/internal/bytecode"
)

// valueType is the runtime value type produced by an expression.
// It is tracked separately from the emitted OpCode so that variables
// (which emit LOAD_LOCAL / LOAD_GLOBAL) can participate in type-sensitive
// operators like `+`, `<`, `==`.
type valueType string

const (
	valInt   valueType = "int"
	valFloat valueType = "float"
	valStr   valueType = "str"
	valBool  valueType = "bool"
	valNil   valueType = "nil"
)

// Compiler translates a typed AST into bytecode.
type Compiler struct {
	mod       *bytecode.Module
	fn        *bytecode.Function
	scopes    []map[string]int
	varTypes  []valueType  // indexed by local slot (parallel to NumLocals)
	functions map[string]int // function name → index in mod.Functions
}

// Compile translates a typed Program into a Module.
func Compile(prog *ast.Program, name string) (*bytecode.Module, error) {
	c := &Compiler{
		mod:       bytecode.NewModule(name),
		scopes:    []map[string]int{{}},
		functions: map[string]int{},
	}
	mainFn := &bytecode.Function{Name: "main", Arity: 0}
	c.mod.AddFunction(mainFn)
	c.fn = mainFn
	c.functions["main"] = 0
	for i, s := range prog.Stmts {
		isLast := i == len(prog.Stmts)-1
		if err := c.compileStmt(s, isLast); err != nil {
			return nil, err
		}
	}
	c.fn.Emit(bytecode.HALT, 0)
	return c.mod, nil
}

func (c *Compiler) pushScope() {
	c.scopes = append(c.scopes, map[string]int{})
}

func (c *Compiler) popScope() {
	c.scopes = c.scopes[:len(c.scopes)-1]
}

// declareLocal reserves a slot for `name` and records its value type `vt`
// so subsequent VariableExpr lookups can produce the right value type.
func (c *Compiler) declareLocal(name string, vt valueType) int {
	scope := c.scopes[len(c.scopes)-1]
	if idx, ok := scope[name]; ok {
		// Re-declaration in same scope (e.g. `let x = ...; let x = ...`)
		// Update the recorded type to match the new binding.
		if idx < len(c.varTypes) {
			c.varTypes[idx] = vt
		}
		return idx
	}
	idx := c.fn.NumLocals
	scope[name] = idx
	for len(c.varTypes) <= idx {
		c.varTypes = append(c.varTypes, valNil)
	}
	c.varTypes[idx] = vt
	c.fn.NumLocals++
	return idx
}

// lookupLocal returns the slot index and value type for a local variable.
// Returns (-1, "") if not found.
func (c *Compiler) lookupLocal(name string) (int, valueType) {
	for i := len(c.scopes) - 1; i >= 0; i-- {
		if idx, ok := c.scopes[i][name]; ok {
			var vt valueType
			if idx < len(c.varTypes) {
				vt = c.varTypes[idx]
			}
			return idx, vt
		}
	}
	return -1, ""
}

func (c *Compiler) compileStmt(s ast.Statement, isLast bool) error {
	switch n := s.(type) {
	case *ast.ExprStmt:
		if _, err := c.compileExpr(n.X); err != nil {
			return err
		}
		if !isLast {
			c.fn.Emit(bytecode.POP, 0)
		}
		return nil
	case *ast.LetStmt:
		return c.compileLet(n)
	case *ast.AssignStmt:
		return c.compileAssign(n)
	case *ast.IfStmt:
		return c.compileIf(n)
	case *ast.WhileStmt:
		return c.compileWhile(n)
	case *ast.ForStmt:
		return fmt.Errorf("compileStmt: for-in loop not yet implemented (M2-B.5 follow-up)")
	case *ast.FnDecl:
		return c.compileFnDecl(n)
	case *ast.ReturnStmt:
		return c.compileReturn(n)
	}
	return fmt.Errorf("compileStmt: unsupported statement type %T", s)
}

func (c *Compiler) compileLet(n *ast.LetStmt) error {
	vt, err := c.compileExpr(n.Value)
	if err != nil {
		return err
	}
	slot := c.declareLocal(n.Name, vt)
	c.fn.Emit(bytecode.STORE_LOCAL, slot)
	c.fn.Emit(bytecode.POP, 0)
	return nil
}

func (c *Compiler) compileAssign(n *ast.AssignStmt) error {
	if _, err := c.compileExpr(n.Value); err != nil {
		return err
	}
	v, ok := n.Target.(*ast.VariableExpr)
	if !ok {
		return fmt.Errorf("compileAssign: target must be a variable (got %T)", n.Target)
	}
	slot, _ := c.lookupLocal(v.Name)
	if slot < 0 {
		return fmt.Errorf("compileAssign: undefined variable %s", v.Name)
	}
	c.fn.Emit(bytecode.STORE_LOCAL, slot)
	c.fn.Emit(bytecode.POP, 0)
	return nil
}
