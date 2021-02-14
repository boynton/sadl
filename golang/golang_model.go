package golang

import (
	"fmt"
	"text/template"

	"github.com/boynton/sadl"
)

func (gen *Generator) CreateModel() {
	if gen.Err != nil {
		return
	}
	gen.Begin()
	gen.EmitInterface()
	gen.EmitTypeDefs()
	if gen.createTimestamp {
		gen.EmitTimestamp()
	}
	if gen.createDecimal {
		gen.EmitDecimal()
	}
	content := gen.End()
	fname := sadl.Uncapitalize(gen.Name) + "_model.go"
	gen.WriteGoFile(fname, content, gen.Pkg)
}

func (gen *Generator) EmitInterface() {
	funcMap := template.FuncMap{
		"openBrace":   func() string { return "{" },
		"serviceName": func() string { return gen.Name },
		"signature": func(hd *sadl.HttpDef) string {
			name := gen.Capitalize(hd.Name)
			reqType := gen.RequestTypeName(hd)
			resType := gen.ResponseTypeName(hd)
			return name + "(req *" + reqType + ") (*" + resType + ", error)"
		},
		"capitalize": func(s string) string { return gen.Capitalize(s) },
	}
	gen.EmitTemplate("interface", interfaceTemplate, gen, funcMap)
}

var interfaceTemplate = `
//
// {{serviceName}} is the interface that the service implementation must conform to
//
type {{serviceName}} interface {{openBrace}}{{range .Model.Http}}
    {{signature .}}{{end}}
}

`

func (gen *Generator) RequestTypeName(hd *sadl.HttpDef) string {
	return gen.Capitalize(hd.Name) + "Request"
}

func (gen *Generator) ResponseTypeName(hd *sadl.HttpDef) string {
	return gen.Capitalize(hd.Name) + "Response"
}

func (gen *Generator) EmitRequestType(hd *sadl.HttpDef) {
	name := gen.RequestTypeName(hd)
	td := &sadl.TypeDef{
		Name: name,
	}
	td.Type = "Struct"
	fields := make([]*sadl.StructFieldDef, 0)
	for _, p := range hd.Inputs {
		fields = append(fields, &p.StructFieldDef)
	}
	td.Fields = fields
	gen.EmitStructType(td, nil)
}

func (gen *Generator) EmitResponseType(hd *sadl.HttpDef) {
	name := gen.ResponseTypeName(hd)
	td := &sadl.TypeDef{
		Name: name,
	}
	td.Type = "Struct"
	fields := make([]*sadl.StructFieldDef, 0)
	for _, p := range hd.Expected.Outputs {
		fields = append(fields, &p.StructFieldDef)
	}
	td.Fields = fields
	gen.EmitStructType(td, nil)
}

func (gen *Generator) EmitTypeDefs() {
	errors := make(map[string]bool, 0)
	for _, hd := range gen.Model.Http {
		gen.EmitRequestType(hd)
		gen.EmitResponseType(hd)
		for _, ed := range hd.Exceptions {
			errors[ed.Type] = true
		}
	}
	for _, td := range gen.Model.Types {
		gen.EmitType(td, errors)
	}
}

func (gen *Generator) EmitType(td *sadl.TypeDef, errors map[string]bool) {
	gen.Emit("\n//\n// " + td.Name + "\n//\n")
	switch td.Type {
	case "Struct":
		gen.EmitStructType(td, errors)
	case "UnitValue":
		gen.EmitUnitValueType(td)
	case "Enum":
		gen.EmitEnumType(td)
	case "String":
		gen.Emit("type " + td.Name + " string\n")
	case "Decimal":
		gen.Emit("type " + td.Name + " Decimal\n")
	case "Array":
		gen.EmitArrayType(td)
	default:
		fmt.Println(td.Type)
		panic("Check this")
		//do nothing, i.e. a String subclass
	}

}

func (gen *Generator) EmitArrayType(td *sadl.TypeDef) {
	itemType := gen.nativeTypeName(&td.TypeSpec, td.Items)
	gen.Emit("type " + td.Name + " []" + itemType + "\n")
}

func (gen *Generator) EmitStructType(td *sadl.TypeDef, errors map[string]bool) {
	gen.Emit("type " + td.Name + " struct {\n")
	for _, fd := range td.Fields {
		fname := capitalize(fd.Name)
		ftype := gen.nativeTypeName(&fd.TypeSpec, fd.Type)
		anno := " `json:\"" + fd.Name
		if !fd.Required {
			anno = anno + ",omitempty"
		}
		anno = anno + "\"`"
		gen.Emit("    " + fname + " " + ftype + anno + "\n")
	}
	gen.Emit("}\n\n")
	if errors != nil {
		if _, ok := errors[td.Name]; ok {
			gen.Emit("func (e *" + td.Name + ") Error() string {\n\treturn \"" + td.Name + "\"\n}\n\n")
		}
	}
}

func (gen *Generator) EmitEnumType(td *sadl.TypeDef) {
	if gen.Err != nil {
		return
	}
	funcMap := template.FuncMap{
		"openBrace": func() string { return "{" },
	}
	gen.EmitTemplate("enumType", enumTemplate, td, funcMap)
}

