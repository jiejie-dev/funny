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
	ReferencesProvider         bool                  `json:"referencesProvider"`
	RenameProvider             *RenameOptions        `json:"renameProvider,omitempty"`
}

type RenameOptions struct {
	PrepareProvider bool `json:"prepareProvider"`
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

// --- References ---

type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

type ReferenceParams struct {
	TextDocumentPositionParams
	Context ReferenceContext `json:"context"`
}

// --- Rename ---

type RenameParams struct {
	TextDocumentPositionParams
	NewName string `json:"newName"`
}

type PrepareRenameParams = TextDocumentPositionParams

type PrepareRenameResult struct {
	Range       Range  `json:"range"`
	Placeholder string `json:"placeholder"`
}

type WorkspaceEdit struct {
	Changes map[string][]TextEdit `json:"changes"`
}

// --- funny/planGraph (custom extension) ---
//
// This is not part of standard LSP 3.17; it's a funny-specific request
// that lets an editor render a `plan` block as a step graph instead of
// (or alongside) plain text. See docs/language-manual.md's "LSP Server"
// section for the schema description.

type PlanGraphParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type PlanGraphResult struct {
	Plans []PlanGraph `json:"plans"`
}

type PlanGraph struct {
	Name  string     `json:"name"`
	Range Range      `json:"range"`
	Nodes []PlanNode `json:"nodes"`
	Edges []PlanEdge `json:"edges"`
}

type PlanNode struct {
	ID       string     `json:"id"`
	Label    string     `json:"label"`
	Kind     string     `json:"kind"` // step kind: tool/guard/transform/parallel/branch/delay, or "task" for a parallel step's concurrent child
	Range    Range      `json:"range"`
	Retry    *RetryInfo `json:"retry,omitempty"`
	Timeout  string     `json:"timeout,omitempty"`
	ParentID string     `json:"parentId,omitempty"` // set on a parallel step's concurrent children
}

type RetryInfo struct {
	Max     int      `json:"max"`
	Backoff string   `json:"backoff,omitempty"`
	On      []string `json:"on,omitempty"`
}

// PlanEdge.Kind is "sequence" (A runs, then B runs) or "parallel" (A
// spawns concurrent child B; see internal/agent/engine.go's execParallel).
type PlanEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
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
