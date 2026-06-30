// v2/internal/compiler/control.go
package compiler

import (
	"fmt"

	"github.com/jerloo/funny/v2/internal/ast"
	"github.com/jerloo/funny/v2/internal/bytecode"
)

// compileIf translates: `if cond: then else:?` into JUMP_IF_FALSE + then + JUMP + else.
//
// Emitted layout:
//   <compile cond>
//   JUMP_IF_FALSE <thenEnd>     ; jump if false
//   <compile then>
//   JUMP <end>                  ; skip else
//   thenEnd:
//   <compile else (if any)>
//   end:
func (c *Compiler) compileIf(n *ast.IfStmt) error {
	if _, err := c.compileExpr(n.Cond); err != nil {
		return err
	}
	jumpIfFalseIdx := len(c.fn.Code)
	c.fn.Emit(bytecode.JUMP_IF_FALSE, 0) // placeholder

	if err := c.compileBlock(n.Then); err != nil {
		return err
	}

	if n.ElseBlock == nil {
		// Patch: JUMP_IF_FALSE target = current end
		c.fn.Code[jumpIfFalseIdx].Arg = len(c.fn.Code)
		return nil
	}

	// Has else: emit JUMP over else
	jumpOverElseIdx := len(c.fn.Code)
	c.fn.Emit(bytecode.JUMP, 0) // placeholder

	// Patch: JUMP_IF_FALSE target = current position (start of else)
	c.fn.Code[jumpIfFalseIdx].Arg = len(c.fn.Code)

	if err := c.compileBlock(n.ElseBlock); err != nil {
		return err
	}

	// Patch: JUMP over else target = current end
	c.fn.Code[jumpOverElseIdx].Arg = len(c.fn.Code)
	return nil
}

// compileWhile translates: `while cond: body` into:
//
//   loopStart:
//   <compile cond>
//   JUMP_IF_FALSE <loopEnd>
//   <compile body>
//   JUMP loopStart
//   loopEnd:
func (c *Compiler) compileWhile(n *ast.WhileStmt) error {
	loopStart := len(c.fn.Code)
	if _, err := c.compileExpr(n.Cond); err != nil {
		return err
	}
	exitJumpIdx := len(c.fn.Code)
	c.fn.Emit(bytecode.JUMP_IF_FALSE, 0) // placeholder

	if err := c.compileBlock(n.Body); err != nil {
		return err
	}

	c.fn.Emit(bytecode.JUMP, loopStart)

	// Patch: JUMP_IF_FALSE target = current end (after the JUMP back)
	c.fn.Code[exitJumpIdx].Arg = len(c.fn.Code)
	return nil
}

// compileBlock compiles a block of statements in a new scope.
func (c *Compiler) compileBlock(b *ast.Block) error {
	if b == nil {
		return fmt.Errorf("compileBlock: nil block")
	}
	c.pushScope()
	defer c.popScope()
	for _, s := range b.Statements {
		if err := c.compileStmt(s, false); err != nil {
			return err
		}
	}
	return nil
}