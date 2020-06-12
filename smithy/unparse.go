package smithy

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/boynton/sadl/util"
)

//
// Generate Smithy IDL to describe the Smithy model
//
func (ast *AST) IDL(namespace string) string {
	ns, _, _ := ast.NamespaceAndServiceVersion()
	if namespace != "" {
		ns = namespace
	}
	w := &IdlWriter{
		namespace: ns,
	}

	w.Begin()
	//	w.Emit("$version: %q\n", ast.Version) //only if a version-specific feature is needed. Could be "1" or "1.0"
	emitted := make(map[string]bool, 0)
	for k, v := range ast.Metadata {
		w.Emit("metadata %s = %s", k, util.Pretty(v))
	}
	w.Emit("\nnamespace %s\n\n", ns)
	for nsk, shape := range ast.Shapes {
		shapeAbsName := strings.Split(nsk, "#")
		shapeNs := shapeAbsName[0]
		shapeName := shapeAbsName[1]
		if shapeNs == ns {
			//only decompile stuff in the main namespace. Standard Smithy toolings seems to introduce aws.api shapes, and we
			//can assume the smithy.api on every import/export
			if shape.Type == "service" {
				w.EmitServiceShape(shapeName, shape)
				break
			}
		}
	}
	for nsk, v := range ast.Shapes {
		lst := strings.Split(nsk, "#")
		if lst[0] == ns {
			k := lst[1]
			if v.Type == "operation" {
				w.EmitShape(k, v)
				emitted[k] = true
				ki := k + "Input"
				if vi, ok := ast.Shapes[ki]; ok { //FIX ME
					w.EmitShape(ki, vi)
					emitted[ki] = true
				}
				ko := k + "Output"
				if vo, ok := ast.Shapes[ns+"#"+ko]; ok {
					w.EmitShape(ko, vo)
					emitted[ko] = true
				}
			}
		}
	}
	for nsk, v := range ast.Shapes {
		lst := strings.Split(nsk, "#")
		k := lst[1]
		if !emitted[k] {
			w.EmitShape(k, v)
		}
	}
	return w.End()
}

type IdlWriter struct {
	buf       bytes.Buffer
	writer    *bufio.Writer
	namespace string
	name      string
}

func (w *IdlWriter) Begin() {
	w.buf.Reset()
	w.writer = bufio.NewWriter(&w.buf)
}

func (w *IdlWriter) Emit(format string, args ...interface{}) {
	w.writer.WriteString(fmt.Sprintf(format, args...))
}

