// v2/internal/vm/builtins.go
package vm

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/jerloo/funny/v2/internal/bytecode"
)

// convertJSON converts a generic Go value (from json.Unmarshal) into a funny
// Value, using []any and map[string]any instead of []interface{} and
// map[string]interface{}.
func convertJSON(x any) bytecode.Value {
	switch v := x.(type) {
	case nil:
		return nil
	case bool:
		return v
	case float64:
		return v
	case string:
		return v
	case []any:
		out := make([]bytecode.Value, len(v))
		for i, e := range v {
			out[i] = convertJSON(e)
		}
		return out
	case map[string]any:
		out := make(map[string]bytecode.Value, len(v))
		for k, e := range v {
			out[k] = convertJSON(e)
		}
		return out
	default:
		return fmt.Sprintf("%v", v)
	}
}

// execCallBuiltin handles CALL_BUILTIN nameIdx.
// The constant pool at nameIdx must be a struct{ Name string; Arity int }
// identifying the builtin and its argument count. Pops Arity args from stack
// (in source order, bottom-of-stack first) and consumes them.
func (v *VM) execCallBuiltin(nameIdx int) error {
	info, ok := v.mod.Constants[nameIdx].(bytecode.BuiltinInfo)
	if !ok {
		return fmt.Errorf("vm: CALL_BUILTIN name is not a BuiltinInfo")
	}
	name := info.Name
	arity := info.Arity
	if len(v.stack) < arity {
		return fmt.Errorf("vm: %s() requires %d arguments", name, arity)
	}
	start := len(v.stack) - arity
	switch name {
	case "print":
		for i := start; i < len(v.stack); i++ {
			if i > start {
				fmt.Print(" ")
			}
			fmt.Print(v.stack[i])
		}
		v.stack = v.stack[:start]
	case "println":
		for i := start; i < len(v.stack); i++ {
			if i > start {
				fmt.Print(" ")
			}
			fmt.Print(v.stack[i])
		}
		fmt.Println()
		v.stack = v.stack[:start]
	case "len":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: len() requires 1 argument")
		}
		x := v.stack[len(v.stack)-1]
		v.stack = v.stack[:len(v.stack)-1]
		switch val := x.(type) {
		case string:
			v.stack = append(v.stack, len(val))
		case []bytecode.Value:
			v.stack = append(v.stack, len(val))
		default:
			v.stack = append(v.stack, reflect.ValueOf(val).Len())
		}
	case "to_str":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: to_str() requires 1 argument")
		}
		v.stack[len(v.stack)-1] = fmt.Sprintf("%v", v.stack[len(v.stack)-1])
	case "to_int":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: to_int() requires 1 argument")
		}
		switch x := v.stack[len(v.stack)-1].(type) {
		case int:
			_ = x
		case float64:
			v.stack[len(v.stack)-1] = int(x)
		case string:
			var n int
			for _, c := range x {
				if c >= '0' && c <= '9' {
					n = n*10 + int(c-'0')
				}
			}
			v.stack[len(v.stack)-1] = n
		}
	case "type_of":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: type_of() requires 1 argument")
		}
		switch v.stack[len(v.stack)-1].(type) {
		case nil:
			v.stack[len(v.stack)-1] = "nil"
		case bool:
			v.stack[len(v.stack)-1] = "bool"
		case int:
			v.stack[len(v.stack)-1] = "int"
		case float64:
			v.stack[len(v.stack)-1] = "float"
		case string:
			v.stack[len(v.stack)-1] = "str"
		case []bytecode.Value:
			v.stack[len(v.stack)-1] = "list"
		case map[string]bytecode.Value:
			v.stack[len(v.stack)-1] = "map"
		default:
			v.stack[len(v.stack)-1] = "unknown"
		}
