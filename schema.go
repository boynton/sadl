package sadl

import (
	"encoding/json"
	"fmt"
)

var _ = json.Marshal
var _ = fmt.Printf

type BaseType int

const (
	_ BaseType = iota
	BaseTypeBool
	BaseTypeInt8
	BaseTypeInt16
	BaseTypeInt32
	BaseTypeInt64
	BaseTypeFloat32
	BaseTypeFloat64
	BaseTypeDecimal
	BaseTypeBytes
	BaseTypeString
	BaseTypeTimestamp
	BaseTypeQuantity
	BaseTypeUUID
	BaseTypeArray
	BaseTypeMap
	BaseTypeStruct
	BaseTypeEnum
	BaseTypeUnion
	BaseTypeAny
)

var namesBaseType = []string{
	BaseTypeBool:      "Bool",
	BaseTypeInt8:      "Int8",
	BaseTypeInt16:     "Int16",
	BaseTypeInt32:     "Int32",
	BaseTypeInt64:     "Int64",
	BaseTypeFloat32:   "Float32",
	BaseTypeFloat64:   "Float64",
	BaseTypeDecimal:   "Decimal",
	BaseTypeBytes:     "Bytes",
	BaseTypeString:    "String",
	BaseTypeTimestamp: "Timestamp",
	BaseTypeQuantity:  "Quantity",
	BaseTypeUUID:      "UUID",
	BaseTypeArray:     "Array",
	BaseTypeMap:       "Map",
	BaseTypeStruct:    "Struct",
	BaseTypeEnum:      "Enum",
	BaseTypeUnion:     "Union",
	BaseTypeAny:       "Any",
}

type Schema struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Version     string            `json:"version,omitempty"`
	Comment     string            `json:"comment,omitempty"`
	Types       []*TypeDef        `json:"types,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type TypeDef struct {
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	Comment     string            `json:"comment,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`

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

type EnumElementDef struct {
	Symbol      string            `json:"symbol"`
	Comment     string            `json:"comment,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type StructFieldDef struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Required    bool              `json:"required,omitempty"`
	Default     interface{}       `json:"default,omitempty"`
	Comment     string            `json:"comment,omitempty"`
	Items       string            `json:"items,omitempty"`
	Keys        string            `json:"keys,omitempty"`
	Value       string            `json:"value,omitempty"`
	Unit        string            `json:"unit,omitempty"`
	Pattern     string            `json:"pattern,omitempty"`
	Values      []string          `json:"values,omitempty"`
	Min         *Decimal          `json:"min,omitempty"`
	Max         *Decimal          `json:"max,omitempty"`
	MinSize     *int32            `json:"minsize,omitempty"`
	MaxSize     *int32            `json:"maxsize,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}