func (w *IdlWriter) EmitShape(name string, shape *Shape) {
	switch strings.ToLower(shape.Type) {
	case "boolean":
		w.EmitBooleanShape(name, shape)
	case "byte", "short", "integer", "long", "float", "double", "bigInteger", "bigdecimal":
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
	case "resource":
		w.EmitResourceShape(name, shape)
	case "operation":
		w.EmitOperationShape(name, shape)
	case "service":
		/* handled up front */
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

func (w *IdlWriter) EmitLengthTrait(v interface{}, indent string) {
	l := asStruct(v)
	min := get(l, "min")
	max := get(l, "max")
	if min != nil && max != nil {
		w.Emit("@length(min: %d, max: %d)\n", asInt(min), asInt(max))
	} else if max != nil {
		w.Emit("@length(max: %d)\n", asInt(max))
	} else if min != nil {
		w.Emit("@length(min: %d)\n", asInt(min))
	}
}

func (w *IdlWriter) EmitRangeTrait(v interface{}, indent string) {
	l := asStruct(v)
	min := get(l, "min")
	max := get(l, "max")
	if min != nil && max != nil {
		w.Emit("@range(min: %d, max: %d)\n", asInt(min), asInt(max))
	} else if max != nil {
		w.Emit("@range(max: %d)\n", asInt(max))
	} else if min != nil {
		w.Emit("@range(min: %d)\n", asInt(min))
	}
}

func (w *IdlWriter) EmitEnumTrait(v interface{}, indent string) {
	en := asArray("enum")
	if len(en) > 0 {
		s := util.Pretty(en)
		w.Emit("@enum(%s)\n", s)
	}
}

func (w *IdlWriter) EmitDeprecatedTrait(v interface{}, indent string) {
	/*
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
	*/
	panic("fix me")
}

func (w *IdlWriter) EmitHttpTrait(rv interface{}, indent string) {
	var method, uri string
	code := 0
	switch v := rv.(type) {
	case *HttpTrait:
		method = v.Method
		uri = v.Uri
		code = v.Code
	case map[string]interface{}:
		method = getString(v, "method")
		uri = getString(v, "uri")
		code = getInt(v, "code")
	default:
		panic("What?!")
	}
	s := fmt.Sprintf("method: %q, uri: %q", method, uri)
	if code != 0 {
		s = s + fmt.Sprintf(", code: %d", code)
	}
	w.Emit("@http(%s)\n", s)
}

func (w *IdlWriter) EmitHttpErrorTrait(rv interface{}, indent string) {
	var status int
	switch v := rv.(type) {
	case int32:
		status = int(v)
	default:
		//		fmt.Printf("http error arg, expected an int32, found %s with type %s\n", rv, sadl.Kind(rv))
	}
	if status != 0 {
		w.Emit("@httpError(%d)\n", status)
	}
}

func (w *IdlWriter) EmitSimpleShape(shapeName, name string) {
	w.Emit("%s %s\n", shapeName, name)
}

func (w *IdlWriter) EmitBooleanShape(name string, shape *Shape) {
	w.EmitTraits(shape.Traits, "")
	w.EmitSimpleShape("boolean", name)
}

func (w *IdlWriter) EmitNumericShape(shapeName, name string, shape *Shape) {
	w.EmitTraits(shape.Traits, "")
	w.EmitSimpleShape(shapeName, name)
}

func (w *IdlWriter) EmitStringShape(name string, shape *Shape) {
	w.EmitTraits(shape.Traits, "")
	w.Emit("%s %s\n", shape.Type, name)
}

func (w *IdlWriter) EmitTimestampShape(name string, shape *Shape) {
	w.EmitTraits(shape.Traits, "")
	w.Emit("timestamp %s\n", name)
}

func (w *IdlWriter) EmitBlobShape(name string, shape *Shape) {
	w.EmitTraits(shape.Traits, "")
	w.Emit("blob %s\n", name)
}

func (w *IdlWriter) EmitCollectionShape(shapeName, name string, shape *Shape) {
	w.EmitTraits(shape.Traits, "")
	w.Emit("%s %s {\n", shapeName, name)
	w.Emit("    member: %s\n", w.stripLocalNamespace(shape.Member.Target))
	w.Emit("}\n")
}

func (w *IdlWriter) EmitMapShape(name string, shape *Shape) {
	w.EmitTraits(shape.Traits, "")
	w.Emit("map %s {\n    key: %s,\n    value: %s\n}\n", name, shape.Key.Target, shape.Value.Target)
}

func (w *IdlWriter) EmitUnionShape(name string, shape *Shape) {
	w.EmitTraits(shape.Traits, "")
	w.Emit("union %s {\n", name)
	count := len(shape.Members)
	for fname, mem := range shape.Members {
		w.EmitTraits(mem.Traits, "    ")
		w.Emit("    %s: %s", fname, w.stripLocalNamespace(mem.Target))
		count--
		if count > 0 {
			w.Emit(",\n")
		} else {
			w.Emit("\n")
		}
	}
	w.Emit("}\n")
}

func stripNamespace(trait string) string {
	n := strings.Index(trait, "#")
	if n < 0 {
		return trait
	}
	return trait[n+1:]
}

func (w *IdlWriter) stripLocalNamespace(name string) string {
	prefix := w.namespace + "#"
	if strings.HasPrefix(name, prefix) {
		return name[len(prefix):]
	}
	return name
}

func (w *IdlWriter) EmitTraits(traits map[string]interface{}, indent string) {
	//note: documentation has an alternate for ("///"+comment), but then must be before other traits.
	for k, v := range traits {
		switch k {
		case "smithy.api#sensitive", "smithy.api#required", "smithy.api#readonly", "smithy.api#idempotent":
			w.EmitBooleanTrait(asBool(v), stripNamespace(k), indent)
		case "smithy.api#documentation":
			w.EmitDocumentation(asString(v), indent)
		case "smithy.api#httpLabel", "smithy.api#httpPayload":
			w.EmitBooleanTrait(asBool(v), stripNamespace(k), indent)
		case "smithy.api#httpQuery", "smithy.api#httpHeader":
			w.EmitStringTrait(asString(v), stripNamespace(k), indent)
		case "smithy.api#deprecated":
			w.EmitDeprecatedTrait(v, indent)
		case "smithy.api#http":
			w.EmitHttpTrait(v, indent)
		case "smithy.api#httpError":
			w.EmitHttpErrorTrait(v, indent)
		case "smithy.api#length":
			w.EmitLengthTrait(v, indent)
		case "smithy.api#range":
			w.EmitRangeTrait(v, indent)
		case "smithy.api#enum":
			w.EmitEnumTrait(v, indent)
		case "smithy.api#pattern", "smithy.api#error":
			w.EmitStringTrait(asString(v), stripNamespace(k), indent)
		case "aws.protocols#restJson1":
			w.Emit("%s@%s\n", indent, k) //FIXME for the non-default attributes
		default:
			//fixme "smithy.api#paginated"
			panic("fix me: emit trait " + k)
		}
	}
}

func (w *IdlWriter) EmitStructureShape(name string, shape *Shape) {
	w.EmitTraits(shape.Traits, "")
	w.Emit("structure %s {\n", name)
	indent := "    "
	for k, v := range shape.Members {
		w.EmitTraits(v.Traits, indent)
		w.Emit("%s%s: %s,\n", indent, k, v.Target)
	}
	w.Emit("}\n")
}

func listOfShapeRefs(label string, format string, lst []*ShapeRef, absolute bool) string {
	s := ""
	if len(lst) > 0 {
		s = label + ": ["
		for n, a := range lst {
			if n > 0 {
				s = s + ", "
			}
			target := a.Target
			if !absolute {
				target = stripNamespace(target)
			}
			s = s + fmt.Sprintf(format, target)
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
	w.EmitTraits(shape.Traits, "")
	w.Emit("service %s {\n", name)
	w.Emit("    version: %q,\n", shape.Version)
	if len(shape.Operations) > 0 {
		w.Emit("    %s\n", listOfShapeRefs("operations", "%s", shape.Operations, false))
	}
	if len(shape.Resources) > 0 {
		w.Emit("    %s\n", listOfShapeRefs("resources", "%s", shape.Resources, false))
	}
	w.Emit("}\n\n")
}

func (w *IdlWriter) EmitResourceShape(name string, shape *Shape) {
	w.EmitTraits(shape.Traits, "")
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
			w.Emit("    %s\n", listOfShapeRefs("operations", "%s", shape.Operations, true))
		}
		if len(shape.CollectionOperations) > 0 {
			w.Emit("    %s\n", listOfShapeRefs("collectionOperations", "%s", shape.CollectionOperations, true))
		}
	}
	w.Emit("}\n")
}

func (w *IdlWriter) EmitOperationShape(name string, shape *Shape) {
	w.EmitTraits(shape.Traits, "")
	w.Emit("operation %s {\n", name)
	if shape.Input != nil {
		w.Emit("    input: %s,\n", stripNamespace(shape.Input.Target))
	}
	if shape.Output != nil {
		w.Emit("    output: %s,\n", stripNamespace(shape.Output.Target))
	}
	if len(shape.Errors) > 0 {
		w.Emit("    %s,\n", listOfShapeRefs("errors", "%s", shape.Errors, false))
	}
	w.Emit("}\n")
}

func (w *IdlWriter) End() string {
	w.writer.Flush()
	return w.buf.String()
}
