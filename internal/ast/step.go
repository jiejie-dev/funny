// v2/internal/ast/step.go
package ast

// StepKind identifies the kind of step within a plan block.
type StepKind string

const (
	StepTool      StepKind = "tool"
	StepGuard     StepKind = "guard"
	StepTransform StepKind = "transform"
	StepParallel  StepKind = "parallel"
	StepBranch    StepKind = "branch"
	StepDelay     StepKind = "delay"
)

func (k StepKind) String() string { return string(k) }

// Step represents a single step within a plan block.
type Step struct {
	NodePos Pos
	Name    string
	Kind    StepKind
	Body    *Block
	// BranchCases holds the case-list for `-> branch` steps:
	//   cond => "target_step"
	// When non-empty, the engine dispatches to a named plan step instead of
	// running Body. Body remains for backward-compatible if/else fallback.
	BranchCases []BranchCase
	Retry       *Retry
	Timeout     string // raw duration string e.g. "5s"
}

// BranchCase maps a condition to a target step name in the same plan.
type BranchCase struct {
	Cond   Expression
	Target string
}

func (s *Step) Pos() Pos    { return s.NodePos }
func (s *Step) stmtMarker() {}
func (s *Step) nodeMarker() {}
func (s *Step) String() string {
	out := "step " + s.Name + " " + string(s.Kind) + ":\n"
	if s.Retry != nil {
		out += "    retry: " + s.Retry.String() + "\n"
	}
	if s.Timeout != "" {
		out += "    timeout: " + s.Timeout + "\n"
	}
	for _, c := range s.BranchCases {
		out += "    " + c.Cond.String() + " => \"" + c.Target + "\"\n"
	}
	if s.Body != nil {
		out += s.Body.String()
	}
	return out
}

// Retry config for a step.
type Retry struct {
	Max     int
	Backoff string // "constant" | "linear" | "exp"
	// On lists error type names to retry on (struct names, or "str" for
	// string errors). Empty means retry every failure.
	On []string
}

func (r *Retry) String() string {
	out := "max=" + itoa(r.Max) + " backoff=" + r.Backoff
	if len(r.On) > 0 {
		out += " on="
		for i, s := range r.On {
			if i > 0 {
				out += ","
			}
			out += s
		}
	}
	return out
}

// itoa is a tiny integer-to-string helper to avoid importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
