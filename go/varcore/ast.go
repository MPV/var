package varcore

// ast.go — the parser's immutable AST. Port of ast.py / ast.ts.
//
// A Block is one recognized markdown block. Concrete block types carry
// exported fields; the unexported kind()/blockSpan() methods let the structurer
// treat blocks uniformly while the conformance projection type-switches on the
// concrete type. Blocks are values, never mutated after construction.

// SegmentOffset maps a block-text offset to its source offset. Block text is
// the raw source minus BLOCK markers only (list bullets, blockquote `>`
// prefixes), so a paragraph or list item has a single entry and a blockquote
// one entry per quoted line. Inline markup is never stripped.
type SegmentOffset struct {
	TextOffset   int
	SourceOffset int
}

// Block is any recognized markdown block node.
type Block interface {
	kind() string
	blockSpan() Span
}

// Heading is an ATX heading (`# ...`). Becomes a scope marker, not an example.
type Heading struct {
	Level int
	Text  string
	Sp    Span
}

func (h Heading) kind() string    { return "heading" }
func (h Heading) blockSpan() Span { return h.Sp }

// Paragraph is a run of non-blank, non-structural lines.
type Paragraph struct {
	Text       string
	Sp         Span
	SegmentMap []SegmentOffset
}

func (p Paragraph) kind() string    { return "paragraph" }
func (p Paragraph) blockSpan() Span { return p.Sp }

// ListItem is a single ordered or unordered list item.
type ListItem struct {
	Text       string
	Sp         Span
	SegmentMap []SegmentOffset
	Ordered    bool
	MarkerSpan Span
}

func (l ListItem) kind() string    { return "list_item" }
func (l ListItem) blockSpan() Span { return l.Sp }

// Blockquote is one or more contiguous `>`-prefixed lines.
type Blockquote struct {
	Text       string
	Sp         Span
	SegmentMap []SegmentOffset
}

func (b Blockquote) kind() string    { return "blockquote" }
func (b Blockquote) blockSpan() Span { return b.Sp }

// Row is one row of a table: trimmed cells plus a span per cell.
type Row struct {
	Cells     []string
	CellSpans []Span
	Sp        Span
}

// Table is a header row, a delimiter row (consumed, not stored), and body rows.
type Table struct {
	Sp     Span
	Header Row
	Rows   []Row
}

func (t Table) kind() string    { return "table" }
func (t Table) blockSpan() Span { return t.Sp }

// Fence is a fenced code block (```), which the plan stage may read as a doc
// string.
type Fence struct {
	Sp       Span
	Info     string
	Body     string
	BodySpan Span
}

func (f Fence) kind() string    { return "fence" }
func (f Fence) blockSpan() Span { return f.Sp }

// ThematicBreak is a horizontal rule (`---`). Closes an open attachment.
type ThematicBreak struct {
	Sp Span
}

func (t ThematicBreak) kind() string    { return "thematic_break" }
func (t ThematicBreak) blockSpan() Span { return t.Sp }

// Example is a group of blocks under a heading scope.
type Example struct {
	ScopeStack []string
	Sp         Span
	Body       []Block
}

// VarDoc is the parsed document: its examples and any orphan table/fence
// attachments that were not merged into an example.
type VarDoc struct {
	Path              string
	Source            string
	Examples          []Example
	OrphanAttachments []Block // Table or Fence
}
