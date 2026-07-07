// v2/internal/compiler/compiler.go
package compiler

import (
	"fmt"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/bytecode"
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
	mod          *bytecode.Module
	fn           *bytecode.Function
	pos          ast.Pos // source position for the next emitted instruction
	scopes       []map[string]int
	varTypes     []valueType                     // indexed by local slot (parallel to NumLocals)
	functions    map[string]int                  // function name → index in mod.Functions
	fnRetTypes   map[string]valueType            // function name → declared return value type
	structFields map[string]map[string]valueType // struct name → field name → value type
	loopStack    []loopFrame                     // active loops for break/continue
}

type loopFrame struct {
	breakPatches    []int
	continuePatches []int
}

// Compile translates a typed Program into a Module.
func Compile(prog *ast.Program, name string) (*bytecode.Module, error) {
	c := &Compiler{
		mod:          bytecode.NewModule(name),
		scopes:       []map[string]int{{}},
		functions:    map[string]int{},
		fnRetTypes:   map[string]valueType{},
		structFields: map[string]map[string]valueType{},
	}
	// Two passes so struct A can have a field typed as struct B regardless
	// of which one is declared first: pass 1 registers every struct name
	// (so annotationValueType recognizes it as *some* struct), pass 2 fills
	// in field types now that all names are known.
	for _, s := range prog.Stmts {
		if sd, ok := s.(*ast.StructDecl); ok {
			c.structFields[sd.Name] = map[string]valueType{}
		}
	}
	for _, s := range prog.Stmts {
		if sd, ok := s.(*ast.StructDecl); ok {
			for _, p := range sd.Fields {
				c.structFields[sd.Name][p.Name] = annotationValueType(p.TypeAnn, c.structFields)
			}
		}
	}
	mainFn := &bytecode.Function{Name: "main", Arity: 0}
	c.mod.AddFunction(mainFn)
	c.fn = mainFn
	c.functions["main"] = 0
	lastMeaningful := -1
	for i := len(prog.Stmts) - 1; i >= 0; i-- {
		if _, isComment := prog.Stmts[i].(*ast.CommentStmt); isComment {
			continue
		}
		lastMeaningful = i
		break
	}
	for i, s := range prog.Stmts {
		isLast := i == lastMeaningful
		if err := c.compileStmt(s, isLast); err != nil {
			return nil, err
		}
	}
	if len(prog.Stmts) > 0 {
		c.pos = prog.Stmts[lastMeaningful].Pos()
	}
	c.emit(bytecode.HALT, 0)
	return c.mod, nil
}

// emit records c.pos alongside the instruction for debugger source maps.
func (c *Compiler) emit(op bytecode.OpCode, arg int) {
	loc := bytecode.SourceLoc{File: c.pos.File, Line: c.pos.Line, Col: c.pos.Col}
	c.fn.EmitAt(op, arg, loc)
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
	for len(c.fn.LocalNames) <= idx {
		c.fn.LocalNames = append(c.fn.LocalNames, "")
	}
	c.fn.LocalNames[idx] = name
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
	c.pos = s.Pos()
	switch n := s.(type) {
	case *ast.ExprStmt:
		if _, err := c.compileExpr(n.X); err != nil {
			return err
		}
		// POP only if the expression leaves a value on the stack.
		// Function calls (e.g. println) consume their own args and don't push
		// a result, so POPping after them would underflow.
		if !isLast {
			if _, isCall := n.X.(*ast.CallExpr); !isCall {
				c.emit(bytecode.POP, 0)
			}
		}
		return nil
	case *ast.LetStmt:
		return c.compileLet(n)
	case *ast.AssignStmt:
		return c.compileAssign(n)
	case *ast.IfStmt:
		return c.compileIf(n, isLast)
	case *ast.WhileStmt:
		return c.compileWhile(n)
	case *ast.ForStmt:
		return c.compileFor(n)
	case *ast.MatchStmt:
		return c.compileMatch(n)
	case *ast.BreakStmt:
		return c.compileBreak()
	case *ast.ContinueStmt:
		return c.compileContinue()
	case *ast.FnDecl:
		return c.compileFnDecl(n)
	case *ast.ReturnStmt:
		return c.compileReturn(n)
	case *ast.StructDecl:
		return nil
	case *ast.CommentStmt:
		return nil
	case *ast.ImportDecl:
		return nil
	case *ast.MetaBlock:
		return nil
	case *ast.PlanBlock:
		return nil
	case *ast.TestBlock:
		return nil
	}
	return fmt.Errorf("compileStmt: unsupported statement type %T", s)
}

func (c *Compiler) compileLet(n *ast.LetStmt) error {
	vt, err := c.compileExpr(n.Value)
	if err != nil {
		return err
	}
	// compileExpr reports valNil ("untracked") for things like an empty
	// list/map literal (`let xs: list[LogEntry] = []`) that carry no
	// element-type information of their own to infer from. The type
	// checker already trusts the explicit annotation for exactly this
	// case (see checkLet's isEmptyContainerLiteral handling), so fall
	// back to it here too rather than leaving `xs` permanently valNil:
	// otherwise every later `xs[i].field` or `for e in xs` derived type
	// silently degrades to "untracked", which used to make comparisons
	// like `e.response_ms > xs[i].response_ms` fail to compile at all
	// ("unsupported op > for nil") once both sides ended up untracked.
	if vt == valNil && n.TypeAnn != "" {
		if at := annotationValueType(n.TypeAnn, c.structFields); at != valNil {
			vt = at
		}
	}
	slot := c.declareLocal(n.Name, vt)
	c.emit(bytecode.STORE_LOCAL, slot)
	c.emit(bytecode.POP, 0)
	return nil
}

func (c *Compiler) compileAssign(n *ast.AssignStmt) error {
	if fe, ok := n.Target.(*ast.FieldExpr); ok {
		return c.compileFieldAssign(fe, n.Value)
	}
	if idx, ok := n.Target.(*ast.IndexExpr); ok {
		return c.compileIndexAssign(idx, n.Value)
	}
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
	c.emit(bytecode.STORE_LOCAL, slot)
	c.emit(bytecode.POP, 0)
	return nil
}

// compileIndexAssign compiles `obj[idx] = value` into SET_INDEX. Push order
// (value, object, index) mirrors execSetIndex's stack layout, which pops
// index and object and leaves value on top for the trailing POP.
func (c *Compiler) compileIndexAssign(idx *ast.IndexExpr, value ast.Expression) error {
	if _, err := c.compileExpr(value); err != nil {
		return err
	}
	if _, err := c.compileExpr(idx.Object); err != nil {
		return err
	}
	if _, err := c.compileExpr(idx.Index); err != nil {
		return err
	}
	c.emit(bytecode.SET_INDEX, 0)
	c.emit(bytecode.POP, 0)
	return nil
}

// compileFieldAssign compiles `obj.field = value` into SET_FIELD. Stack layout
// (bottom to top): value, object, field name — mirroring SET_INDEX.
func (c *Compiler) compileFieldAssign(fe *ast.FieldExpr, value ast.Expression) error {
	if _, err := c.compileExpr(value); err != nil {
		return err
	}
	if _, err := c.compileExpr(fe.Object); err != nil {
		return err
	}
	nameIdx := c.mod.AddConstant(fe.Field)
	c.emit(bytecode.PUSH_STR, nameIdx)
	c.emit(bytecode.SET_FIELD, 0)
	c.emit(bytecode.POP, 0)
	return nil
}
