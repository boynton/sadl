package smithy

import(
	"bytes"
	"bufio"
	"fmt"
	"strings"
//	"github.com/boynton/sadl"
)

//
// Generate Smithy IDL to describe the Smithy model
//
func (model *Model) IDL() string {
	w := &IdlWriter{}

	w.Begin()
	w.Emit("$version: %q\n", model.Version)
	emitted := make(map[string]bool, 0)
	lastNs := ""
	for nsk, v := range model.Shapes {
		lst := strings.Split(nsk, "#")
		ns := lst[0]
		k := lst[1]
		if ns != lastNs {
			w.Emit("\nnamespace %s\n\n", ns)
			lastNs = ns
		}
		if v.Type == "operation" {
			w.EmitShape(k, v)
			emitted[k] = true
			ki := k+"Input"
			if vi, ok := model.Shapes[ki]; ok { //FIX ME
				w.EmitShape(ki, vi)
				emitted[ki] = true
			}
			ko := k+"Output"
			if vo, ok := model.Shapes[ns+"#"+ko]; ok {
				w.EmitShape(ko, vo)
				emitted[ko] = true
			}
		}
	}
	for nsk, v := range model.Shapes {
		lst := strings.Split(nsk, "#")
		k := lst[1]
		if !emitted[k] {
			w.EmitShape(k, v)
		}
	}
	//out of band traits here
//		for k, v := range namespace.Traits {
//			fmt.Println("FIX ME trait", k, v)
//		}
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

func documentation(shape *Shape) string {
	if shape.Traits != nil {
		return shape.Traits.Documentation
	}
	return ""
}

func nonNilTraits(traits *Traits) *Traits {
	if traits == nil {
		return &Traits{}
	}
	return traits
}

func (w *IdlWriter) EmitShape(name string, shape *Shape) {
	traits := nonNilTraits(shape.Traits)
	w.EmitDocumentation(traits.Documentation, "")	
	w.EmitDeprecated(traits.Deprecated, "")
	w.EmitBooleanTrait(traits.Sensitive, "sensitive", "")
	w.EmitBooleanTrait(traits.ReadOnly, "readonly", "")
	w.EmitBooleanTrait(traits.Idempotent, "idempotent", "")
	
	if traits.Http != nil {
		s := ""
		if traits.Http.Method != "" {
			s = fmt.Sprintf("method: %q", traits.Http.Method)
		}
		if traits.Http.Uri != "" {
			if s != "" {
				s = s + ", "
			}
			s = s + fmt.Sprintf("uri: %q", traits.Http.Uri)
		}
		if traits.Http.Code != 0 {
			if s != "" {
				s = s + ", "
			}
			s = s + fmt.Sprintf("code: %d", traits.Http.Code)
		}
		w.Emit("@http(%s)\n", s)
	}
	if traits.HttpError != 0 {
		//note: @retryable
		w.Emit("@error(\"client\")\n@httpError(%d)\n", traits.HttpError)
	}
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

func (w *IdlWriter) EmitBooleanTrait(b bool, tname, indent string) {
	if b {
		w.Emit("%s@%s\n", indent, tname)
	}
}

func (w *IdlWriter) EmitStringTrait(v, tname, indent string) {
	if v != "" {
		if v == "-" { //hack
			w.Emit("%s@%s\n", indent, tname)
		} else {
			w.Emit("%s@%s(%q)\n", indent, tname, v)
		}
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
	tr := nonNilTraits(shape.Traits)
	if tr.Length != nil {
		l := tr.Length
		if l.Min != nil && l.Max != nil {
			w.Emit("@length(min: %d, max: %d)\n", *l.Min, *l.Max)
		} else if l.Max != nil {
			w.Emit("@length(max: %d)\n", *l.Max)
		} else if l.Min != nil {
			w.Emit("@length(min: %d)\n", *l.Min)
		}
	}
	if tr.Pattern != "" {
		w.Emit("@pattern(%q)\n", tr.Pattern)
	}
	if tr.Enum != nil {
		s := ""
		for k, item := range tr.Enum {
			if s != "" {
				s = s + ", "
			}
			s = s + fmt.Sprintf("%s: {name: %q}", k, item.Name)
		}
		w.Emit("@enum(%s)\n", s)
	}
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
		if nonNilTraits(mem.Traits).Sensitive {
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
		traits := nonNilTraits(v.Traits)
		w.EmitBooleanTrait(traits.Sensitive, "sensitive", indent)
		w.EmitBooleanTrait(traits.Required, "required", indent)
		w.EmitBooleanTrait(traits.HttpLabel, "httpLabel", indent)
		w.EmitStringTrait(traits.HttpQuery, "httpQuery", indent)
		w.EmitStringTrait(traits.HttpHeader, "httpHeader", indent)
		w.EmitBooleanTrait(traits.HttpPayload, "httpPayload", indent)
		w.EmitDocumentation(traits.Documentation, indent)
		w.Emit("%s%s: %s,\n", indent, k, v.Target)
	}
	w.Emit("}\n")
}

func listOfMembers(label string, format string, lst []*Member) string {
	s := ""
	if len(lst) > 0 {
		s = label + ": ["
		for n, a := range lst {
			if n > 0 {
				s = s + ", "
			}
			s = s + fmt.Sprintf(format, a.Target)
		}
		s = s + "]"
	}
	return s
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
	traits := nonNilTraits(shape.Traits)
	if len(traits.Protocols) > 0 {
		s := "@protocols(["
		for n, p := range traits.Protocols {
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
		w.Emit("    %s\n", listOfMembers("resources", "%s", shape.Resources))
	}
	w.Emit("}\n")
}

func (w *IdlWriter) EmitResourceShape(name string, shape *Shape) {
	w.Emit("resource %s {\n", name)
	if len(shape.Identifiers) > 0 {
		w.Emit("    identifiers: {\n")
		for k, v := range shape.Identifiers {
			w.Emit("        %s: %s,\n", k, v.Target) //fixme
		}
		w.Emit("    }\n")
		if shape.Create != nil {
			w.Emit("    create: %v\n", shape.Create)
		}
		if shape.Put != nil {
			w.Emit("    put: %v\n", shape.Put)
		}
		if shape.Read != nil {
			w.Emit("    read: %v\n", shape.Read)
		}
		if shape.Update != nil {
			w.Emit("    update: %v\n", shape.Update)
		}
		if shape.Delete != nil {
			w.Emit("    delete: %v\n", shape.Delete)
		}
		if shape.List != nil {
			w.Emit("    list: %v\n", shape.List)
		}
		if len(shape.Operations) > 0 {
			w.Emit("    %s\n", listOfMembers("operations", "%s", shape.Operations))
		}
		if len(shape.CollectionOperations) > 0 {
			w.Emit("    %s\n", listOfMembers("collectionOperations", "%s", shape.CollectionOperations))
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
			es = es + e.Target
		}
		es = es + "]"
	}
	out := ""
	if shape.Output != nil {
		out = " -> " + shape.Output.Target
	}
	w.Emit("operation %s(%s)%s%s\n", name, shape.Input.Target, out, es)
}

func (w *IdlWriter) End() string {
	w.writer.Flush()
	return w.buf.String()
}

