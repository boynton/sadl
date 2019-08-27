package smithy

import(
	"bytes"
	"bufio"
	"fmt"
)

//
// Generate Smithy IDL to describe the Smithy model
//
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
	indent := "    "
	for k, v := range shape.Members {
		w.EmitBooleanTrait(v.Sensitive, "sensitive", indent)
		w.EmitBooleanTrait(v.Required, "required", indent)
		w.EmitDocumentation(v.Documentation, indent)
		w.Emit("%s%s: %s,\n", indent, k, v.Target)
	}
	w.Emit("}\n")
}

func listOfStrings(label string, format string, lst []string) string {
	s := ""
	if len(lst) > 0 {
		s = label + ": ["
		for n, a := range lst {
			if n > 0 {
				s = s + ", "
			}
			s = s + fmt.Sprintf(format, a)
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
			s = s + fmt.Sprintf("{name: %q%s%s}", p.Name, listOfStrings(", auth", "%q", p.Auth), listOfStrings(", tags", "%q", p.Tags))
		}
		s = s + "])\n"
		w.Emit(s)
	}
	w.Emit("service %s {\n", name)
	w.Emit("    version: %q\n", shape.Version)
	if len(shape.Resources) > 0 {
		w.Emit("    %s\n", listOfStrings("resources", "%s", shape.Resources))
	}
	w.Emit("}\n")
}

func (w *IdlWriter) EmitResourceShape(name string, shape *Shape) {
	w.Emit("resource %s {\n", name)
	if len(shape.Identifiers) > 0 {
		w.Emit("    identifiers: {\n")
		for k, v := range shape.Identifiers {
			w.Emit("        %s: %s,\n", k, v)
		}
		w.Emit("    }\n")
		if shape.Create != "" {
			w.Emit("    create: %s\n", shape.Create)
		}
		if shape.Put != "" {
			w.Emit("    put: %s\n", shape.Put)
		}
		if shape.Read != "" {
			w.Emit("    read: %s\n", shape.Read)
		}
		if shape.Update != "" {
			w.Emit("    update: %s\n", shape.Update)
		}
		if shape.Delete != "" {
			w.Emit("    delete: %s\n", shape.Delete)
		}
		if shape.List != "" {
			w.Emit("    list: %s\n", shape.List)
		}
		if len(shape.Operations) > 0 {
			w.Emit("    %s\n", listOfStrings("operations", "%s", shape.Operations))
		}
		if len(shape.CollectionOperations) > 0 {
			w.Emit("    %s\n", listOfStrings("collectionOperations", "%s", shape.CollectionOperations))
		}
	}
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

