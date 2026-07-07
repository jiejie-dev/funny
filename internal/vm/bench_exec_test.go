package vm

import (
	"os"
	"testing"
	"time"

	"github.com/jiejie-dev/funny/v2/internal/ast"
	"github.com/jiejie-dev/funny/v2/internal/bytecode"
	"github.com/jiejie-dev/funny/v2/internal/compiler"
	"github.com/jiejie-dev/funny/v2/internal/evaluator"
	"github.com/jiejie-dev/funny/v2/internal/parser"
	"github.com/jiejie-dev/funny/v2/internal/types"
)

// precompiledFib returns a compiled fib(20) module and parsed program.
func precompiledFib(tb testing.TB) (*bytecode.Module, *ast.Program) {
	tb.Helper()
	data, err := os.ReadFile("../../testdata/vm/fib.fn")
	if err != nil {
		tb.Fatal(err)
	}
	p := parser.New(string(data), "fib.fn")
	prog, err := p.Parse()
	if err != nil {
		tb.Fatal(err)
	}
	env := types.NewEnv(nil)
	if err := types.Check(prog, env); err != nil {
		tb.Fatal(err)
	}
	mod, err := compiler.Compile(prog, "fib.fn")
	if err != nil {
		tb.Fatal(err)
	}
	return mod, prog
}

func BenchmarkFib_VM_ExecOnly(b *testing.B) {
	mod, _ := precompiledFib(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := New(mod)
		if _, err := m.Run(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFib_Interpreter_ExecOnly(b *testing.B) {
	_, prog := precompiledFib(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e := evaluator.New(nil)
		if err := e.Exec(prog); err != nil {
			b.Fatal(err)
		}
	}
}

func TestFib_SpeedupRatio(t *testing.T) {
	mod, prog := precompiledFib(t)
	const iters = 30
	vm := New(mod)
	startVM := time.Now()
	for i := 0; i < iters; i++ {
		if _, err := vm.Run(); err != nil {
			t.Fatal(err)
		}
	}
	vmDur := time.Since(startVM)

	startInt := time.Now()
	for i := 0; i < iters; i++ {
		e := evaluator.New(nil)
		if err := e.Exec(prog); err != nil {
			t.Fatal(err)
		}
	}
	intDur := time.Since(startInt)

	ratio := float64(intDur) / float64(vmDur)
	t.Logf("exec-only ratio: %.2fx (vm=%v, interp=%v)", ratio, vmDur/iters, intDur/iters)
	if ratio < 5.0 {
		t.Fatalf("exec-only speedup %.2fx is below 5× target", ratio)
	}
}
