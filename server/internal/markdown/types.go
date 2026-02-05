package markdown

// NodeType represents the type of a markdown element
type NodeType string

const (
	NodeDocument        NodeType = "document"
	NodeSection         NodeType = "section"
	NodeHeading         NodeType = "heading"
	NodeParagraph       NodeType = "paragraph"
	NodeList            NodeType = "list"
	NodeListItem        NodeType = "list_item"
	NodeFootnote        NodeType = "footnote"
	NodeCodeBlock       NodeType = "code_block"
	NodeHorizontalRule  NodeType = "horizontal_rule"
	NodeTable           NodeType = "table"
	NodeDisplayEquation NodeType = "display_equation"
)

// ListType represents ordered or unordered lists
type ListType string

const (
	ListOrdered   ListType = "ordered"
	ListUnordered ListType = "unordered"
)

// TableAlignment represents column alignment in tables
type TableAlignment string

const (
	AlignLeft   TableAlignment = "left"
	AlignCenter TableAlignment = "center"
	AlignRight  TableAlignment = "right"
	AlignNone   TableAlignment = "none"
)

// Node represents a node in the Markdown AST
type Node struct {
	Type           NodeType         `json:"type"`
	Content        string           `json:"content,omitempty"`
	Title          string           `json:"title,omitempty"`
	Level          int              `json:"level,omitempty"`
	ListType       ListType         `json:"list_type,omitempty"`
	Depth          int              `json:"depth,omitempty"`
	Index          int              `json:"index,omitempty"`
	FootnoteNumber int              `json:"footnote_number,omitempty"`
	IsMultiline    bool             `json:"is_multiline,omitempty"`
	Children       []*Node          `json:"children,omitempty"`
	Rows           []*TableRow      `json:"rows,omitempty"`
	Alignments     []TableAlignment `json:"alignments,omitempty"`
}

// TableRow represents a row in a markdown table
type TableRow struct {
	Cells    []string `json:"cells"`
	IsHeader bool     `json:"is_header"`
}
