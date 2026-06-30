// v2/internal/evaluator/builtin.go
package evaluator

import (
	"fmt"
	"strconv"
)

type builtinFn struct {
	name string
	fn   func(e *Evaluator, args []any) (any, error)
}

var builtins = map[string]builtinFn{
	"print": {
		name: "print",
		fn: func(e *Evaluator, args []any) (any, error) {
			fmt.Print(args...)
			return nil, nil
		},
	},
	"println": {
		name: "println",
		fn: func(e *Evaluator, args []any) (any, error) {
			fmt.Println(args...)
			return nil, nil
		},
	},
	"len": {
		name: "len",
		fn: func(e *Evaluator, args []any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("len() takes exactly 1 argument")
			}
			switch v := args[0].(type) {
			case string:
				return len(v), nil
			case []any:
				return len(v), nil
			}
			return nil, fmt.Errorf("len() not supported for type %T", args[0])
		},
	},
	"to_str": {
		name: "to_str",
		fn: func(e *Evaluator, args []any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("to_str() takes exactly 1 argument")
			}
			return fmt.Sprintf("%v", args[0]), nil
		},
	},
	"to_int": {
		name: "to_int",
		fn: func(e *Evaluator, args []any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("to_int() takes exactly 1 argument")
			}
			switch v := args[0].(type) {
			case int:
				return v, nil
			case float64:
				return int(v), nil
			case string:
				n, err := strconv.Atoi(v)
				if err != nil {
					return nil, fmt.Errorf("to_int() cannot parse %q", v)
				}
				return n, nil
			}
			return nil, fmt.Errorf("to_int() not supported for type %T", args[0])
		},
	},
	"type_of": {
		name: "type_of",
		fn: func(e *Evaluator, args []any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("type_of() takes exactly 1 argument")
			}
			switch args[0].(type) {
			case nil:
				return "nil", nil
			case bool:
				return "bool", nil
			case int:
				return "int", nil
			case float64:
				return "float", nil
			case string:
				return "str", nil
			case []any:
				return "list", nil
			case map[string]any:
				return "map", nil
			}
			return "unknown", nil
		},
	},
}
