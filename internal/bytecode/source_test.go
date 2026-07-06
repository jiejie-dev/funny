package bytecode

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceLoc_Display(t *testing.T) {
	assert.Equal(t, "<unknown>", SourceLoc{}.Display())
	assert.Equal(t, "a.fn:3:5", SourceLoc{File: "a.fn", Line: 2, Col: 4}.Display())
}

func TestFunction_EmitAt_ParallelLocations(t *testing.T) {
	fn := &Function{Name: "main"}
	fn.EmitAt(PUSH_INT, 0, SourceLoc{File: "t.fn", Line: 1, Col: 0})
	fn.Emit(PUSH_INT, 1)
	require.Len(t, fn.Code, 2)
	require.Len(t, fn.Locations, 2)
	assert.Equal(t, "t.fn:2:1", fn.Locations[0].Display())
	assert.True(t, fn.Locations[1].IsZero())
}

func TestModule_SourceMapJSON(t *testing.T) {
	fn := &Function{Name: "main", NumLocals: 1, LocalNames: []string{"x"}}
	fn.EmitAt(PUSH_INT, 0, SourceLoc{File: "t.fn", Line: 0, Col: 2})
	fn.EmitAt(STORE_LOCAL, 0, SourceLoc{File: "t.fn", Line: 0, Col: 6})
	mod := NewModule("t.fn")
	mod.AddConstant(42)
	mod.AddFunction(fn)
	data, err := mod.SourceMapJSON()
	require.NoError(t, err)
	var sm ModuleSourceMap
	require.NoError(t, json.Unmarshal(data, &sm))
	require.Len(t, sm.Functions, 1)
	require.Len(t, sm.Functions[0].Instructions, 2)
	assert.Equal(t, 1, sm.Functions[0].Instructions[0].Line)
	assert.Equal(t, "PUSH_INT", sm.Functions[0].Instructions[0].Op)
	assert.Equal(t, []string{"x"}, sm.Functions[0].LocalNames)
}
