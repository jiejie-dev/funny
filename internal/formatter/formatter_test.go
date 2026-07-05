package formatter

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormat_LetAndExprStmt(t *testing.T) {
	out, err := Format([]byte("let   x=1\nprintln(x)\n"), "t")
	require.NoError(t, err)
	assert.Equal(t, "let x = 1\nprintln(x)\n", out)
}

func TestFormat_LetWithTypeAnnotation(t *testing.T) {
	out, err := Format([]byte("let x:int=1\n"), "t")
	require.NoError(t, err)
	assert.Equal(t, "let x: int = 1\n", out)
}

func TestFormat_TrailingComment(t *testing.T) {
	out, err := Format([]byte("let x = 1  # note\n"), "t")
	require.NoError(t, err)
	assert.Equal(t, "let x = 1  # note\n", out)
}

func TestFormat_StandaloneComment(t *testing.T) {
	out, err := Format([]byte("let x = 1\n# note\nlet y = 2\n"), "t")
	require.NoError(t, err)
	assert.Equal(t, "let x = 1\n# note\nlet y = 2\n", out)
}

func TestFormat_DocComment(t *testing.T) {
	out, err := Format([]byte("## doc note\nlet x = 1\n"), "t")
	require.NoError(t, err)
	assert.Equal(t, "## doc note\nlet x = 1\n", out)
}

func TestFormat_StructLiteralFieldOrderDeterministic(t *testing.T) {
	out1, err := Format([]byte("let p = Point(y: 2, x: 1)\n"), "t")
	require.NoError(t, err)
	out2, err := Format([]byte(out1), "t")
	require.NoError(t, err)
	assert.Equal(t, out1, out2)
	assert.Equal(t, "let p = Point(x: 1, y: 2)\n", out1)
}

func TestFormat_FStringRoundtrip(t *testing.T) {
	out, err := Format([]byte(`println(f"hi {name}, total {price:.2f}")`+"\n"), "t")
	require.NoError(t, err)
	assert.Equal(t, `println(f"hi {name}, total {price:.2f}")`+"\n", out)
}

func TestFormat_FStringBraceEscape(t *testing.T) {
	out, err := Format([]byte(`f"{{literal}}"`+"\n"), "t")
	require.NoError(t, err)
	assert.Equal(t, `f"{{literal}}"`+"\n", out)
}

func TestFormat_AssignStmt(t *testing.T) {
	out, err := Format([]byte("x=x+1\n"), "t")
	require.NoError(t, err)
	assert.Equal(t, "x = x + 1\n", out)
}

func TestFormat_ReturnNoValue(t *testing.T) {
	out, err := Format([]byte("fn f():\n    return\n"), "t")
	require.NoError(t, err)
	assert.Equal(t, "fn f():\n    return\n", out)
}

func TestFormat_ListAndBinary(t *testing.T) {
	out, err := Format([]byte("let x=[1,2,3]\nlet y=1+2*3\n"), "t")
	require.NoError(t, err)
	assert.Equal(t, "let x = [1, 2, 3]\nlet y = 1 + 2 * 3\n", out)
}

func TestFormat_MapLiteral(t *testing.T) {
	out, err := Format([]byte(`let m: map[str, int] = {"a": 1, "b": 2}`+"\n"), "t")
	require.NoError(t, err)
	assert.Equal(t, "let m: map[str, int] = {\"a\": 1, \"b\": 2}\n", out)
}

func TestFormat_MapLiteral_MultiLineInputCollapsesToOneLine(t *testing.T) {
	src := "let m = {\n    \"a\": 1,\n    \"b\": 2,\n}\n"
	out, err := Format([]byte(src), "t")
	require.NoError(t, err)
	assert.Equal(t, "let m = {\"a\": 1, \"b\": 2}\n", out)
}

func TestFormat_IdempotentOnItself(t *testing.T) {
	src := "let x = 1\nprintln(x)\n"
	out1, err := Format([]byte(src), "t")
	require.NoError(t, err)
	out2, err := Format([]byte(out1), "t")
	require.NoError(t, err)
	assert.Equal(t, out1, out2)
}

func TestFormat_IfElifElse(t *testing.T) {
	src := "if x>0:\n    println(1)\nelif x<0:\n    println(2)\nelse:\n    println(3)\n"
	out, err := Format([]byte(src), "t")
	require.NoError(t, err)
	assert.Equal(t, "if x > 0:\n    println(1)\nelif x < 0:\n    println(2)\nelse:\n    println(3)\n", out)
}

func TestFormat_For(t *testing.T) {
	out, err := Format([]byte("for i in items:\n    println(i)\n"), "t")
	require.NoError(t, err)
	assert.Equal(t, "for i in items:\n    println(i)\n", out)
}

func TestFormat_While(t *testing.T) {
	out, err := Format([]byte("while x<10:\n    x=x+1\n"), "t")
	require.NoError(t, err)
	assert.Equal(t, "while x < 10:\n    x = x + 1\n", out)
}

func TestFormat_Match(t *testing.T) {
	src := "match x:\n    1 =>\n        println(\"one\")\n    2 =>\n        println(\"two\")\n"
	out, err := Format([]byte(src), "t")
	require.NoError(t, err)
	assert.Equal(t, src, out)
}

