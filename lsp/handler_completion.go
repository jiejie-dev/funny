package lsp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jerloo/funny"
	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
	"go.uber.org/zap"
)

func (h Handler) handleTextDocumentCompletion(ctx context.Context, conn jsonrpc2.JSONRPC2, req *jsonrpc2.Request, params lsp.CompletionParams) (*lsp.CompletionList, error) {
	defer func() { // 必须要先声明defer，否则不能捕获到panic异常
		if err := recover(); err != nil {
			h.log.Error("error happend", zap.Error(err.(error)))
		}
	}()
	if !IsURI(params.TextDocument.URI) {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: fmt.Sprintf("textDocument/completion not yet supported for out-of-workspace URI (%q)", params.TextDocument.URI),
		}
	}

	cl := &lsp.CompletionList{
		IsIncomplete: false,
		Items:        make([]lsp.CompletionItem, 0),
	}
	contents, ok := h.documentContents.Get(string(params.TextDocument.URI))
	if !ok {
		return cl, errors.New("document content not found")
	}
	builtinParser := funny.NewParser([]byte(funny.BuiltinsDotFunny), "")
	builtinBlock, err := builtinParser.Parse()
	if err != nil {
		return nil, err
	}

	parser := funny.NewParser(contents, UriToRealPath(params.TextDocument.URI))
	items, err := parser.Parse()
	if err != nil {
		return nil, err
	}

	var currentToken *funny.Token
	var lastToken funny.Token
	var fields []string
	for index, token := range parser.Tokens {
		if token.Position.Line == params.Position.Line && token.Position.Col+token.Position.Length == params.Position.Character {
			currentToken = &token
			if index < len(parser.Tokens)-1 {
				if parser.Tokens[index+1].Kind == funny.DOT {
					fields = append(fields, token.Data)
				}
			}
			if index > 0 {
				lastToken = parser.Tokens[index-1]
			}
			break
		}
	}
	l := 0
	if currentToken != nil {
		l = currentToken.Position.Length
		h.log.Info("current", zap.Any("current", currentToken))
	}
	if currentToken.Kind == funny.DOT {
		builtinFieldBlock := getFieldBlock(h.log, []*funny.Block{builtinBlock}, fields)
		blocks := collectBlocks(h.log, params.Position.Line, items)
		fieldBlock := getFieldBlock(h.log, blocks, fields)
		var bs []*funny.Block
		if fieldBlock != nil {
			bs = append(bs, fieldBlock)
		}
		if builtinFieldBlock != nil {
			bs = append(bs, builtinFieldBlock)
		}
		cl.Items = collectCompletionItems(params, bs, l, true)
	} else if lastToken.Kind == funny.DOT {
		builtinFieldBlock := getFieldBlock(h.log, []*funny.Block{builtinBlock}, fields)
		blocks := collectBlocks(h.log, params.Position.Line, items)
		fieldBlock := getFieldBlock(h.log, blocks, fields)
		var bs []*funny.Block
		if fieldBlock != nil {
			bs = append(bs, fieldBlock)
		}
		if builtinFieldBlock != nil {
			bs = append(bs, builtinFieldBlock)
		}
		cl.Items = collectCompletionItems(params, bs, -l, true)
	} else {
		// h.log.Info("tokens", zap.Any("tokens", parser.Tokens))

		blocks := collectBlocks(h.log, params.Position.Line, items)
		h.log.Info("funny:completion", zap.Any("blocks", blocks))

		fds := collectCompletionItems(params, blocks, l, false)
		fdsBuiltins := collectCompletionItems(params, []*funny.Block{builtinBlock}, l, false)
		fds = append(fds, fdsBuiltins...)
		h.log.Info("funny:completion", zap.Any("fds", fds))
		cl.Items = fds
	}
	return cl, nil
}

