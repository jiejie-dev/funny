// v2/internal/cli/run.go
package cli

import (
	"encoding/json"

	"github.com/jerloo/funny/v2/internal/evaluator"
	"github.com/jerloo/funny/v2/internal/parser"
	"github.com/jerloo/funny/v2/internal/types"
)

func Run(src []byte, file string) error {
	p := parser.New(string(src), file)
	prog, err := p.Parse()
	if err != nil {
		return err
	}
	env := types.NewEnv(nil)
	if err := types.Check(prog, env); err != nil {
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
