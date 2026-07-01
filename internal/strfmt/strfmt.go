// Package strfmt implements a small, Python/Rust-flavored format-spec
// mini-language used by f-string interpolation (`f"{expr:spec}"`). It is
// shared by the type checker (spec validation), the tree-walking evaluator,
// and the bytecode VM's FORMAT_VALUE opcode, so all three execution paths
// stringify values identically.
//
// Spec grammar: [[fill]align][sign]['0'][width]['.' precision][type]
//   - align: one of < > ^; fill is a single char immediately before align
//     (default fill is a space)
//   - sign: '+' forces a leading sign on non-negative numbers
//   - '0': zero-pad shorthand (fill '0', align '>') when no explicit align given
//   - width: minimum field width (decimal digits)
//   - precision: '.' + decimal digits — decimal places for f/%, max length
//     for s/default
//   - type: one of d f x X o b s % (missing = default Stringify)
package strfmt

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var specRe = regexp.MustCompile(`^(?:(.)?([<>^]))?(\+)?(0)?([0-9]*)(?:\.([0-9]+))?([a-zA-Z%]?)$`)

// Spec is a parsed format spec.
type Spec struct {
	Fill         rune
	Align        rune // '<', '>', '^', or 0 (auto)
	Sign         bool
	ZeroPad      bool
	Width        int
	HasWidth     bool
	Precision    int
	HasPrecision bool
	Type         rune // 'd','f','x','X','o','b','s','%', or 0
}

// ParseSpec parses a format spec string (the part after ':' in `{expr:spec}`).
func ParseSpec(s string) (Spec, error) {
	if s == "" {
		return Spec{}, nil
	}
	m := specRe.FindStringSubmatch(s)
	if m == nil {
		return Spec{}, fmt.Errorf("invalid format spec %q", s)
	}
	var sp Spec
	if m[2] != "" {
		sp.Align = rune(m[2][0])
		if m[1] != "" {
			sp.Fill = rune(m[1][0])
		} else {
			sp.Fill = ' '
		}
	}
	sp.Sign = m[3] == "+"
	sp.ZeroPad = m[4] == "0"
	if m[5] != "" {
		w, _ := strconv.Atoi(m[5])
		sp.Width, sp.HasWidth = w, true
	}
	if m[6] != "" {
		pr, _ := strconv.Atoi(m[6])
		sp.Precision, sp.HasPrecision = pr, true
	}
	if m[7] != "" {
		sp.Type = rune(m[7][0])
	}
	return sp, nil
}

// Stringify converts a value to its default (spec-less) display string.
func Stringify(val any) string {
	switch v := val.(type) {
	case nil:
		return "nil"
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Format converts val to a string according to the given format spec string
// (the raw text after ':' — may be empty for default formatting).
func Format(val any, specStr string) (string, error) {
	sp, err := ParseSpec(specStr)
	if err != nil {
		return "", err
	}
	body, defaultAlign, err := render(val, sp)
	if err != nil {
		return "", err
	}
	return pad(body, sp, defaultAlign), nil
}

func toFloat(val any) (float64, bool) {
	switch v := val.(type) {
	case int:
		return float64(v), true
	case float64:
		return v, true
	}
	return 0, false
}

func toInt(val any) (int, bool) {
	switch v := val.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	}
	return 0, false
}

func render(val any, sp Spec) (body string, defaultAlign rune, err error) {
	switch sp.Type {
	case 0:
		return Stringify(val), '<', nil
	case 's':
		s := Stringify(val)
		if sp.HasPrecision && sp.Precision < len(s) {
			s = s[:sp.Precision]
		}
		return s, '<', nil
	case 'd':
		i, ok := toInt(val)
		if !ok {
			return "", 0, fmt.Errorf("format spec 'd' requires a number, got %T", val)
		}
		return signPrefix(i < 0, sp.Sign) + strconv.Itoa(abs(i)), '>', nil
	case 'x', 'X':
		i, ok := toInt(val)
		if !ok {
			return "", 0, fmt.Errorf("format spec '%c' requires a number, got %T", sp.Type, val)
		}
		s := strconv.FormatInt(int64(abs(i)), 16)
		if sp.Type == 'X' {
			s = strings.ToUpper(s)
		}
		return signPrefix(i < 0, sp.Sign) + s, '>', nil
	case 'o':
		i, ok := toInt(val)
		if !ok {
			return "", 0, fmt.Errorf("format spec 'o' requires a number, got %T", val)
		}
		return signPrefix(i < 0, sp.Sign) + strconv.FormatInt(int64(abs(i)), 8), '>', nil
	case 'b':
		i, ok := toInt(val)
		if !ok {
			return "", 0, fmt.Errorf("format spec 'b' requires a number, got %T", val)
		}
		return signPrefix(i < 0, sp.Sign) + strconv.FormatInt(int64(abs(i)), 2), '>', nil
	case 'f':
		f, ok := toFloat(val)
		if !ok {
			return "", 0, fmt.Errorf("format spec 'f' requires a number, got %T", val)
		}
		prec := 6
		if sp.HasPrecision {
			prec = sp.Precision
		}
		return signPrefix(f < 0, sp.Sign) + strconv.FormatFloat(absf(f), 'f', prec, 64), '>', nil
	case '%':
		f, ok := toFloat(val)
		if !ok {
			return "", 0, fmt.Errorf("format spec '%%' requires a number, got %T", val)
		}
		prec := 6
		if sp.HasPrecision {
			prec = sp.Precision
		}
		return signPrefix(f < 0, sp.Sign) + strconv.FormatFloat(absf(f*100), 'f', prec, 64) + "%", '>', nil
	}
	return "", 0, fmt.Errorf("unknown format type %q", string(sp.Type))
}

func signPrefix(neg, forcePlus bool) string {
	if neg {
		return "-"
	}
	if forcePlus {
		return "+"
	}
	return ""
}

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

func absf(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func pad(body string, sp Spec, defaultAlign rune) string {
	if !sp.HasWidth || len(body) >= sp.Width {
		return body
	}
	align := sp.Align
	fill := sp.Fill
	if align == 0 {
		if sp.ZeroPad {
			align, fill = '>', '0'
		} else {
			align, fill = defaultAlign, ' '
		}
	}
	padLen := sp.Width - len(body)
	switch align {
	case '>':
		if fill == '0' && len(body) > 0 && (body[0] == '-' || body[0] == '+') {
			// Zero-padding: insert zeros after the sign, not before it.
			return body[:1] + strings.Repeat(string(fill), padLen) + body[1:]
		}
		return strings.Repeat(string(fill), padLen) + body
	case '^':
		left := padLen / 2
		right := padLen - left
		return strings.Repeat(string(fill), left) + body + strings.Repeat(string(fill), right)
	default: // '<'
		return body + strings.Repeat(string(fill), padLen)
	}
}
