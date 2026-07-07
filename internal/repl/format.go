package repl

import (
	"fmt"
)

// FormatValue renders a REPL result for display.
func FormatValue(v any) string {
	if v == nil {
		return "nil"
	}
	switch x := v.(type) {
	case string:
		return fmt.Sprintf("%q", x)
	default:
		return fmt.Sprintf("%v", x)
	}
}
