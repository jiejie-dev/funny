package docgen

import (
	"testing"

	"github.com/jiejie-dev/funny/v2/internal/parser"
	"github.com/jiejie-dev/funny/v2/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectSymbolsAndIndex(t *testing.T) {
	src := `## Greets someone
fn greet(name: str) -> str:
    return "hi"

## One item
struct Item:
    id: int
`
	prog, err := parser.New(src, "test.fn").Parse()
	require.NoError(t, err)
	env := types.NewEnv(nil)
	require.NoError(t, types.Check(prog, env))

	symbols := CollectSymbols(prog, env)
	require.Len(t, symbols, 2)
	assert.Equal(t, "greet", symbols[0].Name)
	assert.Equal(t, "Greets someone", symbols[0].Summary)
	assert.Equal(t, "Item", symbols[1].Name)
	assert.Equal(t, "One item", symbols[1].Summary)

	idx := SymbolIndex(prog, env)
	assert.Equal(t, "Greets someone", idx["greet"].Summary)
	assert.Equal(t, "One item", idx["Item"].Summary)
}
