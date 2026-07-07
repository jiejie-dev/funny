package evaluator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiejie-dev/funny/v2/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluator_CancelWhileLoop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e := NewWithContext(NewScope(nil), ctx)
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	src := "while true:\n    let x = 1\n"
	prog, err := parser.New(src, "t.fn").Parse()
	require.NoError(t, err)
	_, _, err = e.ExecCell(prog)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCancelled))
}

func TestEvaluator_NoCancelByDefault(t *testing.T) {
	e := New(nil)
	prog, err := parser.New("let x = 1\n", "t.fn").Parse()
	require.NoError(t, err)
	require.NoError(t, e.Exec(prog))
	v, ok := e.Scope().Get("x")
	require.True(t, ok)
	assert.Equal(t, 1, v)
}
