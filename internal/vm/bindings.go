package vm

import (
	"strings"

	"github.com/jiejie-dev/funny/v2/internal/bytecode"
)

// MainBindings returns named local bindings from the main frame after Run.
// Returns nil when no main frame is active.
func (v *VM) MainBindings() map[string]bytecode.Value {
	if len(v.frames) == 0 {
		return nil
	}
	frame := &v.frames[0]
	out := make(map[string]bytecode.Value)
	for i, val := range frame.locals {
		if i < len(frame.fn.LocalNames) {
			name := frame.fn.LocalNames[i]
			if name != "" && !strings.HasPrefix(name, "__") {
				out[name] = val
			}
		}
	}
	return out
}
