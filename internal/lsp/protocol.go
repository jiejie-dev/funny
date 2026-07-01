package lsp

// This file defines the minimal subset of the LSP 3.17 protocol types this
// server needs. It deliberately does not import a third-party LSP protocol
// library so the server has zero new external dependencies; only the
// message shapes actually produced/consumed below are modeled.

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

type TextDocumentItem struct {
	URI     string `json:"uri"`
	LangID  string `json:"languageId"`
	Version int    `json:"version"`
	Text    string `json:"text"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// --- Lifecycle ---

type InitializeParams struct {
	ProcessID int    `json:"processId"`
	RootURI   string `json:"rootUri"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   ServerInfo         `json:"serverInfo"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type TextDocumentSyncKind int

const (
	SyncNone TextDocumentSyncKind = 0
	SyncFull TextDocumentSyncKind = 1
)

type ServerCapabilities struct {
	TextDocumentSync           TextDocumentSyncKind  `json:"textDocumentSync"`
	HoverProvider              bool                  `json:"hoverProvider"`
	CompletionProvider         *CompletionOptions    `json:"completionProvider,omitempty"`
	SignatureHelpProvider      *SignatureHelpOptions `json:"signatureHelpProvider,omitempty"`
	DefinitionProvider         bool                  `json:"definitionProvider"`
	DocumentSymbolProvider     bool                  `json:"documentSymbolProvider"`
	DocumentFormattingProvider bool                  `json:"documentFormattingProvider"`
}

type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

type SignatureHelpOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

// --- Text document sync ---

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type TextDocumentContentChangeEvent struct {
	// Only full-document sync is supported, so Text is always the entire
	// new document content (no Range).
	Text string `json:"text"`
}

type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// --- Diagnostics ---

type DiagnosticSeverity int

const (
	SeverityError   DiagnosticSeverity = 1
	SeverityWarning DiagnosticSeverity = 2
	SeverityInfo    DiagnosticSeverity = 3
	SeverityHint    DiagnosticSeverity = 4
)

type Diagnostic struct {
	Range    Range              `json:"range"`
	Severity DiagnosticSeverity `json:"severity"`
	Code     string             `json:"code,omitempty"`
	Source   string             `json:"source"`
	Message  string             `json:"message"`
}

type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// --- Hover ---

type MarkupContent struct {
	Kind  string `json:"kind"` // "markdown" | "plaintext"
	Value string `json:"value"`
}

type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

// --- Completion ---

type CompletionItemKind int

const (
	CIKText     CompletionItemKind = 1
	CIKMethod   CompletionItemKind = 2
	CIKFunction CompletionItemKind = 3
	CIKField    CompletionItemKind = 5
	CIKVariable CompletionItemKind = 6
	CIKClass    CompletionItemKind = 7
	CIKKeyword  CompletionItemKind = 14
)

type CompletionItem struct {
	Label         string             `json:"label"`
	Kind          CompletionItemKind `json:"kind"`
	Detail        string             `json:"detail,omitempty"`
	Documentation string             `json:"documentation,omitempty"`
	InsertText    string             `json:"insertText,omitempty"`
}

type CompletionParams struct {
	TextDocumentPositionParams
}

// --- Signature help ---

type ParameterInformation struct {
	Label string `json:"label"`
}

type SignatureInformation struct {
	Label      string                 `json:"label"`
	Parameters []ParameterInformation `json:"parameters,omitempty"`
}

type SignatureHelp struct {
	Signatures      []SignatureInformation `json:"signatures"`
	ActiveSignature int                    `json:"activeSignature"`
	ActiveParameter int                    `json:"activeParameter"`
}

// --- Document symbols ---

type SymbolKind int

const (
	SKFile     SymbolKind = 1
	SKModule   SymbolKind = 2
	SKClass    SymbolKind = 5
	SKMethod   SymbolKind = 6
	SKField    SymbolKind = 8
	SKFunction SymbolKind = 12
	SKVariable SymbolKind = 13
	SKStruct   SymbolKind = 23
	SKEvent    SymbolKind = 24 // used for plan `step` nodes
)

type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           SymbolKind       `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// --- Formatting ---

type FormattingOptions struct {
	TabSize      int  `json:"tabSize"`
	InsertSpaces bool `json:"insertSpaces"`
}

type DocumentFormattingParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Options      FormattingOptions      `json:"options"`
}

type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}
