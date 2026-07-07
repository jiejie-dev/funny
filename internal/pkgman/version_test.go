package pkgman

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVersion(t *testing.T) {
	v, err := ParseVersion("1.2.3")
	require.NoError(t, err)
	assert.Equal(t, Version{1, 2, 3}, v)

	v, err = ParseVersion("v2.0.0")
	require.NoError(t, err)
	assert.Equal(t, Version{2, 0, 0}, v)
}

func TestSatisfiesConstraint(t *testing.T) {
	require.NoError(t, SatisfiesConstraint("", "1.0.0"))
	require.NoError(t, SatisfiesConstraint("*", "9.9.9"))
	require.NoError(t, SatisfiesConstraint("1.2.3", "1.2.3"))
	require.Error(t, SatisfiesConstraint("1.2.3", "1.2.4"))

	require.NoError(t, SatisfiesConstraint(">=1.0.0", "1.5.0"))
	require.Error(t, SatisfiesConstraint(">=2.0.0", "1.9.9"))

	require.NoError(t, SatisfiesConstraint("^1.2.0", "1.3.0"))
	require.Error(t, SatisfiesConstraint("^1.2.0", "2.0.0"))
}

func TestResolvedVersion(t *testing.T) {
	assert.Equal(t, "1.2.0", ResolvedVersion("git+https://github.com/org/repo@v1.2.0"))
	assert.Equal(t, "0.0.0", ResolvedVersion("path:vendor/math.fn"))
}
