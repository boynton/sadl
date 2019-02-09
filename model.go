package sadl

import (
	"encoding/json"
	"fmt"
	"regexp"
)

type Model struct {
	Schema
	typeIndex map[string]*TypeDef
}

func NewModel(schema *Schema) (*Model, error) {
	model := &Model{
		Schema:    *schema,
		typeIndex: make(map[string]*TypeDef, 0),
	}
	for _, name := range BaseTypes {
		model.typeIndex[name] = &TypeDef{Name: name, TypeSpec: TypeSpec{Type: name}}
	}
	for _, td := range schema.Types {
		if _, ok := model.typeIndex[td.Name]; ok {
			return nil, fmt.Errorf("Duplicate type: %s", td.Name)
		}
		model.typeIndex[td.Name] = td
	}
	return model, nil
}

func (model *Model) FindType(name string) *TypeDef {
	if model.typeIndex != nil {
		if t, ok := model.typeIndex[name]; ok {
			return t
		}
	}
	return nil
}

func (model *Model) ValidateAgainstTypeSpec(td *TypeSpec, value interface{}) error {
	switch td.Type {
	case "Timestamp":
		return model.ValidateTimestamp(td, value)
	case "String":
		return model.ValidateString(td, value)
	default:
		return nil
	}
}

func (model *Model) Validate(typename string, value interface{}) error {
	td := model.FindType(typename)
	if td == nil {
		return fmt.Errorf("Undefined type: %s", typename)
	}
	return model.ValidateAgainstTypeSpec(&td.TypeSpec, value)
}

func (model *Model) ValidateString(td *TypeSpec, val interface{}) error {
	if sp, ok := val.(*string); ok {
		s := *sp
		if td.MinSize != nil {
			if len(s) < int(*td.MinSize) {
				return model.fail(td, val, fmt.Sprintf("'minsize=%d' constraint failed", *td.MinSize))
			}
		}
		if td.MaxSize != nil {
			if len(s) > int(*td.MaxSize) {
				return model.fail(td, val, fmt.Sprintf("'maxsize=%d' constraint failed", *td.MaxSize))
			}
		}
		if td.Values != nil {
			for _, match := range td.Values {
				if s == match {
					return nil
				}
			}
			return model.fail(td, val, fmt.Sprintf("'values=%v' constraint failed", td.Values))
		}
		if td.Pattern != "" {
			pat := td.Pattern
			matcher, err := regexp.Compile(pat)
			if err != nil {
				return model.fail(td, val, fmt.Sprintf("Bad pattern specified in String type definition %q", pat))
			}
			if !matcher.MatchString(s) {
				return model.fail(td, val, fmt.Sprintf("'pattern=%q' constraint failed", pat))
			}
		}
		return nil
	}
	return model.fail(td, val, "type mismatch")
}

func (model *Model) ValidateTimestamp(td *TypeSpec, val interface{}) error {
	if _, ok := val.(*Timestamp); ok {
		return nil
	}
	if s, ok := val.(*string); ok {
		_, err := ParseTimestamp(*s)
		if err == nil {
			return nil
		}
	}
	return model.fail(td, val, "format invalid")
}

func (model *Model) IsNumericType(td *TypeSpec) bool {
	switch td.Type {
	case "Int8", "Int16", "Int32", "Int64", "Float32", "Float64", "Decimal":
		return true
	}
	return false
}

//and so on

func (model *Model) fail(td *TypeSpec, val interface{}, msg string) error {
	return fmt.Errorf("Validation error: not a valid %s (%s): %s", td.Type, msg, pretty(val))
}

func pretty(obj interface{}) string {
	j, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		return fmt.Sprint(obj)
	}
	return string(j)
}
