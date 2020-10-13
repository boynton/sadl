package graphql

import (
	"bufio"
	"bytes"
	"fmt"

	"github.com/boynton/sadl"
)

func Export(model *sadl.Model, conf *sadl.Data) error {
	s, err := FromSadl(model, conf)
	if err != nil {
		panic(fmt.Sprintf("*** %v\n", err))
		//		return err
	}
	fmt.Print(s)
	return nil
}

func FromSadl(model *sadl.Model, conf *sadl.Data) (string, error) {
	w := &GraphqlWriter{
		//		namespace: ns,
		model:         model,
		config:        conf,
		arrays:        make(map[string]*sadl.TypeDef, 0),
		customScalars: make(map[string]bool, 0),
	}
	w.Begin()
	var err error
	for _, td := range model.Types {
		if td.Type == "Array" {
			w.arrays[td.Name] = td
		}
	}
	for _, td := range model.Types {
		switch td.Type {
		case "Enum":
			err = w.EmitEnumDef(td)
		case "Struct":
			err = w.EmitStructDef(td)
		case "Union":
			err = w.EmitUnionDef(td)
		case "String", "Bool", "Int8", "Int16", "Int32", "Int64", "Float32", "Float64", "Decimal":
			//these get replace by the equivalent type, you cannot "subtype" thesein GraphQL
		case "Array":
			//skip these. All references to it should be replaced with literal GraphQL list syntax.
		default:
			panic("fixme: " + td.Type)
		}
		if err != nil {
			return "", err
		}
	}
	for k, _ := range w.customScalars {
		w.Emit("scalar %s\n", k)
	}
	return w.End(), nil
}

type GraphqlWriter struct {
	model         *sadl.Model
	arrays        map[string]*sadl.TypeDef
	config        *sadl.Data
	buf           bytes.Buffer
	writer        *bufio.Writer
	namespace     string
	name          string
	version       string
	customScalars map[string]bool
}

func (w *GraphqlWriter) Begin() {
	w.buf.Reset()
	w.writer = bufio.NewWriter(&w.buf)
}

func (w *GraphqlWriter) Emit(format string, args ...interface{}) {
	w.writer.WriteString(fmt.Sprintf(format, args...))
}

func (w *GraphqlWriter) EmitEnumDef(td *sadl.TypeDef) error {
	w.Emit("enum %s {\n", td.Name)
	for _, ed := range td.Elements {
		w.Emit("  %s\n", ed.Symbol)
	}
	w.Emit("}\n\n")
	return nil
}

func (w *GraphqlWriter) EmitStructDef(td *sadl.TypeDef) error {
	w.Emit("type %s {\n", td.Name)
	for _, fd := range td.Fields {
		required := ""
		if fd.Required {
			required = "!"
		}
		if fd.Comment != "" {
			//TO DO: format as block before the field
			w.Emit("  # %s\n", fd.Comment)
		}
		ftype := w.typeRef(&fd.TypeSpec)
		w.Emit("  %s: %s%s\n", fd.Name, ftype, required)
	}
	w.Emit("}\n\n")
	return nil
}

func (w *GraphqlWriter) customScalar(name string, defaultMapping string) string {
	tname := w.config.GetString("custom-scalars", name)
	if tname != "" {
		w.customScalars[tname] = true
		return tname
	}
	return defaultMapping
}

func (w *GraphqlWriter) scalarType(name string) string {
	switch name {
	case "Bool":
		return "Boolean"
	case "String":
		return "String"
	case "UUID", "Timestamp":
		return w.customScalar(name, "String")
	case "Int32", "Int16", "Int8":
		return "Int"
	case "Float64", "Float32":
		return "Float"
	case "Int64", "Decimal":
		return w.customScalar(name, "Float")
	default:
		return ""
	}
}

func (w *GraphqlWriter) typeRef(ts *sadl.TypeSpec) string {
	if td, ok := w.arrays[ts.Type]; ok {
		ts = &td.TypeSpec
	}
	tsName := w.scalarType(ts.Type)
	if tsName != "" {
		return tsName
	}
	switch ts.Type {
	case "Array":
		return fmt.Sprintf("[%s!]", ts.Items)
	default:
		td := w.model.FindType(ts.Type)
		if td != nil {
			tsName = w.scalarType(td.Type)
			if tsName != "" {
				return tsName
			}
		}
		return ts.Type
	}
}

func (w *GraphqlWriter) EmitUnionDef(td *sadl.TypeDef) error {
	w.Emit("union %s =\n", td.Name)
	for i, uv := range td.Variants {
		if i > 0 {
			w.Emit("  | ")
		} else {
			w.Emit("    ")
		}
		vtype := uv.Type
		w.Emit("%s\n", vtype)
	}
	w.Emit("\n")
	return nil
}

func (w *GraphqlWriter) End() string {
	w.writer.Flush()
	return w.buf.String()
}
