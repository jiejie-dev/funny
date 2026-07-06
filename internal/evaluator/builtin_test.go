package evaluator

import (
	"testing"

	"github.com/jiejie-dev/funny/v2/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runSrc(t *testing.T, src string) *Evaluator {
	t.Helper()
	p := parser.New(src, "")
	prog, err := p.Parse()
	require.NoError(t, err)
	e := New(nil)
	require.NoError(t, e.Exec(prog))
	return e
}

func TestBuiltin_Print(t *testing.T) {
	runSrc(t, `print("hello")`)
}

func TestBuiltin_Len(t *testing.T) {
	e := runSrc(t, `let n = len("hello")`)
	v, _ := e.scope.Get("n")
	assert.Equal(t, 5, v)
}

func TestBuiltin_LenList(t *testing.T) {
	e := runSrc(t, `let n = len([1, 2, 3])`)
	v, _ := e.scope.Get("n")
	assert.Equal(t, 3, v)
}

func TestBuiltin_ToStr(t *testing.T) {
	e := runSrc(t, `let s = to_str(42)`)
	v, _ := e.scope.Get("s")
	assert.Equal(t, "42", v)
}

func TestBuiltin_ToInt(t *testing.T) {
	e := runSrc(t, `let n = to_int("42")`)
	v, _ := e.scope.Get("n")
	assert.Equal(t, 42, v)
}

func TestBuiltin_TypeOf(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{`let t = type_of(42)`, "int"},
		{`let t = type_of(3.14)`, "float"},
		{`let t = type_of("hi")`, "str"},
		{`let t = type_of(true)`, "bool"},
		{`let t = type_of(nil)`, "nil"},
		{`let t = type_of([1, 2])`, "list"},
	}
	for _, c := range cases {
		e := runSrc(t, c.src)
		v, _ := e.scope.Get("t")
		assert.Equal(t, c.want, v, "src=%s", c.src)
	}
}

func TestBuiltin_AppendAndSqrt(t *testing.T) {
	e := runSrc(t, `let xs = append([1], 2)
let r = sqrt(16)
`)
	v, _ := e.scope.Get("xs")
	assert.Equal(t, []any{1, 2}, v)
	r, _ := e.scope.Get("r")
	assert.Equal(t, 4.0, r)
}

func TestBuiltin_RegexAndB64(t *testing.T) {
	e := runSrc(t, `let ok = regex_match("^h", "hello")
let enc = b64_encode("hi")
`)
	ok, _ := e.scope.Get("ok")
	assert.Equal(t, true, ok)
	enc, _ := e.scope.Get("enc")
	assert.Equal(t, "aGk=", enc)
}
