package smithy

import (
	"bytes"
	"fmt"
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/boynton/sadl"
)

func FromSADL(schema *sadl.Model, ns string) (*Model, error) {
	model := &Model{
		Version: "0.4.0",
		Namespaces: make(map[string]*Namespace, 0),
	}
/*
	model.Metadata = make(map[string]Node, 0)
	model.Metadata["foo"] = "baz"
	model.Metadata["hello"] = "bar"
	model.Metadata["lorem"] = map[string]interface{}{
		"ipsum": []string{"dolor"},
	}
*/
	theNamespace := &Namespace{
		Shapes: make(map[string]*Shape, 0),
	}
	model.Namespaces[ns] = theNamespace
	for _, td := range schema.Types {
		var shape Shape
		switch td.Type {
		case "String":
			shape = shapeFromString(td)
		case "Enum":
			shape = shapeFromEnum(td)
		case "Struct":
			shape = shapeFromStruct(td)
		default:
			fmt.Println("So far:", sadl.Pretty(model))
			panic("handle this type:" + sadl.Pretty(td))
		}
		if td.Comment != "" {
			shape.Documentation = td.Comment
		}
		if td.Annotations != nil {
			for k, v := range td.Annotations {
				switch k {
				case "x_sensitive":
					shape.Sensitive = true
				case "x_deprecated":
					dep := &Deprecated{}
					if v != "" {
						n := strings.Index(v, "|")
						if n >= 0 {
							dep.Since = v[:n]
							dep.Message = v[n+1:]
						} else {
							dep.Message = v
						}
					}
					shape.Deprecated = dep
				}
			}
		}
		theNamespace.Shapes[td.Name] = &shape
	}
	return model, nil
}

func typeReference(ts *sadl.TypeSpec) string {
	switch ts.Type {
	case "Bool":
		return "Boolean"
	case "Int8":
		return "Byte"
	case "Int16":
		return "Short"
	case "Int32":
		return "Integer"
	case "Int64":
		return "Long"
	case "Float32":
		return "Float"
	case "Float64":
		return "Double"
	case "Decimal":
		return "BigDecimal"
	case "Timestamp":
		return "Timestamp"
	case "UUID":
		return "String" //!
	case "Bytes":
		return "Blob"
	case "String":
		return "String"
	case "Array":
		return "List"
	case "Map":
		return "Map"
//	case "Struct": /naked struct
//		return "?"
	default:
		return ts.Type
	}
}

func listTypeReference(prefix string, fd *sadl.StructFieldDef) string {
	fmt.Println("FIX ME: inline defs not allowed, synthesize one to refer to:", prefix, sadl.Pretty(fd))
	ftype := capitalize(prefix) + capitalize(fd.Name)
	return ftype
}

func capitalize(s string) string {
	return strings.ToUpper(s[0:1]) + s[1:]
}

func shapeFromStruct(td *sadl.TypeDef) Shape {
	shape := Shape{
		Type: "structure",
	}
	members := make(map[string]*Member, 0)
	for _, fd := range td.Fields {
		ftype := typeReference(&fd.TypeSpec)
		switch ftype {
		case "List":
			//inline field defs not supported by Smithy, synthesize a type for the specific field
			ftype = listTypeReference(td.Name, fd)
		}
		member := &Member{
			Target: ftype,
		}
		members[fd.Name] = member
	}
	shape.Members = members
	return shape
}

func shapeFromString(td *sadl.TypeDef) Shape {
	shape := Shape{
		Type: "string",
	}
	min := int64(-1)
	max := int64(-1)
	if td.MinSize != nil {
		min = *td.MinSize
	}
	if td.MaxSize != nil {
		max = *td.MaxSize
	}
	shape.Length = length(min, max)
	if td.Pattern != "" {
		shape.Pattern = td.Pattern
	}
	return shape
}

func shapeFromEnum(td *sadl.TypeDef) Shape {
	shape := Shape{
		Type: "string",
	}
	//for sadl, enum values *are* the symbols, so the name must be set to match the key
	//note that this same form can work with values, where the name is optional but the key is the actual value
	items := make(map[string]*Item, 0)
	shape.Enum = items
	for _, el := range td.Elements {
		item := &Item{
			Name: el.Symbol, //the programmatic name, might be different than the value itself in Smithy. In SADL, they are the same.
			Documentation: el.Comment,
		}
		items[el.Symbol] = item
		//el.Annotations -> if contains x_tags, then expand to item.Tags
	}
	return shape
}

func length(min int64, max int64) *Length {
	l := &Length{}
	if min < 0 && max < 0 {
		return nil
	}
	if min >= 0 {
		l.Min = &min
	}
	if max >= 0 {
		l.Max = &max
	}
	return l
}

