package funny

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func RunSingle(data interface{}) (*Funny, Value) {
	i := NewFunny()
	var d []byte
	switch v := data.(type) {
	case string:
		d = []byte(v)
	}
	r, _ := i.Run(d)
	return i, Value(r)
}

func TestFunny_Assign(t *testing.T) {
	i := NewFunny()
	i.Assign("a", Value(1))
	flag := false
	var val interface{}
	for _, scope := range i.Vars {
		for k, v := range scope {
			if k == "a" {
				flag = true
				val = v
			}
		}
	}
	if !flag {
		t.Error("assign error key not in scope")
	} else {
		if val != 1 {
			t.Error("assign error value not equal 1")
		}
	}
	scope := Scope{}
	i.PushScope(scope)
	i.Assign("b", Value(2))
	v := i.Lookup("b")
	if v != 2 {
		t.Errorf("val not eq 2 %s", v)
	}
	i.Assign("a", Value(3))
	a := i.Lookup("a")
	if a != 3 {
		t.Errorf("a not eq 3 %s", a)
	}
	i.PopScope()
	v = i.LookupDefault("b", nil)
	if v != nil {
		t.Error("pop scope error")
	}
}

func TestFunny_Lookup(t *testing.T) {
	i := NewFunny()
	i.Assign("a", Value(1))
	val := i.Lookup("a")
	if val != 1 {
		t.Error("lookup error")
	}
}

func TestFunny_EvalFunctionCall(t *testing.T) {
	i := NewFunny()
	i.Run("echo(1)")
}

func TestFunny_EvalFunctionCall2(t *testing.T) {
	i := NewFunny()
	i.Run("echo2(b){echo(b)} \n echo2(1)")
}

// func TestFunny_EvalFieldFunctionCall(t *testing.T) {
// 	i := NewFunny()
// 	parser := NewParser([]byte(`
//         baseUrl = 'test'
//         token = 'testtoken'
//         f() {
//             return httpreq('GET', baseUrl + 'api/appraisal/admin/terms', {

//             }, {
//               Authorization = 'Bearer ' + token
//             }, debug)
//           }
//         r = f()
//         echoln(r)
//     `), "")
// 	i.Run(Program{
// 		parser.Parse(),
// 	})
// }

func TestFunny_EvalPlus(t *testing.T) {
	i := NewFunny()
	i.Run("  a = 1 + 1")
	a := i.Lookup("a")
	if a != 2 {
		t.Error("eval plus error")
	}
}

func TestFunny_Run(t *testing.T) {
	data := `
a = 1
b = 2
c = a + b

echo(c)

p(a, b){
    return a + b
}

d = p(a,b)

return d - 1`

	_, r := RunSingle(data)
	if r != 2 {
		t.Error("RunSingle funny.fun must return 2")
	}
}

func TestFunny_Return(t *testing.T) {
	data := `
testReturn(t){
    if t < 1 {
        return t
    }
    return testReturn(t-1)
}

t = testReturn(10)`
	_, r := RunSingle(data)
	ty := Typing(r)
	t.Log(ty)
	t.Log(r)
}

func TestFunny_Fib(t *testing.T) {
	data := `
fib(n) {
    echoln('n: ', n)
    if n < 2 {
      return n
    } else {
      return fib(n - 2) + fib(n - 1)
    }
}

return fib(5)`

	_, r := RunSingle(data)
	ty := Typing(r)
	t.Log(ty)
	t.Log(r)
}

func TestFunny_EvalBlock(t *testing.T) {
	data := `
a = 2
b = 1
if a > b {
return a
} else {
return b
}`

	_, r := RunSingle(data)
	if r != 2 {
		t.Error(fmt.Sprintf("RunSingle funny.fun must return 2 but got %s", r))
	}
}

func TestFuny_EvalInTrue(t *testing.T) {
	data := `a = 2 in [2]`
	i := NewFunny()
	i.Run(data)
	aInArray := i.Lookup("a")
	assert.True(t, aInArray.(bool))
}

func TestFuny_EvalInFalse(t *testing.T) {
	data := `a = 2 in [1]`
	i := NewFunny()
	i.Run(data)
	aInArray := i.Lookup("a")
	assert.True(t, !aInArray.(bool))
}

func TestFuny_EvalNotIn(t *testing.T) {
	data := `a = 2 not in [2]`
	i := NewFunny()
	i.Run(data)
	aInArray := i.Lookup("a")
	fmt.Println(aInArray)
	assert.True(t, !aInArray.(bool))
}

func TestFunnyIfStatementWithElseIf(t *testing.T) {
	data := `
a = 1
if a > 3 {
echoln(true)
} else if a == 1 {
b = 2
echoln('else if')
} else {
b = 3
echoln('else')
}
`
	i := NewFunny()
	i.Run(data)
	aInArray := i.Lookup("b")
	assert.Equal(t, 2, aInArray.(int))
}

func TestFunnyFieldAccessString(t *testing.T) {
	data := `
m = {
  a = 1
}
b = 'a'
c = m['a']
`
	i := NewFunny()
	i.Run(data)
	aInArray := i.Lookup("c")
	assert.Equal(t, 1, aInArray.(int))
}

func TestFunnyFieldAccessNamed(t *testing.T) {
	data := `
m = {
  a = 1
}
b = 'a'
c = m[b]
`
	i := NewFunny()
	i.Run(data)
	aInArray := i.Lookup("c")
	assert.Equal(t, 1, aInArray.(int))
}

func TestBuiltinFunctionRegexMatch(t *testing.T) {
	data := `
c = regexMatch('a', 'abcde')
`
	i := NewFunny()
	i.Run(data)
	matched := i.Lookup("c")
	assert.Equal(t, true, matched.(bool))
}

func TestBuiltinFunctionRegexMapMatch(t *testing.T) {
	m := map[string]Value{
		"a": "c",
	}
	data := `
c = regexMapMatch(m, 'abcedfg')
`
	i := NewFunny()
	i.Assign("m", m)
	i.Run(data)
	aInArray := i.Lookup("c")
	assert.Equal(t, true, aInArray.(bool))
}

func TestBuiltinFunctionRegexMapValue(t *testing.T) {
	m := map[string]Value{
		"a": "c",
	}
	data := `
c = regexMapValue(m, 'abcedfg')
`
	i := NewFunny()
	i.Assign("m", m)
	i.Run(data)
	aInArray := i.Lookup("c")
	assert.Equal(t, "c", aInArray.(string))
}

func TestBuiltinFunctionSh(t *testing.T) {
	data := `
sh('ls')
`
	i := NewFunny()
	i.Run(data)
}

func TestFunnySubExpression(t *testing.T) {
	data := `
a = (1 + 2) + 3
`
	i := NewFunny()
	i.Run(data)
	a := i.Lookup("a").(int)
	assert.Equal(t, 6, a)
}
