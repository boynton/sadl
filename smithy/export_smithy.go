package smithy

import (
	"fmt"
	"strings"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/util"
)

func Export(model *sadl.Model, out string, conf map[string]interface{}, exportAst bool) error {
	ns := model.Namespace
	if v, ok := conf["namespace"]; ok {
		if s, ok := v.(string); ok {
			ns = s
		}
	}
	if ns == "" {
		ns = "example"
	}
	ast, err := FromSADL(model, ns)
	if err != nil {
		return fmt.Errorf("sadl2smithy: Cannot convert SADL to Smithy: %v\n", err)
	}
	if exportAst {
		fmt.Println(util.Pretty(ast))
	} else {
		fmt.Print(ast.IDL(ns))
	}
	return nil
}

func AllTypeRefs(model *sadl.Model) map[string]bool {
	refs := make(map[string]bool, 0)
	for _, td := range model.Types {
		noteTypeRefs(refs, model, &td.TypeSpec)
	}
	return refs
}
func noteTypeRefs(refs map[string]bool, model *sadl.Model, ts *sadl.TypeSpec) {
	switch ts.Type {
	case "Bool", "Int8", "Int16", "Int32", "Int64", "Float32", "Float64", "Decimal", "Bytes", "String", "Timestamp", "UUID":
		//primitive type references.
		refs[ts.Type] = true
	case "Enum":
		//always derived, never naked
	case "Array":
		td := model.FindType(ts.Items)
		noteTypeRefs(refs, model, &td.TypeSpec)
	case "Map":
		td := model.FindType(ts.Items)
		noteTypeRefs(refs, model, &td.TypeSpec)
		td = model.FindType(ts.Keys)
		noteTypeRefs(refs, model, &td.TypeSpec)
	case "Struct":
		if len(ts.Fields) == 0 {
			refs["Struct"] = true
		} else {
			for _, fd := range ts.Fields {
				noteTypeRefs(refs, model, &fd.TypeSpec)
			}
		}
	case "UnitValue":
		td := model.FindType(ts.Unit)
		noteTypeRefs(refs, model, &td.TypeSpec)
		td = model.FindType(ts.Value)
		noteTypeRefs(refs, model, &td.TypeSpec)
	case "Union":
		for _, variant := range ts.Variants {
			td := model.FindType(variant.Type)
			noteTypeRefs(refs, model, &td.TypeSpec)
		}
	case "Any":
		//hmm.
	}
}

