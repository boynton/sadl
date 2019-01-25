package main


type Schema struct {
	Namespace string         `json:"namespace"`
	Name string              `json:"name"`
	Version string           `json:"version"`
	Comment string           `json:"comment"`
	Types map[string]TypeDef `json:"types"`
}

type TypeDef struct {
	Name string   `json:"name"`
	Type string   `json:"type"`
	Struct *StructTypeDef `json:"struct,omitempty"`
}

type StructFieldDef struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type StructTypeDef struct {
	Fields []*StructFieldDef `json:"fields,omitempty"`
}
