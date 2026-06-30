// v2/internal/cli/run.go
package cli

import (
	"encoding/json"

	"github.com/jerloo/funny/v2/internal/evaluator"
	"github.com/jerloo/funny/v2/internal/parser"
)

func Run(src []byte, file string) error {
	p := parser.New(string(src), file)
	prog, err := p.Parse()
	if err != nil {
		return err
	}
	e := evaluator.New(nil)
	return e.Exec(prog)
}

func Ast(src []byte, file string) ([]byte, error) {
	p := parser.New(string(src), file)
	prog, err := p.Parse()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(prog, "", "  ")
}