func FromSADL(model *sadl.Model, ns string) (*AST, error) {
	ast := &AST{
		Version:  SmithyVersion,
		Shapes:   make(map[string]*Shape, 0),
		Metadata: make(map[string]interface{}, 0),
	}
	//	ast.Metadata["imported_from_sadl_version"] = sadl.Version
	ast.Metadata["name"] = model.Name

	refs := AllTypeRefs(model)
	if _, ok := refs["UUID"]; ok {
		ast.Shapes[ns+"#UUID"] = uuidShape()
	}

	for _, td := range model.Types {
		err := defineShapeFromTypeSpec(model, ns, ast.Shapes, &td.TypeSpec, td.Name, td.Comment, td.Annotations)
		if err != nil {
			return nil, err
		}
	}
	var ops []*ShapeRef
	prefix := ns + "#"
	for _, hd := range model.Http {
		expectedCode := 200
		if hd.Expected != nil {
			expectedCode = int(hd.Expected.Status)
		}
		path := hd.Path
		n := strings.Index(path, "?")
		if n >= 0 {
			path = path[:n]
		}
		name := capitalize(hd.Name)
		if name == "" {
			name = capitalize(strings.ToLower(hd.Method)) + "Something" //fix!
			fmt.Println("FIX:", name)
		}
		shape := Shape{
			Type: "operation",
		}
		ensureShapeTraits(&shape)["smithy.api#documentation"] = hd.Comment
		ensureShapeTraits(&shape)["smithy.api#http"] = httpTrait(path, hd.Method, expectedCode)
		switch hd.Method {
		case "GET":
			ensureShapeTraits(&shape)["smithy.api#readonly"] = true
		case "PUT", "DELETE":
			ensureShapeTraits(&shape)["smithy.api#idempotent"] = true
		}

		//if we have any inputs, define this
		if len(hd.Inputs) > 0 {
			shape.Input = &ShapeRef{Target: prefix + name + "Input"}
			inShape := Shape{
				Type:    "structure",
				Members: make(map[string]*Member, 0),
			}
			//		inShape.Documentation = "[autogenerated for operation '" + name + "']"
			for _, in := range hd.Inputs {
				mem := &Member{
					Target: typeReferenceByName(in.Type),
				}
				if in.Path {
					ensureMemberTraits(mem)["smithy.api#httpLabel"] = true
					ensureMemberTraits(mem)["smithy.api#required"] = true
				} else if in.Query != "" {
					ensureMemberTraits(mem)["smithy.api#httpQuery"] = in.Query
				} else if in.Header != "" {
					ensureMemberTraits(mem)["smithy.api#httpHeader"] = in.Header
				} else {
					ensureMemberTraits(mem)["smithy.api#httpPayload"] = true
				}
				inShape.Members[in.Name] = mem
			}
			ast.Shapes[shape.Input.Target] = &inShape
		}

		if len(hd.Expected.Outputs) > 0 {
			//if we have any outputs, define this
			shape.Output = &ShapeRef{Target: prefix + name + "Output"}
			outShape := Shape{
				Type:    "structure",
				Members: make(map[string]*Member, 0),
			}
			//		outShape.Documentation = "[autogenerated for operation '" + name + "']"
			for _, out := range hd.Expected.Outputs {
				mem := &Member{
					Target: typeReferenceByName(out.Type),
				}
				if out.Header != "" {
					ensureMemberTraits(mem)["smithy.api#httpHeader"] = out.Header
				} else {
					ensureMemberTraits(mem)["smithy.api#httpPayload"] = true
				}
				outShape.Members[out.Name] = mem
			}
			ast.Shapes[shape.Output.Target] = &outShape
		}

		//if we have any exceptions, define them
		if len(hd.Exceptions) > 0 {
			for _, e := range hd.Exceptions {
				em := &ShapeRef{Target: ns + "#" + e.Type}
				shape.Errors = append(shape.Errors, em)
				if tmp, ok := ast.Shapes[em.Target]; ok {
					ensureShapeTraits(tmp)["smithy.api#httpError"] = e.Status
					ensureShapeTraits(tmp)["smithy.api#error"] = httpErrorCategory(e.Status)
				} else {
					return nil, fmt.Errorf("Cannot find shape for error declaration type %q", e.Type)
				}
			}
		}
		ast.Shapes[prefix+name] = &shape
		ops = append(ops, &ShapeRef{
			Target: prefix + name,
		})

	}
	if len(ops) > 0 {
		service := &Shape{
			Type:       "service",
			Version:    model.Version,
			Operations: ops,
		}
		if service.Version == "" {
			service.Version = "0.0" //Smithy requires a version on a service
		}
		if model.Comment != "" {
			ensureShapeTraits(service)["smithy.api#documentation"] = model.Comment
		}
		ast.Shapes[prefix+model.Name] = service
	}
	return ast, nil
}

func typeReference(ts *sadl.TypeSpec) string {
	return typeReferenceByName(ts.Type)
}

func typeReferenceByName(name string) string {
	switch name {
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
		return "UUID" //depends on the emitted UUID definition, since Smithy does not have UUID types
	case "Bytes":
		return "Blob"
	case "String":
		return "String"
	case "Array":
		return "List"
	case "Map":
		return "Map"
	case "Struct":
		return "Document" //naked struct only.
	default:
		return name
	}
}