func getFieldBlock(logger *zap.Logger, blocks []*funny.Block, fieldAccess []string) *funny.Block {
	for _, block := range blocks {
		for _, statement := range block.Statements {
			if a, ok := statement.(*funny.Assign); ok {
				if v, o := a.Target.(*funny.Variable); o {
					if len(fieldAccess) == 1 || len(fieldAccess) == 0 {
						switch fb := a.Value.(type) {
						case *funny.Block:
							return fb
						case *funny.ImportFunctionCall:
							return fb.Block
						case *funny.Function:
							return fb.Body
						case *funny.FunctionCall:
							panic(fmt.Errorf("not support %s", funny.Typing(a.Value)))
						}
						// panic(fmt.Errorf("if v, o := a.Target.(*funny.Variable); o not support %s %s %v", funny.Typing(a.Value), v.Name, a.Value))
					}
					if len(fieldAccess) > 0 {
						if v.Name == fieldAccess[0] {
							switch fb := a.Value.(type) {
							case *funny.Block:
								sub := fieldAccess[1:]
								if len(sub) > 0 {
									return getFieldBlock(logger, []*funny.Block{fb}, sub)
								} else {
									return fb
								}
							case *funny.Field:
								sub := fieldAccess[1:]
								if len(sub) > 0 {
									return getFieldBlock(logger, []*funny.Block{
										{
											Statements: []funny.Statement{
												fb,
											},
										},
									}, fieldAccess[1:])
								} else {
									return &funny.Block{
										Statements: []funny.Statement{
											fb,
										},
									}
								}
							case *funny.ImportFunctionCall:
								sub := fieldAccess[1:]
								if len(sub) > 0 {
									return getFieldBlock(logger, []*funny.Block{fb.Block}, fieldAccess[1:])
								} else {
									return fb.Block
								}
							}
							panic(fmt.Errorf("v.Name == fieldAccess[0] not support %s %v %s", funny.Typing(a.Value), a.Value, v.Name))
						}
					}
				}
			}
		}
	}
	return nil
}

func collectBlocks(logger *zap.Logger, line int, block *funny.Block) (results []*funny.Block) {
	if len(results) == 0 {
		results = append(results, block)
	}
	for _, statement := range block.Statements {
		if v, ok := statement.(*funny.Block); ok {
			if line >= v.GetPosition().Line && line < v.EndPosition().Line {
				results = append(results, v)
			}
		}
	}
	return
}

func collectCompletionItems(params lsp.CompletionParams, block []*funny.Block, l int, ifdot bool) (results []lsp.CompletionItem) {
	for _, b := range block {
		var comments []*funny.Comment
		newLineCount := 0
		for _, statement := range b.Statements {
			ci := lsp.CompletionItem{}
			switch v := statement.(type) {
			case *funny.Function:
				ci.Label = v.Name
				ci.Detail = v.SignatureString()
			case *funny.Variable:
				ci.Label = v.Name
			case *funny.Assign:
				if target, ok := v.Target.(*funny.Variable); ok {
					ci.Label = target.Name
				}
			case *funny.Block:
				brs := collectCompletionItems(params, []*funny.Block{v}, l, ifdot)
				results = append(results, brs...)
				newLineCount = 0
			case *funny.NewLine:
				newLineCount++
				if newLineCount > 1 {
					comments = make([]*funny.Comment, 0)
				}
			case *funny.Comment:
				comments = append(comments, v)
				newLineCount = 0
			}
			ci.Documentation = joinComments(comments)
			if !ifdot {
				ci.TextEdit = &lsp.TextEdit{
					Range: lsp.Range{
						Start: lsp.Position{
							Line:      params.Position.Line,
							Character: params.Position.Character - l,
						},
						End: lsp.Position{
							Line:      params.Position.Line,
							Character: params.Position.Character,
						},
					},
					NewText: ci.Label,
				}
			} else {
				ci.TextEdit = &lsp.TextEdit{
					Range: lsp.Range{
						Start: lsp.Position{
							Line:      params.Position.Line,
							Character: params.Position.Character + l,
						},
						End: lsp.Position{
							Line:      params.Position.Line,
							Character: params.Position.Character + l + len(ci.Label),
						},
					},
					NewText: ci.Label,
				}
			}
			if ci.Label != "" {
				results = append(results, ci)
			}
		}
	}
	return
}

func joinComments(comments []*funny.Comment) string {
	sb := new(strings.Builder)
	for _, item := range comments {
		sb.WriteString(item.Value)
		sb.WriteString("\n")
	}
	return sb.String()
}
