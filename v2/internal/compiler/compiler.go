// v2/internal/compiler/compiler.go
package compiler

import (
	"fmt"

	"github.com/jerloo/funny/v2/internal/ast"
	"github.com/jerloo/funny/v2/internal/bytecode"
)

// Compiler translates a typed AST into bytecode.
type Compiler struct {
	mod    *bytecode.Module
	fn     *bytecode.Function
	scopes []map[string]int
}

// Compile translates a typed Program into a Module.
func Compile(prog *ast.Program, name string) (*bytecode.Module, error) {
	c := &Compiler{
		mod:    bytecode.NewModule(name),
		scopes: []map[string]int{{}},
	}
	mainFn := &bytecode.Function{Name: "main", Arity: 0}
	c.mod.AddFunction(mainFn)
	c.fn = mainFn
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

func (c *Compiler) declareLocal(name string) int {
	scope := c.scopes[len(c.scopes)-1]
	if idx, ok := scope[name]; ok {
		return idx
	}
	idx := c.fn.NumLocals
	scope[name] = idx
	c.fn.NumLocals++
	return idx
}

func (c *Compiler) lookupLocal(name string) int {
	for i := len(c.scopes) - 1; i >= 0; i-- {
		if idx, ok := c.scopes[i][name]; ok {
			return idx
		}
	}
	return -1
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
	}
	return fmt.Errorf("compileStmt: unsupported statement type %T (control flow not yet implemented)", s)
}

func (c *Compiler) compileLet(n *ast.LetStmt) error {
	if _, err := c.compileExpr(n.Value); err != nil {
		return err
	}
	slot := c.declareLocal(n.Name)
	c.fn.Emit(bytecode.STORE_LOCAL, slot)
	c.fn.Emit(bytecode.POP, 0)
	return nil
}
