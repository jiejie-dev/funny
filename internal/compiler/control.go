// v2/internal/compiler/control.go
package compiler

import (
	"fmt"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/bytecode"
)

// compileIf translates: `if cond: then elif ...: ... else:? ...` into a
// chain of JUMP_IF_FALSE/JUMP pairs.
//
// The parser (see parseIf in internal/parser/statement.go) desugars an
// `elif` chain by parsing each `elif` as a nested *ast.IfStmt hanging off
// n.ElseIf, but *hoists* the chain's ultimate `else:` block up onto the
// outermost IfStmt's ElseBlock field (clearing it from every inner node)
// - this flattening is what lets the formatter print `elif`/`else` back
// out at a single indent level instead of nested `if/else: if/else: ...`.
// That means a naive walk that only ever looks at nesting level n's own
// n.ElseBlock (as this function used to) finds it non-nil the moment
// *any* branch in the chain has a trailing else, and treats it as the
// immediate else for `n.Cond` alone - silently skipping every
// intermediate elif's condition and body entirely. E.g. `if A: .. elif
// B: .. elif C: .. else: ..` compiled as just `if A: .. else: ..`,
// dropping B and C. compileIfChain fixes this by threading the hoisted
// final else block down through the recursion, only actually emitting it
// once the chain genuinely runs out of elifs.
//
// Emitted layout:
//   <compile cond>
//   JUMP_IF_FALSE <next>       ; jump if false
//   <compile then>
//   JUMP <end>                  ; skip elif/else chain
//   next:
//   <compile elif (recursively) or final else, if any>
//   end:
func (c *Compiler) compileIf(n *ast.IfStmt) error {
	return c.compileIfChain(n, n.ElseBlock)
}

