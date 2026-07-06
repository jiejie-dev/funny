package bytecode

import "encoding/json"

// SourceMapEntry is one instruction's source mapping (1-based line/col for tools).
type SourceMapEntry struct {
	IP   int    `json:"ip"`
	Op   string `json:"op"`
	Arg  int    `json:"arg,omitempty"`
	File string `json:"file,omitempty"`
	Line int    `json:"line,omitempty"` // 1-based
	Col  int    `json:"col,omitempty"`  // 1-based
}

// FunctionMap is the source map for one compiled function.
type FunctionMap struct {
	Name         string           `json:"name"`
	Arity        int              `json:"arity"`
	NumLocals    int              `json:"numLocals"`
	LocalNames   []string         `json:"localNames,omitempty"`
	Instructions []SourceMapEntry `json:"instructions"`
}

// ModuleSourceMap is the JSON source map for a compiled module.
type ModuleSourceMap struct {
	Module    string        `json:"module"`
	Functions []FunctionMap `json:"functions"`
}

// SourceMap builds a JSON-serializable source map for the module.
func (m *Module) SourceMap() ModuleSourceMap {
	out := ModuleSourceMap{Module: m.Name}
	for _, fn := range m.Functions {
		fm := FunctionMap{
			Name:       fn.Name,
			Arity:      fn.Arity,
			NumLocals:  fn.NumLocals,
			LocalNames: fn.LocalNames,
		}
		for i, instr := range fn.Code {
			entry := SourceMapEntry{
				IP:  i,
				Op:  string(instr.Op),
				Arg: instr.Arg,
			}
			if i < len(fn.Locations) && !fn.Locations[i].IsZero() {
				loc := fn.Locations[i]
				entry.File = loc.File
				entry.Line = loc.Line + 1
				entry.Col = loc.Col + 1
			}
			fm.Instructions = append(fm.Instructions, entry)
		}
		out.Functions = append(out.Functions, fm)
	}
	return out
}

// SourceMapJSON returns the module source map as indented JSON.
func (m *Module) SourceMapJSON() ([]byte, error) {
	return json.MarshalIndent(m.SourceMap(), "", "  ")
}