//generate Smithy IDL for the model
func (model *Model) IDL() string {
	w := &IdlWriter{}
	w.Begin()
	w.Emit("$version: %q\n", model.Version)
	for ns, namespace := range model.Namespaces {
		w.Emit("\nnamespace %s\n\n", ns)
		for k, v := range namespace.Shapes {
			w.EmitShape(k, v)
		}
		//out of band traits here
		for k, v := range namespace.Traits {
			fmt.Println("FIX ME trait", k, v)
		}
	}
	return w.End()
}

type IdlWriter struct {
	buf bytes.Buffer
	writer *bufio.Writer
}

func (w *IdlWriter) Begin() {
	w.buf.Reset()
   w.writer = bufio.NewWriter(&w.buf)
}

func (w *IdlWriter) Emit(format string, args ...interface{}) {
	w.writer.WriteString(fmt.Sprintf(format, args...))
}

func (w *IdlWriter) EmitShape(name string, shape *Shape) {
	w.EmitDocumentation(shape.Documentation, "")	
	w.EmitDeprecated(shape.Deprecated, "")
	w.EmitBooleanTrait(shape.Sensitive, "sensitive", "")
	w.EmitBooleanTrait(shape.Trait, "trait", "")
	w.EmitBooleanTrait(shape.ReadOnly, "readonly", "")
	w.EmitBooleanTrait(shape.Idempotent, "idempotent", "")
	switch shape.Type {
	case "boolean":
		w.EmitBooleanShape(name, shape)
	case "byte", "short", "integer", "long", "float", "double", "bigInteger", "bigDecimal":
		w.EmitNumericShape(shape.Type, name, shape)
	case "blob":
		w.EmitBlobShape(name, shape)
	case "string":
		w.EmitStringShape(name, shape)
	case "timestamp":
		w.EmitTimestampShape(name, shape)
	case "list", "set":
		w.EmitCollectionShape(shape.Type, name, shape)
	case "map":
		w.EmitMapShape(name, shape)
	case "structure":
		w.EmitStructureShape(name, shape)
	case "union":
		w.EmitUnionShape(name, shape)
	case "service":
		w.EmitServiceShape(name, shape)
	case "resource":
		w.EmitResourceShape(name, shape)
	case "operation":
		w.EmitOperationShape(name, shape)
	default:
		panic("fix: shape of type " + shape.Type)
	}
	w.Emit("\n")
}

func (w *IdlWriter) EmitDocumentation(doc, indent string) {
	if doc != "" {
		w.Emit("%s@documentation(%q)\n", indent, doc)
	}
}

func (w *IdlWriter) EmitDeprecated(dep *Deprecated, indent string) {
	if dep != nil {
		s := indent + "@deprecated"
		if dep.Message != "" {
			s = s + fmt.Sprintf("(message: %q", dep.Message)
		}
		if dep.Since != "" {
			if s == "@deprecated" {
				s = s + fmt.Sprintf("(since: %q)", dep.Since)
			} else {
				s = s + fmt.Sprintf(", since: %q)", dep.Since)
			}
		}
		w.Emit(s+"\n")
	}
}

func (w *IdlWriter) EmitBooleanTrait(b bool, s, indent string) {
	if b {
		w.Emit("%s@%s\n", indent, s)
	}
}

func (w *IdlWriter) EmitSimpleShape(shapeName, name string) {
	w.Emit("%s %s\n", shapeName, name)
}

func (w *IdlWriter) EmitBooleanShape(name string, shape *Shape) {
	w.EmitSimpleShape("boolean", name)
}

func (w *IdlWriter) EmitNumericShape(shapeName, name string, shape *Shape) {
	//traits for numbers
	w.EmitSimpleShape(shapeName, name)
}

func (w *IdlWriter) EmitStringShape(name string, shape *Shape) {
	if shape.Length != nil {
		l := shape.Length
		if l.Min != nil && l.Max != nil {
			w.Emit("@length(min: %d, max: %d)\n", *l.Min, *l.Max)
		} else if l.Max != nil {
			w.Emit("@length(max: %d)\n", *l.Max)
		} else if l.Min != nil {
			w.Emit("@length(min: %d)\n", *l.Min)
		}
	}
	if shape.Pattern != "" {
		w.Emit("@pattern(%q)\n", shape.Pattern)
	}
	//Enum
	w.Emit("%s %s\n", shape.Type, name)
}

func (w *IdlWriter) EmitTimestampShape(name string, shape *Shape) {
	w.Emit("timestamp %s\n", name)
}

func (w *IdlWriter) EmitBlobShape(name string, shape *Shape) {
	w.Emit("blob %s\n", name)
}

