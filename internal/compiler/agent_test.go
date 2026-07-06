// v2/internal/compiler/agent_test.go
package compiler

import (
	"testing"

	"github.com/jiejie-dev/funny/v2/internal/bytecode"
	"github.com/stretchr/testify/assert"
)

// MetaBlock and PlanBlock are purely declarative (consumed by `funny describe`
// tooling), so the compiler must treat them as no-ops, mirroring the
// tree-walking evaluator's behavior.
func TestCompile_MetaBlock_NoOp(t *testing.T) {
	mod := compileExpr(t, `meta:
    name: "demo"
    version: "1.0"
let x = 1
`)
	fn := mod.Functions[0]
	assert.Equal(t, bytecode.HALT, fn.Code[len(fn.Code)-1].Op)
}

func TestCompile_PlanBlock_NoOp(t *testing.T) {
	mod := compileExpr(t, `plan "demo_plan":
    step "setup":
        let x = 10
    step "compute" -> tool:
        let r = x * 2
`)
	fn := mod.Functions[0]
	assert.Equal(t, bytecode.HALT, fn.Code[len(fn.Code)-1].Op)
}