func listTypeReference(model *sadl.Model, ns string, shapes map[string]*Shape, prefix string, fd *sadl.StructFieldDef) string {
	ftype := capitalize(prefix) + capitalize(fd.Name)
	td := model.FindType(ftype)
	if td != nil {
		fmt.Printf("Inline defs not allowed, synthesize %q to refer to: %s\n", ftype, util.Pretty(fd))
		panic("Already have one with that name!!!")
	}
	shape := Shape{
		Type: "list",
	}
	shape.Member = &Member{
		Target: typeReferenceByName(fd.Items),
	}
	ensureShapeTraits(&shape)["smithy.api#documentation"] = "[autogenerated for field '" + fd.Name + "' in struct '" + prefix + "']"
	shapes[ns+"#"+ftype] = &shape
	return ftype
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[0:1]) + s[1:]
}

func ensureShapeTraits(shape *Shape) map[string]interface{} {
	if shape.Traits == nil {
		shape.Traits = make(map[string]interface{}, 0)
	}
	return shape.Traits
}

func ensureMemberTraits(member *Member) map[string]interface{} {
	if member.Traits == nil {
		member.Traits = make(map[string]interface{}, 0)
	}
	return member.Traits
}

func defineShapeFromTypeSpec(model *sadl.Model, ns string, shapes map[string]*Shape, ts *sadl.TypeSpec, name string, comment string, annos map[string]string) error {
	var shape Shape
	switch ts.Type {
	case "Int8", "Int16", "Int32", "Int64", "Float32", "Float64", "Decimal":
		shape = shapeFromNumber(ts)
	case "String":
		shape = shapeFromString(ts)
	case "Enum":
		shape = shapeFromEnum(ts)
	case "Struct":
		shape = shapeFromStruct(model, ns, shapes, name, ts)
	case "Array":
		shape = shapeFromArray(model, ns, shapes, name, ts)
	case "Union":
		shape = shapeFromUnion(model, ns, shapes, name, ts)
	default:
		fmt.Println("So far:", util.Pretty(model))
		panic("handle this type:" + util.Pretty(ts))
	}
	if comment != "" {
		ensureShapeTraits(&shape)["smithy.api#documentation"] = comment
	}
	if annos != nil {
		for k, v := range annos {
			switch k {
			case "x_sensitive":
				ensureShapeTraits(&shape)["smithy.api#sensitive"] = true
			case "x_deprecated":
				dep := make(map[string]interface{}, 0)
				if v != "" {
					n := strings.Index(v, "|")
					if n >= 0 {
						dep["since"] = v[:n]
						dep["message"] = v[n+1:]
					} else {
						dep["message"] = v
					}
					ensureShapeTraits(&shape)["smithy.api#deprecated"] = dep
				}
			}
		}
	}
	shapes[ns+"#"+name] = &shape
	return nil
}

func shapeFromArray(model *sadl.Model, ns string, shapes map[string]*Shape, tname string, ts *sadl.TypeSpec) Shape {
	member := Member{
		Target: EnsureNamespaced(ns, typeReferenceByName(ts.Items)),
	}
	shape := Shape{
		Type:   "list",
		Member: &member,
	}
	l := lengthTrait(ts.MinSize, ts.MaxSize)
	if l != nil {
		ensureShapeTraits(&shape)["smithy.api#length"] = l
	}
	return shape
}

func shapeFromStruct(model *sadl.Model, ns string, shapes map[string]*Shape, tname string, ts *sadl.TypeSpec) Shape {
	shape := Shape{
		Type: "structure",
	}
	members := make(map[string]*Member, 0)
	for _, fd := range ts.Fields {
		ftype := typeReference(&fd.TypeSpec)
		switch ftype {
		case "List":
			ftype = listTypeReference(model, ns, shapes, tname, fd)
		}
		member := &Member{
			Target: ftype,
		}
		if fd.Required {
			ensureMemberTraits(member)["smithy.api#required"] = true
		}
		members[fd.Name] = member
	}
	shape.Members = members
	return shape
}

func shapeFromUnion(model *sadl.Model, ns string, shapes map[string]*Shape, tname string, ts *sadl.TypeSpec) Shape {
	shape := Shape{
		Type: "union",
	}
	members := make(map[string]*Member, 0)
	for _, vd := range ts.Variants { //todo: modify SADL to make unions more like structs
		//		fd := model.FindType(vtype.Type)
		//		ftype := typeReference(&fd.TypeSpec)
		member := &Member{
			Target: EnsureNamespaced(ns, vd.Type),
		}
		ensureMemberTraits(member)["smithy.api#documentation"] = vd.Comment
		members[vd.Name] = member
	}
	shape.Members = members
	return shape
}

