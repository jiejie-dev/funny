package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStepKind_String(t *testing.T) {
	cases := []struct {
		k    StepKind
		want string
	}{
		{StepTool, "tool"},
		{StepGuard, "guard"},
		{StepTransform, "transform"},
		{StepParallel, "parallel"},
		{StepBranch, "branch"},
		{StepDelay, "delay"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, c.k.String())
	}
}
