package parse

/*
type Node struct {
	Token
}

type StringNode struct {
	Node
	value string
}

type Int32Node struct {
	Node
	value *int32
}

type DecimalNode struct {
	Node
	value *Decimal
}

type StructFieldNode struct {
	Value DecimalNode
	value *Decimal

	Name        string            `json:"name"`
	Comment     string            `json:"comment,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Required    bool              `json:"required,omitempty"`
	Default     interface{}       `json:"default,omitempty"`
	TypeSpec
}

type TypeNode struct {
	Token //always UNDEFINED type, used only to locate the definition in the source file
	Type     *Node
	Pattern  *Node
	Values   []*Node
	MinSize  *Node
	MaxSize  *Node
	Fields   []*StructFieldDefNode
	Elements []*EnumElementDef `json:"elements,omitempty"`
	Min      *Decimal          `json:"min,string,omitempty"`
	Max      *Decimal          `json:"max,string,omitempty"`
	Items    string            `json:"items,omitempty"`
	Keys     string            `json:"keys,omitempty"`
	Variants []string          `json:"variants,omitempty"` //FIXME: a variant element, so comments/annotations can be attached
	Unit     string            `json:"unit,omitempty"`
	Value    string            `json:"value,omitempty"`
}

type TypeDefNode struct {
	Name        *Token            `json:"name"`
	Comment     string            `json:"comment,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	TypeNode //
}

type AST struct {
}


type SchemaNode struct {
	Name        StringNode
	Namespace   StringNode
	Version     StringNode
	Comment     StringNode
	Types       []*TypeDefNode
	Annotations map[string]StringNode
}

type StringNode struct {
	token Token
}
type TypeSpecNode struct {
	Type     string            `json:"type"`
	Pattern  string            `json:"pattern,omitempty"`
	Values   []string          `json:"values,omitempty"`
	MinSize  *int32            `json:"minSize,omitempty"`
	MaxSize  *int32            `json:"maxSize,omitempty"`
	Fields   []*StructFieldDef `json:"fields,omitempty"`
	Elements []*EnumElementDef `json:"elements,omitempty"`
	Min      *Decimal          `json:"min,string,omitempty"`
	Max      *Decimal          `json:"max,string,omitempty"`
	Items    string            `json:"items,omitempty"`
	Keys     string            `json:"keys,omitempty"`
	Variants []string          `json:"variants,omitempty"` //FIXME: a variant element, so comments/annotations can be attached
	Unit     string            `json:"unit,omitempty"`
	Value    string            `json:"value,omitempty"`
}

type TypeDef struct {
	Name        string            `json:"name"`
	Comment     string            `json:"comment,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	TypeSpec
}

type EnumElementDef struct {
	Symbol      string            `json:"symbol"`
	Comment     string            `json:"comment,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type StructFieldDef struct {
	Name        string            `json:"name"`
	Comment     string            `json:"comment,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Required    bool              `json:"required,omitempty"`
	Default     interface{}       `json:"default,omitempty"`
	TypeSpec
}
*/