case "ok":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: ok() requires 1 argument")
		}
		val := v.stack[len(v.stack)-1]
		v.stack = v.stack[:len(v.stack)-1]
		v.stack = append(v.stack, makeResult("ok", val))
	case "err":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: err() requires 1 argument")
		}
		val := v.stack[len(v.stack)-1]
		v.stack = v.stack[:len(v.stack)-1]
		v.stack = append(v.stack, makeResult("err", val))
	case "to_json":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: to_json() requires 1 argument")
		}
		val := v.stack[len(v.stack)-1]
		v.stack = v.stack[:len(v.stack)-1]
		canonical, err := json.Marshal(funnyToGo(val))
		if err != nil {
			return fmt.Errorf("vm: to_json: marshal error: %v", err)
		}
		v.stack = append(v.stack, string(canonical))
	case "parse_json":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: parse_json() requires 1 argument")
		}
		s, ok := v.stack[len(v.stack)-1].(string)
		if !ok {
			return fmt.Errorf("vm: parse_json() requires a string argument")
		}
		v.stack = v.stack[:len(v.stack)-1]
		var x any
		if err := json.Unmarshal([]byte(s), &x); err != nil {
			return fmt.Errorf("vm: parse_json: invalid JSON: %v", err)
		}
		v.stack = append(v.stack, convertJSON(x))
	case "now":
		v.stack = append(v.stack, int(time.Now().Unix()))
	case "time_format":
		if len(v.stack) < 2 {
			return fmt.Errorf("vm: time_format() requires 2 arguments")
		}
		layout := v.stack[len(v.stack)-1].(string)
		ts := v.stack[len(v.stack)-2].(int)
		v.stack = v.stack[:len(v.stack)-2]
		t := time.Unix(int64(ts), 0)
		v.stack = append(v.stack, t.Format(layout))
	case "sqrt":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: sqrt() requires 1 argument")
		}
		x := toFloat(v.stack[len(v.stack)-1])
		v.stack = v.stack[:len(v.stack)-1]
		v.stack = append(v.stack, math.Sqrt(x))
	case "pow":
		if len(v.stack) < 2 {
			return fmt.Errorf("vm: pow() requires 2 arguments")
		}
		exp := toFloat(v.stack[len(v.stack)-1])
		base := toFloat(v.stack[len(v.stack)-2])
		v.stack = v.stack[:len(v.stack)-2]
		v.stack = append(v.stack, math.Pow(base, exp))
	case "abs":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: abs() requires 1 argument")
		}
		x := v.stack[len(v.stack)-1]
		v.stack = v.stack[:len(v.stack)-1]
		switch val := x.(type) {
		case int:
			if val < 0 {
				v.stack = append(v.stack, -val)
			} else {
				v.stack = append(v.stack, val)
			}
		case float64:
			v.stack = append(v.stack, math.Abs(val))
		default:
			return fmt.Errorf("vm: abs() requires a number")
		}
	case "str_upper":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: str_upper() requires 1 argument")
		}
		s, _ := v.stack[len(v.stack)-1].(string)
		v.stack = v.stack[:len(v.stack)-1]
		v.stack = append(v.stack, strings.ToUpper(s))
	case "str_lower":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: str_lower() requires 1 argument")
		}
		s, _ := v.stack[len(v.stack)-1].(string)
		v.stack = v.stack[:len(v.stack)-1]
		v.stack = append(v.stack, strings.ToLower(s))
	case "str_contains":
		if len(v.stack) < 2 {
			return fmt.Errorf("vm: str_contains() requires 2 arguments")
		}
		substr := v.stack[len(v.stack)-1].(string)
		s := v.stack[len(v.stack)-2].(string)
		v.stack = v.stack[:len(v.stack)-2]
		v.stack = append(v.stack, strings.Contains(s, substr))
	case "str_split":
		if len(v.stack) < 2 {
			return fmt.Errorf("vm: str_split() requires 2 arguments")
		}
		sep := v.stack[len(v.stack)-1].(string)
		s := v.stack[len(v.stack)-2].(string)
		v.stack = v.stack[:len(v.stack)-2]
		parts := strings.Split(s, sep)
		out := make([]bytecode.Value, len(parts))
		for i, p := range parts {
			out[i] = p
		}
		v.stack = append(v.stack, out)
	case "regex_match":
		if len(v.stack) < 2 {
			return fmt.Errorf("vm: regex_match() requires 2 arguments")
		}
		re, err := regexp.Compile(v.stack[len(v.stack)-2].(string))
		if err != nil {
			return fmt.Errorf("vm: regex_match: %v", err)
		}
		s := v.stack[len(v.stack)-1].(string)
		v.stack = v.stack[:len(v.stack)-2]
		v.stack = append(v.stack, re.MatchString(s))
	case "regex_replace":
		if len(v.stack) < 3 {
			return fmt.Errorf("vm: regex_replace() requires 3 arguments")
		}
		repl := v.stack[len(v.stack)-1].(string)
		s := v.stack[len(v.stack)-2].(string)
		re, err := regexp.Compile(v.stack[len(v.stack)-3].(string))
		if err != nil {
			return fmt.Errorf("vm: regex_replace: %v", err)
		}
		v.stack = v.stack[:len(v.stack)-3]
		v.stack = append(v.stack, re.ReplaceAllString(s, repl))
	case "env_get":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: env_get() requires 1 argument")
		}
		key := v.stack[len(v.stack)-1].(string)
		v.stack = v.stack[:len(v.stack)-1]
		v.stack = append(v.stack, os.Getenv(key))
	case "file_read":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: file_read() requires 1 argument")
		}
		path := v.stack[len(v.stack)-1].(string)
		v.stack = v.stack[:len(v.stack)-1]
		data, err := os.ReadFile(path)
		if err != nil {
			v.stack = append(v.stack, makeResult("err", err.Error()))
			return nil
		}
		v.stack = append(v.stack, makeResult("ok", string(data)))
	case "file_exists":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: file_exists() requires 1 argument")
		}
		path := v.stack[len(v.stack)-1].(string)
		v.stack = v.stack[:len(v.stack)-1]
		_, err := os.Stat(path)
		v.stack = append(v.stack, err == nil)
	case "http_get":
		if len(v.stack) < 1 {
			return fmt.Errorf("vm: http_get() requires 1 argument")
		}
		url := v.stack[len(v.stack)-1].(string)
		v.stack = v.stack[:len(v.stack)-1]
		resp, err := http.Get(url)
		if err != nil {
			v.stack = append(v.stack, makeResult("err", err.Error()))
			return nil
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			v.stack = append(v.stack, makeResult("err", err.Error()))
			return nil
		}
		v.stack = append(v.stack, makeResult("ok", string(data)))
	default:
		return fmt.Errorf("vm: unknown builtin %q", name)
	}
	return nil
}

// toFloat converts an int or float to float64. Other types panic.
func toFloat(val bytecode.Value) float64 {
	switch x := val.(type) {
	case int:
		return float64(x)
	case float64:
		return x
	}
	panic(fmt.Sprintf("vm: expected number, got %T", val))
}

// funnyToGo converts a funny bytecode.Value (using []bytecode.Value and
// map[string]bytecode.Value) into generic Go types ([]any, map[string]any)
// that encoding/json can marshal directly.
func funnyToGo(val bytecode.Value) any {
	switch v := val.(type) {
	case nil:
		return nil
	case bool, int, float64, string:
		return v
	case []bytecode.Value:
		out := make([]any, len(v))
		for i, e := range v {
			out[i] = funnyToGo(e)
		}
		return out
	case map[string]bytecode.Value:
		out := make(map[string]any, len(v))
		for k, e := range v {
			out[k] = funnyToGo(e)
		}
		return out
	}
	return fmt.Sprintf("%v", val)
}
