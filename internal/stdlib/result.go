package stdlib

// MakeResult constructs a Result runtime value: map{tag, val}.
func MakeResult(tag string, val any) map[string]any {
	return map[string]any{"tag": tag, "val": val}
}
