// v2/internal/vm/bench_test.go
package vm

import (
	"os"
	"testing"

	"github.com/jiejie-dev/funny/internal/compiler"
	"github.com/jiejie-dev/funny/internal/evaluator"
	"github.com/jiejie-dev/funny/internal/parser"
	"github.com/jiejie-dev/funny/internal/types"
)

// runSource parses, type-checks, and executes src. If FUNNY_INTERPRET is set
// the tree-walking evaluator is used; otherwise the bytecode VM is used.
// This mirrors cli.Run but is inlined to avoid an import cycle (cli -> vm).
func runSource(src []byte, file string) error {
	p := parser.New(string(src), file)
	prog, err := p.Parse()
	if err != nil {
		return err
	}
	env := types.NewEnv(nil)
	if err := types.Check(prog, env); err != nil {
		return err
	}
	if os.Getenv("FUNNY_INTERPRET") != "" {
		e := evaluator.New(nil)
		return e.Exec(prog)
	}
	mod, err := compiler.Compile(prog, file)
	if err != nil {
		return err
	}
	m := New(mod)
	_, err = m.Run()
	return err
}

func BenchmarkFib_VM(b *testing.B) {
	data, err := os.ReadFile("../../testdata/vm/fib.fn")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := runSource(data, "fib.fn"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFib_Interpreter(b *testing.B) {
	os.Setenv("FUNNY_INTERPRET", "1")
	defer os.Unsetenv("FUNNY_INTERPRET")
	data, err := os.ReadFile("../../testdata/vm/fib.fn")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := runSource(data, "fib.fn"); err != nil {
			b.Fatal(err)
		}
	}
}
