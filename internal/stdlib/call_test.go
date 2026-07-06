package stdlib

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCall_Sqrt(t *testing.T) {
	v, err := Call("sqrt", []any{9})
	require.NoError(t, err)
	assert.Equal(t, 3.0, v)
}

func TestCall_Append(t *testing.T) {
	v, err := Call("append", []any{[]any{1, 2}, 3})
	require.NoError(t, err)
	assert.Equal(t, []any{1, 2, 3}, v)
}

func TestCall_ToJSON_ParseJSON(t *testing.T) {
	s, err := Call("to_json", []any{map[string]any{"a": 1}})
	require.NoError(t, err)
	v, err := Call("parse_json", []any{s})
	require.NoError(t, err)
	m, ok := v.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(1), m["a"])
}

func TestCall_RegexMatch(t *testing.T) {
	v, err := Call("regex_match", []any{"^a", "abc"})
	require.NoError(t, err)
	assert.Equal(t, true, v)
}

func TestCall_B64EncodeDecode(t *testing.T) {
	enc, err := Call("b64_encode", []any{"hi"})
	require.NoError(t, err)
	dec, err := Call("b64_decode", []any{enc})
	require.NoError(t, err)
	m := dec.(map[string]any)
	assert.Equal(t, "ok", m["tag"])
	assert.Equal(t, "hi", m["val"])
}

func TestNames_MatchesTypeCheckerAllowlist(t *testing.T) {
	expected := []string{
		"print", "println", "len", "append", "to_str", "to_int", "to_float", "type_of",
		"ok", "err", "to_json", "parse_json", "now", "time_format",
		"sqrt", "pow", "abs", "str_upper", "str_lower", "str_contains", "str_split",
		"regex_match", "regex_replace", "env_get", "file_read", "file_exists", "http_get",
		"md5", "sha256", "b64_encode", "b64_decode", "jwt_encode", "jwt_decode", "sql_open",
	}
	for _, name := range expected {
		assert.True(t, Names[name], "missing stdlib builtin %q", name)
	}
}
