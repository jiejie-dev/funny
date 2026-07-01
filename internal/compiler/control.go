// v2/internal/compiler/control.go
package compiler

import (
	"fmt"

	"github.com/jiejie-dev/funny/internal/ast"
	"github.com/jiejie-dev/funny/internal/bytecode"
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

// compileFor compiles: for x in iterable: body
//
// Emitted layout (using list and index locals):
//
//	<compile iterable>
//	STORE_LOCAL __for_list__
//	POP
//	PUSH_INT 0
//	STORE_LOCAL __for_idx__
//	POP
// loopStart:
//	LOAD_LOCAL __for_idx__
//	LOAD_LOCAL __for_list__
//	CALL_BUILTIN "len"
//	LT_INT
//	JUMP_IF_FALSE loopEnd
//	LOAD_LOCAL __for_list__
//	LOAD_LOCAL __for_idx__
//	INDEX
//	STORE_LOCAL x
//	POP
//	<compile body>
//	LOAD_LOCAL __for_idx__
//	PUSH_INT <one_const>
//	ADD_INT
//	STORE_LOCAL __for_idx__
//	POP
//	JUMP loopStart
// loopEnd:
func (c *Compiler) compileFor(n *ast.ForStmt) error {
	c.pushScope()
	defer c.popScope()
	iterType, err := c.compileExpr(n.Iterable)
	if err != nil {
		return err
	}
	listSlot := c.declareLocal("__for_list__", valNil)
	c.fn.Emit(bytecode.STORE_LOCAL, listSlot)
	c.fn.Emit(bytecode.POP, 0)
	idxSlot := c.declareLocal("__for_idx__", valInt)
	c.fn.Emit(bytecode.PUSH_INT, 0)
	c.fn.Emit(bytecode.STORE_LOCAL, idxSlot)
	c.fn.Emit(bytecode.POP, 0)
	loopStart := len(c.fn.Code)
	c.fn.Emit(bytecode.LOAD_LOCAL, idxSlot)
	c.fn.Emit(bytecode.LOAD_LOCAL, listSlot)
	nameIdx := c.mod.AddConstant("len")
	c.fn.Emit(bytecode.CALL_BUILTIN, nameIdx)
	c.fn.Emit(bytecode.LT_INT, 0)
	exitJump := len(c.fn.Code)
	c.fn.Emit(bytecode.JUMP_IF_FALSE, 0)
	c.fn.Emit(bytecode.LOAD_LOCAL, listSlot)
	c.fn.Emit(bytecode.LOAD_LOCAL, idxSlot)
	c.fn.Emit(bytecode.INDEX, 0)
	if iterType == "" {
		iterType = valNil
	}
	userSlot := c.declareLocal(n.Name, iterType)
	c.fn.Emit(bytecode.STORE_LOCAL, userSlot)
	c.fn.Emit(bytecode.POP, 0)
	if err := c.compileBlock(n.Body); err != nil {
		return err
	}
	c.fn.Emit(bytecode.LOAD_LOCAL, idxSlot)
	oneIdx := c.mod.AddConstant(1)
	c.fn.Emit(bytecode.PUSH_INT, oneIdx)
	c.fn.Emit(bytecode.ADD_INT, 0)
	c.fn.Emit(bytecode.STORE_LOCAL, idxSlot)
	c.fn.Emit(bytecode.POP, 0)
	c.fn.Emit(bytecode.JUMP, loopStart)
	c.fn.Code[exitJump].Arg = len(c.fn.Code)
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