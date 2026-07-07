package types

import "github.com/jiejie-dev/funny/v2/internal/ast"

func checkTestBlock(n *ast.TestBlock, env *Env) error {
	if n.Body == nil {
		return nil
	}
	return Check(n.Body.ToProgram(), env)
}
