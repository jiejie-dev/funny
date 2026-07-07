package stdlib

import (
	"crypto/md5"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "modernc.org/sqlite"
)

// Names is the set of stdlib builtin function names.
var Names = map[string]bool{
	"print": true, "println": true, "len": true, "to_str": true, "to_int": true,
	"to_float": true, "type_of": true, "ok": true, "err": true,
	"to_json": true, "parse_json": true, "now": true, "time_format": true,
	"sqrt": true, "pow": true, "abs": true,
	"str_upper": true, "str_lower": true, "str_contains": true, "str_split": true,
	"regex_match": true, "regex_replace": true,
	"env_get": true, "file_read": true, "file_exists": true, "http_get": true,
	"md5": true, "sha256": true, "b64_encode": true, "b64_decode": true,
	"jwt_encode": true, "jwt_decode": true, "sql_open": true, "append": true,
	"assert": true, "assert_eq": true,
}

// SideEffectOnly reports builtins that produce no meaningful return value.
func SideEffectOnly(name string) bool {
	return name == "print" || name == "println" || name == "assert" || name == "assert_eq"
}

// Call dispatches a stdlib builtin by name. Args are in source order.
func Call(name string, args []any) (any, error) {
	switch name {
	case "print":
		for i, a := range args {
			if i > 0 {
				fmt.Print(" ")
			}
			fmt.Print(a)
		}
		return nil, nil
	case "println":
		for i, a := range args {
			if i > 0 {
				fmt.Print(" ")
			}
			fmt.Print(a)
		}
		fmt.Println()
		return nil, nil
	case "len":
		if len(args) != 1 {
			return nil, fmt.Errorf("len() takes exactly 1 argument")
		}
		switch val := args[0].(type) {
		case string:
			return len(val), nil
		case []any:
			return len(val), nil
		default:
			return reflect.ValueOf(val).Len(), nil
		}
	case "append":
		if len(args) != 2 {
			return nil, fmt.Errorf("append() takes exactly 2 arguments")
		}
		lst, ok := args[0].([]any)
		if !ok {
			return nil, fmt.Errorf("append() first argument must be a list")
		}
		out := make([]any, len(lst)+1)
		copy(out, lst)
		out[len(lst)] = args[1]
		return out, nil
	case "to_str":
		if len(args) != 1 {
			return nil, fmt.Errorf("to_str() takes exactly 1 argument")
		}
		return fmt.Sprintf("%v", args[0]), nil
	case "to_int":
		if len(args) != 1 {
			return nil, fmt.Errorf("to_int() takes exactly 1 argument")
		}
		switch x := args[0].(type) {
		case int:
			return x, nil
		case float64:
			return int(x), nil
		case string:
			n, err := strconv.Atoi(strings.TrimSpace(x))
			if err != nil {
				var digits int
				for _, c := range x {
					if c >= '0' && c <= '9' {
						digits = digits*10 + int(c-'0')
					}
				}
				return digits, nil
			}
			return n, nil
		}
		return nil, fmt.Errorf("to_int() not supported for type %T", args[0])
	case "to_float":
		if len(args) != 1 {
			return nil, fmt.Errorf("to_float() takes exactly 1 argument")
		}
		switch x := args[0].(type) {
		case float64:
			return x, nil
		case int:
			return float64(x), nil
		case string:
			f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
			if err != nil {
				return float64(0), nil
			}
			return f, nil
		}
		return nil, fmt.Errorf("to_float() not supported for type %T", args[0])
	case "type_of":
		if len(args) != 1 {
			return nil, fmt.Errorf("type_of() takes exactly 1 argument")
		}
		switch args[0].(type) {
		case nil:
			return "nil", nil
		case bool:
			return "bool", nil
		case int:
			return "int", nil
		case float64:
			return "float", nil
		case string:
			return "str", nil
		case []any:
			return "list", nil
		case map[string]any:
			return "map", nil
		default:
			return "unknown", nil
		}
	case "ok":
		if len(args) != 1 {
			return nil, fmt.Errorf("ok() takes exactly 1 argument")
		}
		return MakeResult("ok", args[0]), nil
	case "err":
		if len(args) != 1 {
			return nil, fmt.Errorf("err() takes exactly 1 argument")
		}
		return MakeResult("err", args[0]), nil
	case "to_json":
		if len(args) != 1 {
			return nil, fmt.Errorf("to_json() takes exactly 1 argument")
		}
		s, err := marshalJSON(args[0])
		if err != nil {
			return nil, fmt.Errorf("to_json: %v", err)
		}
		return s, nil
	case "parse_json":
		if len(args) != 1 {
			return nil, fmt.Errorf("parse_json() takes exactly 1 argument")
		}
		s, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("parse_json() requires a string argument")
		}
		v, err := parseJSON(s)
		if err != nil {
			return nil, fmt.Errorf("parse_json: invalid JSON: %v", err)
		}
		return v, nil
	case "now":
		return int(time.Now().Unix()), nil
	case "time_format":
		if len(args) != 2 {
			return nil, fmt.Errorf("time_format() takes exactly 2 arguments")
		}
		ts, ok := args[0].(int)
		if !ok {
			return nil, fmt.Errorf("time_format() first argument must be int")
		}
		layout, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("time_format() second argument must be str")
		}
		return time.Unix(int64(ts), 0).Format(layout), nil
	case "sqrt":
		if len(args) != 1 {
			return nil, fmt.Errorf("sqrt() takes exactly 1 argument")
		}
		return math.Sqrt(toFloat(args[0])), nil
	case "pow":
		if len(args) != 2 {
			return nil, fmt.Errorf("pow() takes exactly 2 arguments")
		}
		return math.Pow(toFloat(args[0]), toFloat(args[1])), nil
	case "abs":
		if len(args) != 1 {
			return nil, fmt.Errorf("abs() takes exactly 1 argument")
		}
		switch val := args[0].(type) {
		case int:
			if val < 0 {
				return -val, nil
			}
			return val, nil
		case float64:
			return math.Abs(val), nil
		default:
			return nil, fmt.Errorf("abs() requires a number")
		}
	case "str_upper":
		if len(args) != 1 {
			return nil, fmt.Errorf("str_upper() takes exactly 1 argument")
		}
		return strings.ToUpper(args[0].(string)), nil
	case "str_lower":
		if len(args) != 1 {
			return nil, fmt.Errorf("str_lower() takes exactly 1 argument")
		}
		return strings.ToLower(args[0].(string)), nil
	case "str_contains":
		if len(args) != 2 {
			return nil, fmt.Errorf("str_contains() takes exactly 2 arguments")
		}
		return strings.Contains(args[0].(string), args[1].(string)), nil
	case "str_split":
		if len(args) != 2 {
			return nil, fmt.Errorf("str_split() takes exactly 2 arguments")
		}
		parts := strings.Split(args[0].(string), args[1].(string))
		out := make([]any, len(parts))
		for i, p := range parts {
			out[i] = p
		}
		return out, nil
	case "regex_match":
		if len(args) != 2 {
			return nil, fmt.Errorf("regex_match() takes exactly 2 arguments")
		}
		re, err := regexp.Compile(args[0].(string))
		if err != nil {
			return nil, fmt.Errorf("regex_match: %v", err)
		}
		return re.MatchString(args[1].(string)), nil
	case "regex_replace":
		if len(args) != 3 {
			return nil, fmt.Errorf("regex_replace() takes exactly 3 arguments")
		}
		re, err := regexp.Compile(args[0].(string))
		if err != nil {
			return nil, fmt.Errorf("regex_replace: %v", err)
		}
		return re.ReplaceAllString(args[1].(string), args[2].(string)), nil
	case "env_get":
		if len(args) != 1 {
			return nil, fmt.Errorf("env_get() takes exactly 1 argument")
		}
		return os.Getenv(args[0].(string)), nil
	case "file_read":
		if len(args) != 1 {
			return nil, fmt.Errorf("file_read() takes exactly 1 argument")
		}
		data, err := os.ReadFile(args[0].(string))
		if err != nil {
			return MakeResult("err", err.Error()), nil
		}
		return MakeResult("ok", string(data)), nil
	case "file_exists":
		if len(args) != 1 {
			return nil, fmt.Errorf("file_exists() takes exactly 1 argument")
		}
		_, err := os.Stat(args[0].(string))
		return err == nil, nil
	case "http_get":
		if len(args) != 1 {
			return nil, fmt.Errorf("http_get() takes exactly 1 argument")
		}
		resp, err := http.Get(args[0].(string))
		if err != nil {
			return MakeResult("err", err.Error()), nil
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return MakeResult("err", err.Error()), nil
		}
		return MakeResult("ok", string(data)), nil
	case "md5":
		if len(args) != 1 {
			return nil, fmt.Errorf("md5() takes exactly 1 argument")
		}
		h := md5.Sum([]byte(args[0].(string)))
		return hex.EncodeToString(h[:]), nil
	case "sha256":
		if len(args) != 1 {
			return nil, fmt.Errorf("sha256() takes exactly 1 argument")
		}
		h := sha256.Sum256([]byte(args[0].(string)))
		return hex.EncodeToString(h[:]), nil
	case "b64_encode":
		if len(args) != 1 {
			return nil, fmt.Errorf("b64_encode() takes exactly 1 argument")
		}
		return base64.StdEncoding.EncodeToString([]byte(args[0].(string))), nil
	case "b64_decode":
		if len(args) != 1 {
			return nil, fmt.Errorf("b64_decode() takes exactly 1 argument")
		}
		data, err := base64.StdEncoding.DecodeString(args[0].(string))
		if err != nil {
			return MakeResult("err", err.Error()), nil
		}
		return MakeResult("ok", string(data)), nil
	case "jwt_encode":
		if len(args) != 3 {
			return nil, fmt.Errorf("jwt_encode() takes exactly 3 arguments")
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"raw_header": args[0].(string),
			"raw_claims": args[1].(string),
		})
		s, err := token.SignedString([]byte(args[2].(string)))
		if err != nil {
			return MakeResult("err", err.Error()), nil
		}
		return s, nil
	case "jwt_decode":
		if len(args) != 2 {
			return nil, fmt.Errorf("jwt_decode() takes exactly 2 arguments")
		}
		parsed, err := jwt.Parse(args[0].(string), func(t *jwt.Token) (any, error) {
			return []byte(args[1].(string)), nil
		})
		if err != nil {
			return MakeResult("err", err.Error()), nil
		}
		if !parsed.Valid {
			return MakeResult("err", "invalid token"), nil
		}
		return MakeResult("ok", parsed.Claims), nil
	case "sql_open":
		if len(args) != 1 {
			return nil, fmt.Errorf("sql_open() takes exactly 1 argument")
		}
		path := args[0].(string)
		db, err := sql.Open("sqlite", path)
		if err != nil {
			return MakeResult("err", err.Error()), nil
		}
		_ = db
		return "sqlite:" + path, nil
	case "assert":
		if len(args) != 1 {
			return nil, fmt.Errorf("assert() takes exactly 1 argument")
		}
		b, ok := args[0].(bool)
		if !ok {
			return nil, fmt.Errorf("assert() argument must be bool")
		}
		if !b {
			return nil, fmt.Errorf("assertion failed")
		}
		return nil, nil
	case "assert_eq":
		if len(args) != 2 {
			return nil, fmt.Errorf("assert_eq() takes exactly 2 arguments")
		}
		if !valuesEqual(args[0], args[1]) {
			return nil, fmt.Errorf("assert_eq failed: %v != %v", args[0], args[1])
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown builtin %q", name)
	}
}

func toFloat(val any) float64 {
	switch x := val.(type) {
	case int:
		return float64(x)
	case float64:
		return x
	}
	panic(fmt.Sprintf("stdlib: expected number, got %T", val))
}

func valuesEqual(a, b any) bool {
	if a == b {
		return true
	}
	if fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b) {
		return true
	}
	return false
}
