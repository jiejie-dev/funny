package strfmt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustFormat(t *testing.T, val any, spec string) string {
	t.Helper()
	s, err := Format(val, spec)
	require.NoError(t, err)
	return s
}

func TestFormat_Default(t *testing.T) {
	assert.Equal(t, "3", mustFormat(t, 3, ""))
	assert.Equal(t, "nil", mustFormat(t, nil, ""))
	assert.Equal(t, "true", mustFormat(t, true, ""))
	assert.Equal(t, "false", mustFormat(t, false, ""))
	assert.Equal(t, "hello", mustFormat(t, "hello", ""))
}

func TestFormat_FloatPrecision(t *testing.T) {
	assert.Equal(t, "3.14", mustFormat(t, 3.14159, ".2f"))
	assert.Equal(t, "3.000000", mustFormat(t, 3.0, "f"))
}

func TestFormat_ZeroPad(t *testing.T) {
	assert.Equal(t, "00042", mustFormat(t, 42, "05d"))
	assert.Equal(t, "-0042", mustFormat(t, -42, "05d"))
}

func TestFormat_Align(t *testing.T) {
	assert.Equal(t, "        hi", mustFormat(t, "hi", ">10"))
	assert.Equal(t, "hi        ", mustFormat(t, "hi", "<10"))
	assert.Equal(t, "    hi    ", mustFormat(t, "hi", "^10"))
}

func TestFormat_CustomFill(t *testing.T) {
	assert.Equal(t, "**hi", mustFormat(t, "hi", "*>4"))
	assert.Equal(t, "hi**", mustFormat(t, "hi", "*<4"))
}

func TestFormat_Hex(t *testing.T) {
	assert.Equal(t, "ff", mustFormat(t, 255, "x"))
	assert.Equal(t, "FF", mustFormat(t, 255, "X"))
}

func TestFormat_Octal(t *testing.T) {
	assert.Equal(t, "10", mustFormat(t, 8, "o"))
}

func TestFormat_Binary(t *testing.T) {
	assert.Equal(t, "101", mustFormat(t, 5, "b"))
}

func TestFormat_Percent(t *testing.T) {
	assert.Equal(t, "50.0%", mustFormat(t, 0.5, ".1%"))
}

func TestFormat_Sign(t *testing.T) {
	assert.Equal(t, "+3", mustFormat(t, 3, "+d"))
	assert.Equal(t, "-3", mustFormat(t, -3, "+d"))
}

func TestFormat_StringPrecisionTruncates(t *testing.T) {
	assert.Equal(t, "hel", mustFormat(t, "hello", ".3s"))
}

func TestFormat_IntCoercedFromFloat(t *testing.T) {
	assert.Equal(t, "3", mustFormat(t, 3.9, "d"))
}

func TestParseSpec_Invalid(t *testing.T) {
	_, err := ParseSpec("garbage!!")
	assert.Error(t, err)
}

func TestFormat_TypeMismatchErrors(t *testing.T) {
	_, err := Format("three", "d")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "'d'")
}

func TestFormat_NoSpecReturnsFastPath(t *testing.T) {
	assert.Equal(t, "[1 2 3]", mustFormat(t, []int{1, 2, 3}, ""))
}