func (c *Compiler) compileIfChain(n *ast.IfStmt, finalElse *ast.Block) error {
	if _, err := c.compileExpr(n.Cond); err != nil {
		return err
	}
	jumpIfFalseIdx := len(c.fn.Code)
	c.fn.Emit(bytecode.JUMP_IF_FALSE, 0) // placeholder

	if err := c.compileBlock(n.Then); err != nil {
		return err
	}

	if n.ElseIf == nil && finalElse == nil {
		// Patch: JUMP_IF_FALSE target = current end
		c.fn.Code[jumpIfFalseIdx].Arg = len(c.fn.Code)
		return nil
	}

	// Has elif and/or a final else: emit JUMP over it
	jumpOverElseIdx := len(c.fn.Code)
	c.fn.Emit(bytecode.JUMP, 0) // placeholder

	// Patch: JUMP_IF_FALSE target = current position (start of elif/else)
	c.fn.Code[jumpIfFalseIdx].Arg = len(c.fn.Code)

	if n.ElseIf != nil {
		// n.ElseIf.ElseBlock is always nil by construction (the parser
		// clears it during hoisting) - finalElse is threaded through
		// instead so the chain's real else is only compiled once, at
		// whichever elif actually falls through.
		if err := c.compileIfChain(n.ElseIf, finalElse); err != nil {
			return err
		}
	} else if finalElse != nil {
		if err := c.compileBlock(finalElse); err != nil {
			return err
		}
	}

	// Patch: JUMP over elif/else target = current end
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
	c.fn.Emit(bytecode.JUMP_IF_FALSE, 0)

	c.pushLoop()
	if err := c.compileBlock(n.Body); err != nil {
		return err
	}
	c.fn.Emit(bytecode.JUMP, loopStart)

	loopEnd := len(c.fn.Code)
	c.popLoop(loopEnd, loopStart)
	c.fn.Code[exitJumpIdx].Arg = loopEnd
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
	// Regression: this used to hardcode Arg=0, which reads whatever value
	// happens to sit at Constants[0] instead of the intended literal 0 -
	// e.g. `for i in [1, 2, 3]:` puts the list's own first element (1) at
	// Constants[0] (compileList's AddConstant call runs first), so the
	// loop's index silently started at 1 and skipped the first item on
	// every iterable whose first constant wasn't already the int 0.
	c.fn.Emit(bytecode.PUSH_INT, c.mod.AddConstant(0))
	c.fn.Emit(bytecode.STORE_LOCAL, idxSlot)
	c.fn.Emit(bytecode.POP, 0)
	loopStart := len(c.fn.Code)
	c.fn.Emit(bytecode.LOAD_LOCAL, idxSlot)
	c.fn.Emit(bytecode.LOAD_LOCAL, listSlot)
	nameIdx := c.mod.AddConstant(bytecode.BuiltinInfo{Name: "len", Arity: 1})
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
	c.pushLoop()
	if err := c.compileBlock(n.Body); err != nil {
		return err
	}
	continueStart := len(c.fn.Code)
	c.fn.Emit(bytecode.LOAD_LOCAL, idxSlot)
	oneIdx := c.mod.AddConstant(1)
	c.fn.Emit(bytecode.PUSH_INT, oneIdx)
	c.fn.Emit(bytecode.ADD_INT, 0)
	c.fn.Emit(bytecode.STORE_LOCAL, idxSlot)
	c.fn.Emit(bytecode.POP, 0)
	c.fn.Emit(bytecode.JUMP, loopStart)
	loopEnd := len(c.fn.Code)
	c.popLoop(loopEnd, continueStart)
	c.fn.Code[exitJump].Arg = loopEnd
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

func (c *Compiler) pushLoop() {
	c.loopStack = append(c.loopStack, loopFrame{})
}

func (c *Compiler) popLoop(loopEnd, continueTarget int) {
	if len(c.loopStack) == 0 {
		return
	}
	top := c.loopStack[len(c.loopStack)-1]
	for _, idx := range top.breakPatches {
		c.fn.Code[idx].Arg = loopEnd
	}
	for _, idx := range top.continuePatches {
		c.fn.Code[idx].Arg = continueTarget
	}
	c.loopStack = c.loopStack[:len(c.loopStack)-1]
}

func (c *Compiler) compileBreak() error {
	if len(c.loopStack) == 0 {
		return fmt.Errorf("compileBreak: break outside for/while")
	}
	idx := len(c.fn.Code)
	c.fn.Emit(bytecode.JUMP, 0)
	top := &c.loopStack[len(c.loopStack)-1]
	top.breakPatches = append(top.breakPatches, idx)
	return nil
}

func (c *Compiler) compileContinue() error {
	if len(c.loopStack) == 0 {
		return fmt.Errorf("compileContinue: continue outside for/while")
	}
	idx := len(c.fn.Code)
	c.fn.Emit(bytecode.JUMP, 0)
	top := &c.loopStack[len(c.loopStack)-1]
	top.continuePatches = append(top.continuePatches, idx)
	return nil
}

func isWildcardPattern(p ast.Expression) bool {
	v, ok := p.(*ast.VariableExpr)
	return ok && v.Name == "_"
}

func (c *Compiler) compilePatternMatch(scrutineeSlot int, pattern ast.Expression) error {
	if isWildcardPattern(pattern) {
		idx := c.mod.AddConstant(true)
		c.fn.Emit(bytecode.PUSH_BOOL, idx)
		return nil
	}
	scrType := valNil
	if scrutineeSlot < len(c.varTypes) {
		scrType = c.varTypes[scrutineeSlot]
	}
	c.fn.Emit(bytecode.LOAD_LOCAL, scrutineeSlot)
	patType, err := c.compileExpr(pattern)
	if err != nil {
		return err
	}
	opType := scrType
	if scrType != patType {
		switch {
		case scrType == valNil && patType != valNil:
			opType = patType
		case patType == valNil && scrType != valNil:
			opType = scrType
		default:
			return fmt.Errorf("compilePatternMatch: type mismatch %s vs %s", scrType, patType)
		}
	}
	op, err := pickBinaryOp("==", opType)
	if err != nil {
		return fmt.Errorf("compilePatternMatch: %w", err)
	}
	c.fn.Emit(op, 0)
	return nil
}

func (c *Compiler) compileMatch(n *ast.MatchStmt) error {
	c.pushScope()
	defer c.popScope()
	scrType, err := c.compileExpr(n.Expr)
	if err != nil {
		return err
	}
	slot := c.declareLocal("__match__", scrType)
	c.fn.Emit(bytecode.STORE_LOCAL, slot)
	c.fn.Emit(bytecode.POP, 0)

	endJumps := []int{}
	for _, arm := range n.Arms {
		if err := c.compilePatternMatch(slot, arm.Pattern); err != nil {
			return err
		}
		skipIdx := len(c.fn.Code)
		c.fn.Emit(bytecode.JUMP_IF_FALSE, 0)
		if err := c.compileBlock(arm.Body); err != nil {
			return err
		}
		endIdx := len(c.fn.Code)
		c.fn.Emit(bytecode.JUMP, 0)
		endJumps = append(endJumps, endIdx)
		c.fn.Code[skipIdx].Arg = len(c.fn.Code)
	}
	matchEnd := len(c.fn.Code)
	for _, idx := range endJumps {
		c.fn.Code[idx].Arg = matchEnd
	}
	return nil
}