func TestFormat_FnDeclWithParamsAndRetType(t *testing.T) {
	out, err := Format([]byte("fn add(a:int,b:int)->int:\n    return a+b\n"), "t")
	require.NoError(t, err)
	assert.Equal(t, "fn add(a: int, b: int) -> int:\n    return a + b\n", out)
}

func TestFormat_PubFnDecl(t *testing.T) {
	out, err := Format([]byte("pub fn f():\n    return 1\n"), "t")
	require.NoError(t, err)
	assert.Equal(t, "pub fn f():\n    return 1\n", out)
}

func TestFormat_StructDecl(t *testing.T) {
	out, err := Format([]byte("pub struct Point:\n    x:int\n    y:int\n"), "t")
	require.NoError(t, err)
	assert.Equal(t, "pub struct Point:\n    x: int\n    y: int\n", out)
}

func TestFormat_MetaBlock(t *testing.T) {
	out, err := Format([]byte("meta:\n    name=\"demo\"\n    version=\"1.0\"\n"), "t")
	require.NoError(t, err)
	assert.Equal(t, "meta:\n    name = \"demo\"\n    version = \"1.0\"\n", out)
}

func TestFormat_PlanAndStep(t *testing.T) {
	out, err := Format([]byte("plan \"demo\":\n    step \"one\":\n        println(1)\n"), "t")
	require.NoError(t, err)
	assert.Equal(t, "plan \"demo\":\n    step \"one\":\n        println(1)\n", out)
}

func TestFormat_StepWithKindAndRetry(t *testing.T) {
	src := "plan \"demo\":\n    step \"one\" -> guard with retry max=3:\n        println(1)\n"
	out, err := Format([]byte(src), "t")
	require.NoError(t, err)
	assert.Equal(t, src, out)
}

// Regression: n.Retry.Backoff and n.Timeout used to be silently dropped by
// the formatter, and an explicit `-> tool` kind co-occurring with `with`
// options was lost too - turning a step's retry/backoff/timeout semantics
// into different behavior on round-trip through `funny fmt`.
func TestFormat_StepWithBackoffAndTimeout(t *testing.T) {
	// "tool" is the default kind, so it isn't re-emitted explicitly; only
	// the retry/backoff/timeout options are checked for round-tripping.
	src := "plan \"demo\":\n    step \"one\" with retry max=2 backoff=exp timeout=\"5s\":\n        println(1)\n"
	out, err := Format([]byte(src), "t")
	require.NoError(t, err)
	assert.Equal(t, src, out)
}

func TestFormat_StepWithTimeoutOnly(t *testing.T) {
	src := "plan \"demo\":\n    step \"one\" with timeout=\"2s\":\n        println(1)\n"
	out, err := Format([]byte(src), "t")
	require.NoError(t, err)
	assert.Equal(t, src, out)
}

// Regression: whole-number float literals (e.g. 500.0) used to be
// reformatted as "500", which re-parses as an int literal, not a float -
// silently changing a function's return/comparison type on round-trip.
func TestFormat_WholeNumberFloatLiteral(t *testing.T) {
	src := "fn f() -> float:\n    return 500.0\n"
	out, err := Format([]byte(src), "t")
	require.NoError(t, err)
	assert.Equal(t, src, out)
}

// Regression: `return nil` used to be reformatted as the syntactically
// invalid `return <nil>`.
func TestFormat_NilLiteral(t *testing.T) {
	src := "fn f() -> str:\n    return nil\n"
	out, err := Format([]byte(src), "t")
	require.NoError(t, err)
	assert.Equal(t, src, out)
}

func TestFormat_NestedBlocksIdempotent(t *testing.T) {
	src := "fn f(x:int)->int:\n    if x>0:\n        for i in x:\n            println(i)\n    return x\n"
	out1, err := Format([]byte(src), "t")
	require.NoError(t, err)
	out2, err := Format([]byte(out1), "t")
	require.NoError(t, err)
	assert.Equal(t, out1, out2)
}

// TestFormat_IdempotentOnAllTestdata guards against a formatter that
// re-formats its own output differently, on every real fixture in the repo
// that currently parses successfully.
func TestFormat_IdempotentOnAllTestdata(t *testing.T) {
	var files []string
	for _, root := range []string{"../../testdata", "../../docs"} {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			if ext := filepath.Ext(path); ext == ".fn" || ext == ".funny" {
				files = append(files, path)
			}
			return nil
		})
		require.NoError(t, err)
	}
	require.NotEmpty(t, files)
	for _, f := range files {
		f := f
		t.Run(f, func(t *testing.T) {
			src, err := os.ReadFile(f)
			require.NoError(t, err)
			out1, err := Format(src, f)
			if err != nil {
				t.Skipf("skipping non-parseable fixture: %v", err)
			}
			out2, err := Format([]byte(out1), f)
			require.NoError(t, err)
			assert.Equal(t, out1, out2, "Format(Format(src)) must equal Format(src)")
		})
	}
}

func TestFormat_ParseErrorPropagates(t *testing.T) {
	_, err := Format([]byte("let = 5\n"), "t")
	assert.Error(t, err)
}
