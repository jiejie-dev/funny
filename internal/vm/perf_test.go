// v2/internal/vm/perf_test.go
package vm_test

import (
	"os"
	"testing"

	"github.com/jiejie-dev/funny/v2/internal/cli"
)

func BenchmarkFibRecursive_VM(b *testing.B) {
	data, err := os.ReadFile("../../testdata/vm/fib.fn")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := cli.Run(data, "fib.fn"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFibRecursive_Interpreter(b *testing.B) {
	os.Setenv("FUNNY_INTERPRET", "1")
	defer os.Unsetenv("FUNNY_INTERPRET")
	data, err := os.ReadFile("../../testdata/vm/fib.fn")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := cli.Run(data, "fib.fn"); err != nil {
			b.Fatal(err)
		}
	}
}