const enumTemplate = `type {{.Name}} int

const (
    _ {{.Name}} = iota{{range .Elements}}
    {{.Symbol}}{{end}}
)

var names{{.Name}} = []string{{openBrace}}{{range .Elements}}
    {{.Symbol}}: "{{.Symbol}}",{{end}}
}

func (e {{.Name}}) String() string {
    return names{{.Name}}[e]
}

func (e {{.Name}}) MarshalJSON() ([]byte, error) {
    return json.Marshal(e.String())
}

func (e *{{.Name}}) UnmarshalJSON(b []byte) error {
    var s string
    err := json.Unmarshal(b, &s)
    if err == nil {
        for v, s2 := range names{{.Name}} {
            if s == s2 {
                *e = {{.Name}}(v)
                return nil
             }
        }
        err = fmt.Errorf("Bad enum symbol for type {{.Name}}: %s", s)
    }
    return err
}
`

func (gen *Generator) EmitDecimal() {
	if gen.Err != nil {
		return
	}
	gen.addImport("encoding/json")
	gen.addImport("fmt")
	gen.addImport("math/big")
	gen.Emit(decimalType)
}

const decimalType = `// Decimal is a big.Float equivalent, but marshals to JSON as strings to preserve precision.

const DecimalPrecision = uint(250)

type Decimal struct {
	big.Float
}

// Encode as a string. Encoding as a JSON number works fine, but the Unbmarshal doesn't. If we use string as the representation in JSON, it works fine.
// What a shame.
func (d *Decimal) MarshalJSON() ([]byte, error) {
	repr := d.Text('f', -1)
	stringRepr := "\"" + repr + "\""
	return []byte(stringRepr), nil
}

func (d *Decimal) UnmarshalJSON(b []byte) error {
	var stringRepr string
	err := json.Unmarshal(b, &stringRepr)
	if err == nil {
		var num *Decimal
		num, err = ParseDecimal(stringRepr)
		if err == nil {
			*d = *num
			return nil
		}
	} else {
		var floatRepr float64
		err = json.Unmarshal(b, &floatRepr)
		if err == nil {
         *d = *NewDecimal(floatRepr)
			return nil
		}
	}
	return fmt.Errorf("Bad Decimal number: %s", string(b))
}

func ParseDecimal(text string) (*Decimal, error) {
	num, _, err := big.ParseFloat(text, 10, DecimalPrecision, big.ToNearestEven)
	if err != nil {
		return nil, fmt.Errorf("Bad Decimal number: %s", text)
	}
   return &Decimal{Float:*num}, nil
}

func NewDecimal(val float64) *Decimal {
   return &Decimal{Float:*big.NewFloat(val)}
}

func (d *Decimal) String() string {
	return fmt.Sprint(d)
}

func (d *Decimal) AsInt32() int32 {
	n := d.AsInt64()
	return int32(n)
}

func (d *Decimal) AsInt64() int64 {
	i, _ := d.Int64()
	return i
}

func (d *Decimal) AsFloat64() float64 {
	f, _ := d.Float64()
	return f
}

func (d *Decimal) AsBigFloat() *big.Float {
   return &d.Float
}

`

func (gen *Generator) EmitTimestamp() {
	if gen.Err != nil {
		return
	}
	gen.addImport("encoding/json")
	gen.addImport("fmt")
	gen.addImport("strings")
	gen.addImport("time")
	gen.Emit(timestamp)
}

var timestamp = `

type Timestamp struct {
	time.Time
}

const RFC3339Milli = "%d-%02d-%02dT%02d:%02d:%02d.%03dZ"

func (ts Timestamp) String() string {
	if ts.IsZero() {
		return ""
	}
	return fmt.Sprintf(RFC3339Milli, ts.Year(), ts.Month(), ts.Day(), ts.Hour(), ts.Minute(), ts.Second(), ts.Nanosecond()/1000000)
}

func (ts Timestamp) MarshalJSON() ([]byte, error) {
	return []byte("\"" + ts.String() + "\""), nil
}

func (ts *Timestamp) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err == nil {
		var tsp Timestamp
		tsp, err = ParseTimestamp(string(j))
		if err == nil {
			*ts = tsp
		}
	}
	return err
}

func ParseTimestamp(s string) (Timestamp, error) {
	layout := "2006-01-02T15:04:05.999Z" //derive this from the spec used for output?
	t, e := time.Parse(layout, s)
	if e != nil {
		if strings.HasSuffix(s, "+00:00") || strings.HasSuffix(s, "-00:00") {
			t, e = time.Parse(layout, s[:len(s)-6]+"Z")
		} else if strings.HasSuffix(s, "+0000") || strings.HasSuffix(s, "-0000") {
			t, e = time.Parse(layout, s[:len(s)-5]+"Z")
		} else {
			t, e = time.Parse("Mon, 02 Jan 2006 15:04:05 GMT", s) //Last-Modified, etc are of in RFC2616 format
		}
		if e != nil {
			var ts Timestamp
			return ts, fmt.Errorf("Bad Timestamp: %q", s)
		}
	}
	return Timestamp{t}, nil
}
`

func (gen *Generator) EmitUnitValueType(td *sadl.TypeDef) {
	panic("emitUnitValueType NYI")
}
