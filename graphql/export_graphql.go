package graphql

import (
	"bufio"
	"bytes"
	"fmt"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/util"
	//	"github.com/graphql-go/graphql/language/ast"
)

func Export(model *sadl.Model, conf map[string]interface{}) error {
	s, err := FromSadl(model, conf)
	if err != nil {
		panic(fmt.Sprintf("*** %v\n", err))
		//		return err
	}
	fmt.Print(s)
	return nil
}

func FromSadl(model *sadl.Model, conf map[string]interface{}) (string, error) {
	w := &GraphqlWriter{
		//		namespace: ns,
		model:  model,
		conf:   conf,
		arrays: make(map[string]*sadl.TypeDef, 0),
	}

	fmt.Println(util.Pretty(model))
	fmt.Println("-------------")
	w.Begin()
	//	w.Emit("$version: %q\n", ast.Version) //only if a version-specific feature is needed. Could be "1" or "1.0"
	//	emitted := make(map[string]bool, 0)
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
		case "String":
			if td.Name == "ID" {
				continue
			}
			panic("fixme: " + td.Type + " for typedef named " + td.Name)
		case "Array":
			//skip these. All references to it should be replaced with literal GraphQL list syntax.
		default:
			panic("fixme: " + td.Type)
		}
		if err != nil {
			return "", err
		}
	}
	return w.End(), nil
}

type GraphqlWriter struct {
	model     *sadl.Model
	arrays    map[string]*sadl.TypeDef
	conf      map[string]interface{}
	buf       bytes.Buffer
	writer    *bufio.Writer
	namespace string
	name      string
	version   string
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

func (w *GraphqlWriter) typeRef(ts *sadl.TypeSpec) string {
	if td, ok := w.arrays[ts.Type]; ok {
		return w.typeRef(&td.TypeSpec)
	}
	switch ts.Type {
	case "Bool":
		return "Boolean"
	case "String":
		return "String"
	case "Int32", "Int16", "Int8":
		return "Int"
	case "Int64":
		return "GraphQL cannot represent integers with more than 32 bits of precision"
	case "Float64", "Float32":
		return "Double"
	case "Array":
		return fmt.Sprintf("[%s!]", ts.Items)
		//what is an n
	default:
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
