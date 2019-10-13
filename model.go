package sadl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
)

type Model struct {
	Schema
	Extensions map[string]interface{} `json:"extensions,omitempty"`
	typeIndex  map[string]*TypeDef
	httpIndex  map[string]*HttpDef
}

func NewModel(schema *Schema) (*Model, error) {
	model := &Model{
		Schema:    *schema,
		typeIndex: make(map[string]*TypeDef, 0),
		httpIndex: make(map[string]*HttpDef, 0),
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
	for _, hd := range schema.Http {
		if _, ok := model.httpIndex[hd.Name]; ok {
			return nil, fmt.Errorf("Duplicate http action: %q", hd.Name)
		}
		model.httpIndex[hd.Name] = hd
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

func (model *Model) FindHttp(name string) *HttpDef {
	if model.httpIndex != nil {
		if h, ok := model.httpIndex[name]; ok {
			return h
		}
	}
	return nil
}

func (model *Model) EquivalentTypesByName(tname1, tname2 string) bool {
	if tname1 == tname2 {
		return true
	}
	td1 := model.FindType(tname1)
	td2 := model.FindType(tname2)
	return model.EquivalentTypes(&td1.TypeSpec, &td2.TypeSpec)
}

func (model *Model) EquivalentTypes(ts1, ts2 *TypeSpec) bool {
	if ts1.Type != ts2.Type {
		return false
	}
	//TODO: check annotations
	switch ts1.Type {
	case "String":
		if ts1.Pattern != ts2.Pattern || ts1.MinSize != ts2.MinSize || ts1.MaxSize != ts1.MaxSize || len(ts1.Values) != len(ts2.Values) {
			return false
		}
		if ts1.Reference != ts2.Reference {
			return false
		}
		if len(ts1.Values) > 0 {
			m := make(map[string]bool, 0)
			for _, v := range ts1.Values {
				m[v] = true
			}
			for _, v := range ts2.Values {
				if _, ok := m[v]; !ok {
					return false
				}
			}
		}
	case "UUID":
		if ts1.Reference != ts2.Reference {
			return false
		}
	case "Struct":
		if len(ts1.Fields) != len(ts2.Fields) {
			return false
		}
	case "Bytes":
		if ts1.MinSize != ts2.MinSize || ts1.MaxSize != ts1.MaxSize {
			return false
		}
	case "UnitValue":
		if !model.EquivalentTypesByName(ts1.Unit, ts2.Unit) {
			return false
		}
		if !model.EquivalentTypesByName(ts1.Value, ts2.Value) {
			return false
		}
	case "Array", "Map":
		if ts1.MinSize != ts2.MinSize || ts1.MaxSize != ts1.MaxSize {
			return false
		}
		if !model.EquivalentTypesByName(ts1.Items, ts2.Items) {
			return false
		}
		if !model.EquivalentTypesByName(ts1.Keys, ts2.Keys) {
			return false
		}
	case "Int8", "Int16", "Int32", "Int64", "Float32", "Float64", "Decimal":
		if ts1.Min != ts2.Min || ts1.Max != ts2.Max {
			return false
		}
	case "Enum":
		if len(ts1.Elements) != len(ts2.Elements) {
			return false
		}
		//order doesn't matter for enums
		m := make(map[string]bool, 0)
		for _, v := range ts1.Elements {
			//TODO: check annotations
			m[v.Symbol] = true
		}
		for _, v := range ts2.Elements {
			if _, ok := m[v.Symbol]; !ok {
				return false
			}
		}
	case "Union":
		if len(ts1.Variants) != len(ts2.Variants) {
			return false
		}
		m := make(map[string]bool, 0)
		for _, v := range ts1.Variants {
			m[v] = true
		}
		for _, v := range ts2.Variants {
			if _, ok := m[v]; !ok {
				return false
			}
		}
	}
	return true
}

func (model *Model) Validate(context string, typename string, value interface{}) error {
	td := model.FindType(typename)
	if td == nil {
		return fmt.Errorf("Undefined type: %s", typename)
	}
	if context == "" {
		context = typename
	}
	return model.ValidateAgainstTypeSpec(context, &td.TypeSpec, value)
}

func (model *Model) ValidateAgainstTypeSpec(context string, td *TypeSpec, value interface{}) error {
	if context == "" {
		context = td.Type
	}
	switch td.Type {
	case "Timestamp":
		return model.ValidateTimestamp(context, td, value)
	case "String":
		return model.ValidateString(context, td, value)
	case "Struct":
		return model.ValidateStruct(context, td, value)
	case "Array":
		return model.ValidateArray(context, td, value)
	case "Map":
		return model.ValidateMap(context, td, value)
	case "Int8", "Int16", "Int32", "Int64", "Float32", "Float64", "Decimal":
		return model.ValidateNumber(context, td, value)
	case "Bool":
		return model.ValidateBool(context, td, value)
	case "Enum":
		return model.ValidateEnum(context, td, value)
	case "UnitValue":
		return model.ValidateUnitValue(context, td, value)
	case "UUID":
		return model.ValidateUUID(context, td, value)
	case "Bytes":
		//fixme
		return nil
	case "Any":
		//must be ok
		return nil
	default:
		t := model.FindType(td.Type)
		if t == nil {
			return fmt.Errorf("%s: no such type '%s'", context, td.Type)
		}
		return model.ValidateAgainstTypeSpec(context, &t.TypeSpec, value)
	}
}

func (model *Model) ValidateBool(context string, td *TypeSpec, value interface{}) error {
	switch value.(type) {
	case *bool, bool:
		return nil
	}
	return fmt.Errorf("%s: Not valid: %v", context, Pretty(value))
}

func (model *Model) ValidateUUID(context string, td *TypeSpec, value interface{}) error {
	var s string
	switch sp := value.(type) {
	case UUID:
		return nil
	case *string:
		s = *sp
	case string:
		s = sp
	}
	if s != "" {
		tmp := ParseUUID(s)
		if tmp != "" {
			return nil
		}
	}
	return fmt.Errorf("%s: Not valid: %v", context, Pretty(value))
}

func (model *Model) ValidateUnitValue(context string, td *TypeSpec, value interface{}) error {
	switch sp := value.(type) {
	case *string:
		s := *sp
		n := strings.Index(s, " ")
		if n >= 3 {
			val := s[:n]
			unit := s[n+1:]
			nval, err := ParseDecimal(val)
			if err == nil {
				err = model.Validate(context+".value", td.Value, nval)
				if err == nil {
					err = model.Validate(context+".unit", td.Unit, unit)
				}
			}
			return err
		}
	}
	return fmt.Errorf("%s: Not valid: %v", context, Pretty(value))
}

func (model *Model) ValidateEnum(context string, td *TypeSpec, value interface{}) error {
	var s string
	switch sp := value.(type) {
	case *string:
		s = *sp
	case string:
		s = sp
	}
	if s != "" {
		for _, el := range td.Elements {
			if el.Symbol == s {
				return nil
			}
		}
	}
	return fmt.Errorf("%s: Not valid: %v", context, Pretty(value))
}

func (model *Model) ValidateNumber(context string, td *TypeSpec, value interface{}) error {
	switch n := value.(type) {
	case *Decimal:
		//number restrictions: min and max, which as expressed as Decimal numbers
		if td != nil {
			var minval *Decimal
			var maxval *Decimal
			switch td.Type {
			case "Decimal", "Float64", "Float32":
				minval = DecimalValue(td.Min, nil)
				maxval = DecimalValue(td.Max, nil)
				//no other limits
			case "Int64":
				minval = DecimalValue(td.Min, math.MinInt64)
				maxval = DecimalValue(td.Max, math.MaxInt64)
			case "Int32":
				minval = DecimalValue(td.Min, math.MinInt32)
				maxval = DecimalValue(td.Max, math.MaxInt32)
			case "Int16":
				minval = DecimalValue(td.Min, math.MinInt16)
				maxval = DecimalValue(td.Max, math.MaxInt16)
			case "Int8":
				minval = DecimalValue(td.Min, math.MinInt8)
				maxval = DecimalValue(td.Max, math.MaxInt8)
			}
			nval := n.AsBigFloat()
			if minval != nil {
				nmin := minval.AsBigFloat()
				if nval.Cmp(nmin) < 0 {
					return fmt.Errorf("%s: Numeric value less than the minimum allowed (%v)", context, minval)
				}
			}
			if maxval != nil {
				nmax := maxval.AsBigFloat()
				if nval.Cmp(nmax) > 0 {
					return fmt.Errorf("%s: Numeric value greater than the maximum allowed (%v)", context, maxval)
				}
			}
		}
	default:
		return fmt.Errorf("%s: Not a number: %v", context, Pretty(value))
	}
	return nil
}

func (model *Model) IsStructField(ts *TypeSpec, name string) bool {
	for _, field := range ts.Fields {
		if name == field.Name {
			return true
		}
	}
	return false
}

func (model *Model) ValidateStruct(context string, td *TypeSpec, value interface{}) error {
	switch m := value.(type) {
	case map[string]interface{}:
		for k, _ := range m {
			if !model.IsStructField(td, k) {
				return fmt.Errorf("Undefined field in %s: '%s'", context, k)
			}
		}
		for _, field := range td.Fields {
			if v, ok := m[field.Name]; ok {
				err := model.ValidateAgainstTypeSpec(context+"."+field.Name, &field.TypeSpec, v)
				if err != nil {
					return err
				}
			} else {
				if field.Required {
					return fmt.Errorf("%s missing required field '%s': %s", context, field.Name, Pretty(value))
				}
			}
		}
	default:
		return fmt.Errorf("Not a Struct: %s", Pretty(td))
	}
	return nil
}

func (model *Model) ValidateArray(context string, td *TypeSpec, value interface{}) error {
	switch a := value.(type) {
	case []interface{}:
		if td.Items != "Any" {
			tdi := model.FindType(td.Items)
			if tdi == nil {
				return fmt.Errorf("%s: Undefined type: %s", context, td.Items)
			}
			for i, item := range a {
				err := model.ValidateAgainstTypeSpec(fmt.Sprintf("%s[%d]", context, i), &tdi.TypeSpec, item)
				if err != nil {
					return err
				}
			}
		}
		if td.MaxSize != nil {
			if len(a) > int(*td.MaxSize) {
				return fmt.Errorf("%s: Array is too large (maxsize=%d): %v", context, *td.MaxSize, Pretty(value))
			}
		}
		if td.MinSize != nil {
			if len(a) < int(*td.MinSize) {
				return fmt.Errorf("%s: Array is too small (minsize=%d): %v", context, *td.MinSize, Pretty(value))
			}
		}
		return nil
	default:
		return fmt.Errorf("%s: Not an Array: %v", context, value)
	}
}

func (model *Model) ValidateMap(context string, td *TypeSpec, value interface{}) error {
	switch a := value.(type) {
	case map[string]interface{}:
		if td.Items != "Any" {
			tdi := model.FindType(td.Items)
			if tdi == nil {
				return fmt.Errorf("%s: Undefined type: %s", context, td.Items)
			}
			for k, item := range a {
				err := model.ValidateAgainstTypeSpec(fmt.Sprintf("%s[%q]", context, k), &tdi.TypeSpec, item)
				if err != nil {
					return err
				}
			}
		}
		if td.MaxSize != nil {
			if len(a) > int(*td.MaxSize) {
				return fmt.Errorf("%s: Map is too large (maxsize=%d): %v", context, *td.MaxSize, Pretty(value))
			}
		}
		if td.MinSize != nil {
			if len(a) < int(*td.MinSize) {
				return fmt.Errorf("%s: Map is too small (minsize=%d): %v", context, *td.MinSize, Pretty(value))
			}
		}
		return nil
	default:
		return fmt.Errorf("%s: Not an Array: %v", context, value)
	}
}

func (model *Model) ValidateString(context string, td *TypeSpec, val interface{}) error {
	var s string
	if sp, ok := val.(*string); ok {
		s = *sp
	} else if ss, ok := val.(string); ok {
		s = ss
	} else {
		return model.fail(td, val, context)
	}
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

func (model *Model) ValidateTimestamp(tname string, td *TypeSpec, val interface{}) error {
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

func IsSymbol(s string) bool {
	if s == "" {
		return false
	}
	if !IsSymbolChar(rune(s[0]), true) { //fixme
		return false
	}
	for _, ch := range s[1:] {
		if !IsSymbolChar(rune(ch), false) { //FIXME
			return false
		}
	}
	return true
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
	//numbers default to Decimal, which serializes to a JSON string, which makes the following message confusing.
	v := ""
	switch d := val.(type) {
	case *Decimal, int32, int64, int16, int8, float32, float64:
		v = fmt.Sprintf("%v", d)
	default:
		v = Pretty(val)
	}
	if msg != "" {
		msg = " (" + msg + ")"
	}
	return fmt.Errorf("Validation error: not a valid %s%s: %s", td.Type, msg, v)
}

func AsString(obj interface{}) string {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(&obj); err != nil {
		return fmt.Sprint(obj)
	}
	s := buf.String()
	s = strings.Trim(s, " \n")
	return string(s)
}
