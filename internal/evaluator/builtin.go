// v2/internal/evaluator/builtin.go
package evaluator

import (
	"fmt"

	"github.com/jiejie-dev/funny/v2/internal/stdlib"
)

func callBuiltin(name string, args []any) (any, error) {
	if !stdlib.Names[name] {
		return nil, fmt.Errorf("unknown builtin %q", name)
	}
	return stdlib.Call(name, args)
}

func isBuiltin(name string) bool {
	return stdlib.Names[name]
}
