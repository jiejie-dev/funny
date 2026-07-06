package formatter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormat_BranchCaseList(t *testing.T) {
	src := "plan \"demo\":\n" +
		"    step \"route\" -> branch:\n" +
		"        status == 200 => \"success\"\n" +
		"        _ => \"fail\"\n"
	out, err := Format([]byte(src), "t")
	require.NoError(t, err)
	assert.Equal(t, src, out)
}
