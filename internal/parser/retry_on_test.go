package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStep_RetryOn(t *testing.T) {
	step := parseStepFrom(t, "plan \"p\":\n    step \"s\" -> tool with retry max=3 on=NetworkError,FatalError:\n        let x = 1\n")
	require.NotNil(t, step.Retry)
	require.Equal(t, 3, step.Retry.Max)
	require.Equal(t, []string{"NetworkError", "FatalError"}, step.Retry.On)
}

func TestParseStep_RetryOnRequiresName(t *testing.T) {
	_, err := New("plan \"p\":\n    step \"s\" -> tool with retry max=3 on=:\n        let x = 1\n", "test.fn").Parse()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "E1052")
}
