package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceMap_ContainsLocations(t *testing.T) {
	src := `let x = 42
println(x)
`
	out, err := SourceMap([]byte(src), "t.fn")
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, `"module": "t.fn"`)
	assert.Contains(t, s, `"localNames"`)
	assert.Contains(t, s, `"line":`)
}

func TestDisasm_IncludesSourceComments(t *testing.T) {
	src := `let x = 1
`
	out, err := Disasm([]byte(src), "t.fn")
	require.NoError(t, err)
	assert.Contains(t, out, "; t.fn:")
}

func TestDebug_StepAndQuit(t *testing.T) {
	src := `let x = 10
let y = 20
`
	in := strings.NewReader("step\nstep\nquit\n")
	var buf bytes.Buffer
	err := Debug([]byte(src), "t.fn", DebugOptions{}, in, &buf)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "(dbg)")
	assert.Contains(t, out, "t.fn:")
}

func TestDebug_BreakpointFlag(t *testing.T) {
	src := `let x = 1
println(x)
`
	in := strings.NewReader("continue\nquit\n")
	var buf bytes.Buffer
	err := Debug([]byte(src), "t.fn", DebugOptions{Breakpoints: []string{"2"}}, in, &buf)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "t.fn:2:")
}

func TestParseBreakpoint(t *testing.T) {
	f, l, err := parseBreakpoint("12", "main.fn")
	require.NoError(t, err)
	assert.Equal(t, "main.fn", f)
	assert.Equal(t, 12, l)

	f, l, err = parseBreakpoint("other.fn:5", "main.fn")
	require.NoError(t, err)
	assert.Equal(t, "other.fn", f)
	assert.Equal(t, 5, l)
}
