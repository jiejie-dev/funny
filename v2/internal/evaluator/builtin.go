// v2/internal/evaluator/builtin.go
package evaluator

type builtinFn struct {
	name string
	fn   func(e *Evaluator, args []any) (any, error)
}

var builtins = map[string]builtinFn{}