func (w *IdlWriter) EmitCollectionShape(shapeName, name string, shape *Shape) {
	w.Emit("%s %s {\n", shapeName, name)
	//traits for the collection
	//traits for member
	w.Emit("    member: %s\n", shape.Member.Target)
	w.Emit("}\n")
}

func (w *IdlWriter) EmitMapShape(name string, shape *Shape) {
	//todo: traits
	w.Emit("map %s {\n    key: %s,\n    value: %s\n}\n", name, shape.Key.Target, shape.Value.Target)
}

func (w *IdlWriter) EmitUnionShape(name string, shape *Shape) {
	w.Emit("union %s {\n", name)
	count := len(shape.Members)
	for fname, mem := range shape.Members {
		traits := ""
		if mem.Sensitive {
			traits = traits + "@sensitive "
		}
		//TODO other traits
		w.Emit("    %s%s: %s", traits, fname, mem.Target)
		count--
		if count > 0 {
			w.Emit(",\n")
		} else {
			w.Emit("\n")
		}		
	}
	w.Emit("}\n")
}

func (w *IdlWriter) EmitStructureShape(name string, shape *Shape) {
	w.Emit("structure %s {\n", name)
	count := len(shape.Members)
	indent := "    "
	for k, v := range shape.Members {
		w.EmitBooleanTrait(v.Sensitive, "sensitive", indent)
		w.EmitBooleanTrait(v.Required, "required", indent)
		w.EmitDocumentation(v.Documentation, indent)
		w.Emit("%s%s: %s", indent, k, v.Target)
		count--
		if count > 0 {
			w.Emit(",\n")
		} else {
			w.Emit("\n")
		}
	}
	w.Emit("}\n")
}

func listOfStrings(label string, lst []string) string {
	s := ""
	if len(lst) > 0 {
		s = label + ": ["
		for n, a := range lst {
			if n > 0 {
				s = s + ", "
			}
			s = s + fmt.Sprintf("%q", a)
		}
		s = s + "]"
	}
	return s
}

func (w *IdlWriter) EmitServiceShape(name string, shape *Shape) {
	if len(shape.Protocols) > 0 {
		s := "@protocols(["
		for n, p := range shape.Protocols {
			if n > 0 {
				s = s + ", "
			}
			s = s + fmt.Sprintf("{name: %q%s%s}", p.Name, listOfStrings(", auth", p.Auth), listOfStrings(", tags", p.Tags))
		}
		s = s + "])\n"
		w.Emit(s)
	}
	w.Emit("service %s {\n", name)
	w.Emit("    version: %q\n", shape.Version)
	if len(shape.Resources) > 0 {
		w.Emit("    %s\n", listOfStrings("resources", shape.Resources))
	}
	w.Emit("}\n")
}

func (w *IdlWriter) EmitResourceShape(name string, shape *Shape) {
	w.Emit("resource %s {\n", name)
	w.Emit(" ...fix me...\n")
	w.Emit("}\n")
}

func (w *IdlWriter) EmitOperationShape(name string, shape *Shape) {
	es := ""
	if len(shape.Errors) > 0 {
		for _, e := range shape.Errors {
			if es == "" {
				es = " errors ["
			} else {
				es = es + ", "
			}
			es = es + e
		}
		es = es + "]"
	}
	out := ""
	if shape.Output != "" {
		out = " -> " + shape.Output
	}
	w.Emit("operation %s(%s)%s%s\n", name, shape.Input, out, es)
}

func (w *IdlWriter) End() string {
	w.writer.Flush()
	return w.buf.String()
}

func (w *IdlWriter) stringDef(td *sadl.TypeDef) {
}


/*
	gen := &Generator{
		Generator: sadl.Generator{
			Model:  model,
			OutDir: outdir,
		},
	}
	pdir := filepath.Join(outdir)
	err := os.MkdirAll(pdir, 0755)
	if err != nil {
		gen.Err = err
	}
	return gen
*/
/*
model, err := smithy.FromSADL(schema)
	gen, err := model.NewGenerator(schema, "/tmp") //DO: remove the outdir arg, will print to stdout
	model, err := gen.Export()
*/

type Generator struct {
	sadl.Generator
}

func NewGenerator(model *sadl.Model, outdir string) *Generator {
	gen := &Generator{
		Generator: sadl.Generator{
			Model:  model,
			OutDir: outdir,
		},
	}
	pdir := filepath.Join(outdir)
	err := os.MkdirAll(pdir, 0755)
	if err != nil {
		gen.Err = err
	}
	return gen
}

func (gen *Generator) Export() (*Model, error) {
//	sadlModel := gen.Model
	smithyModel := &Model{
		Version: "0.4.0",
	}
	fmt.Println("build the model here")
	//build the model
//	oas.Info.Title = model.Comment //how we import it.
//	oas.Info.Version = model.Version
	return smithyModel, nil
}