func shapeFromString(ts *sadl.TypeSpec) Shape {
	shape := Shape{
		Type: "string",
	}
	l := lengthTrait(ts.MinSize, ts.MaxSize)
	if l != nil {
		ensureShapeTraits(&shape)["smithy.api#length"] = l
	}
	if ts.Pattern != "" {
		ensureShapeTraits(&shape)["smithy.api#pattern"] = ts.Pattern
	}
	if len(ts.Values) > 0 {
		e := make(EnumTrait, 0)
		for _, s := range ts.Values {
			ei := &EnumTraitItem{
				Value: s,
				//maybe preserve an annotation for constant name for smithy completeness?
			}
			//TODO: el.Annotations -> if contains x_tags, then expand to item.Tags
			e = append(e, ei)
		}
		ensureShapeTraits(&shape)["smithy.api#enum"] = e
	}
	return shape
}

func shapeFromNumber(ts *sadl.TypeSpec) Shape {
	shape := Shape{}
	switch ts.Type {
	case "Int8":
		shape.Type = "byte"
	case "Int16":
		shape.Type = "short"
	case "Int32":
		shape.Type = "integer"
	case "Int64":
		shape.Type = "long"
	case "Float32":
		shape.Type = "float"
	case "Float64":
		shape.Type = "double"
	case "Decimal":
		shape.Type = "bigDecimal"
	}
	if ts.Min != nil || ts.Max != nil {
		ensureShapeTraits(&shape)["smithy.api#range"] = rangeTrait(ts.Min, ts.Max)
	}
	return shape
}

func uuidShape() *Shape {
	shape := Shape{
		Type: "string",
	}
	ensureShapeTraits(&shape)["smithy.api#pattern"] = "([a-f0-9]{8}(-[a-f0-9]{4}){4}[a-f0-9]{8})"
	return &shape
}

func shapeFromEnum(ts *sadl.TypeSpec) Shape {
	shape := Shape{
		Type: "string",
	}
	//for sadl, enum values *are* the symbols, so the name must be set to match the key
	//note that this same form can work with values, where the name is optional but the key is the actual value
	items := make(EnumTrait, 0)
	for _, el := range ts.Elements {
		item := &EnumTraitItem{
			Value:         el.Symbol, //required. Q: how to get parity in SADL with differing value and names? In SADL, they are the same
			Name:          el.Symbol, //optional, the symbolic constant name, for codegen.
			Documentation: el.Comment,
		}
		//TODO: el.Annotations -> if contains x_tags, then expand to item.Tags
		items = append(items, item)
	}
	ensureShapeTraits(&shape)["smithy.api#enum"] = items
	return shape
}

func httpTrait(path, method string, code int) map[string]interface{} {
	t := make(map[string]interface{}, 0)
	t["uri"] = path
	t["method"] = method
	if code != 0 {
		t["code"] = code
	}
	return t
}

func lengthTrait(min *int64, max *int64) map[string]interface{} {
	if min == nil && max == nil {
		return nil
	}
	l := make(map[string]interface{}, 0)
	if min != nil {
		l["min"] = *min
	}
	if max != nil {
		l["max"] = *max
	}
	return l
}

func rangeTrait(min *sadl.Decimal, max *sadl.Decimal) map[string]interface{} {
	if min == nil && max == nil {
		return nil
	}
	l := make(map[string]interface{}, 0)
	if min != nil {
		l["min"] = *min
	}
	if max != nil {
		l["max"] = *max
	}
	return l
}

func httpErrorCategory(status int32) string {
	//Smithy 1.0 only specifies "client" and "server", with apparently no way to handle other status codes
	if status < 200 {
		return "informational"
	}
	if status < 300 {
		return "success"
	}
	if status < 400 {
		return "redirect"
	}
	if status >= 500 {
		return "server"
	}
	return "client"
}
