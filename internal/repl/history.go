package repl

import "strings"

// History stores recent REPL inputs (newest last).
type History struct {
	lines []string
	limit int
}

func NewHistory(limit int) *History {
	if limit <= 0 {
		limit = 100
	}
	return &History{limit: limit}
}

func (h *History) Add(line string) {
	if h == nil {
		return
	}
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed[0] == ':' {
		return
	}
	h.lines = append(h.lines, trimmed)
	if len(h.lines) > h.limit {
		h.lines = h.lines[len(h.lines)-h.limit:]
	}
}

func (h *History) Lines() []string {
	if h == nil {
		return nil
	}
	out := make([]string, len(h.lines))
	copy(out, h.lines)
	return out
}
