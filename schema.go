package sadl

var BaseTypes = []string{
	"Bool",
	"Int8",
	"Int16",
	"Int32",
	"Int64",
	"Float32",
	"Float64",
	"Decimal",
	"Bytes",
	"String",
	"Timestamp",
	"UnitValue",
	"UUID",
	"Array",
	"Map",
	"Struct",
	"Enum",
	"Union",
	"Any",
}

type Schema struct {
	Sadl        string            `json:"sadl"`
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Version     string            `json:"version,omitempty"`
	Comment     string            `json:"comment,omitempty"`
	Types       []*TypeDef        `json:"types,omitempty"`
	Examples    []*ExampleDef     `json:"examples,omitempty"`
	Operations  []*OperationDef   `json:"operations,omitempty"`
	Http        []*HttpDef        `json:"http,omitempty"`
	Base        string            `json:"base,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type TypeSpec struct {
	Type      string             `json:"type"`
	Pattern   string             `json:"pattern,omitempty"`
	Values    []string           `json:"values,omitempty"`
	MinSize   *int64             `json:"minSize,omitempty"`
	MaxSize   *int64             `json:"maxSize,omitempty"`
	Fields    []*StructFieldDef  `json:"fields,omitempty"`
	Elements  []*EnumElementDef  `json:"elements,omitempty"`
	Min       *Decimal           `json:"min,string,omitempty"`
	Max       *Decimal           `json:"max,string,omitempty"`
	Items     string             `json:"items,omitempty"`
	Keys      string             `json:"keys,omitempty"`
	Variants  []*UnionVariantDef `json:"variants,omitempty"` //FIXME: a variant element, so comments/annotations can be attached
	Unit      string             `json:"unit,omitempty"`
	Value     string             `json:"value,omitempty"`
	Reference string             `json:"reference,omitempty"`
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

type UnionVariantDef struct {
	Name        string            `json:"name"`
	Comment     string            `json:"comment,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	TypeSpec
}

type ExampleDef struct {
	Target      string            `json:"target"`
	Name        string            `json:"name,omitempty"`
	Example     interface{}       `json:"example,omitempty"`
	Comment     string            `json:"comment,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type OperationDef struct {
	Name        string             `json:"name,omitempty"`
	Comment     string             `json:"comment,omitempty"`
	Annotations map[string]string  `json:"annotations,omitempty"`
	Inputs      []*OperationInput  `json:"inputs,omitempty"`
	Outputs     []*OperationOutput `json:"outputs,omitempty"`
	Exceptions  []string           `json:"exceptions,omitempty"`
}

type OperationInput struct {
	StructFieldDef
}

type OperationOutput struct {
	Name        string            `json:"name"`
	Comment     string            `json:"comment,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	TypeSpec
}

type HttpDef struct {
	Name        string               `json:"name,omitempty"`
	Resource    string               `json:"resource,omitempty"`
	Comment     string               `json:"comment,omitempty"`
	Annotations map[string]string    `json:"annotations,omitempty"`
	Method      string               `json:"method"`
	Path        string               `json:"path"`
	Inputs      []*HttpParamSpec     `json:"inputs,omitempty"`
	Expected    *HttpExpectedSpec    `json:"expected,omitempty"`
	Exceptions  []*HttpExceptionSpec `json:"exceptions,omitempty"`
}

type HttpParamSpec struct {
	Header string `json:"header,omitempty"`
	Query  string `json:"query,omitempty"`
	Path   bool   `json:"path,omitempty"`
	StructFieldDef
}

type HttpExpectedSpec struct {
	Outputs     []*HttpParamSpec  `json:"outputs,omitempty"`
	Status      int32             `json:"status"`
	Comment     string            `json:"comment,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type HttpExceptionSpec struct {
	Type        string            `json:"type"`
	Status      int32             `json:"status"`
	Comment     string            `json:"comment,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}
