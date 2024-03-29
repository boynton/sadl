package smithy

import (
	"fmt"
	"strings"

	"github.com/boynton/data"
	"github.com/boynton/sadl"
	smithylib "github.com/boynton/smithy"
)

var inputSuffix = "Request"
var outputSuffix = "Response"

func Export(model *sadl.Model, out string, conf *sadl.Data, exportAst bool) error {
	if conf.GetBool("useInputOutput") {
		inputSuffix = "Input"
		outputSuffix = "Output"
	}
	ns := model.Namespace
	ns2 := conf.GetString("namespace")
	if ns2 != "" {
		ns = ns2
	}
	if ns == "" {
		ns = "example"
	}
	ast, err := FromSADL(model, ns)
	if err != nil {
		return fmt.Errorf("sadl2smithy: Cannot convert SADL to Smithy: %v\n", err)
	}
	if exportAst {
		fmt.Println(sadl.Pretty(ast))
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

func FromSADL(model *sadl.Model, ns string) (*smithylib.AST, error) {
	ast := &smithylib.AST{
		Smithy:   "2",
		Metadata: data.NewObject(),
		Shapes:   smithylib.NewShapes(),
	}
	//	ast.Metadata["imported_from_sadl_version"] = sadl.Version
	//	ast.Metadata["name"] = model.Name
	if model.Base != "" {
		ast.Metadata.Put("base", model.Base)
	}

	for _, td := range model.Types {
		err := defineShapeFromTypeSpec(model, ns, ast.Shapes, &td.TypeSpec, td.Name, td.Comment, td.Annotations)
		if err != nil {
			return nil, err
		}
	}
	var ops []*smithylib.ShapeRef
	prefix := ns + "#"
	for _, hd := range model.Http {
		expectedCode := 200
		if hd.Expected != nil {
			expectedCode = int(hd.Expected.Status)
		}
		path := hd.Path
		if model.Base != "" {
			path = model.Base + path
		}
		n := strings.Index(path, "?")
		if n >= 0 {
			path = path[:n]
		}
		name := capitalize(hd.Name)
		if name == "" {
			name = capitalize(strings.ToLower(hd.Method)) + "Something" //fix!
			fmt.Println("FIX:", name)
		}
		shape := smithylib.Shape{
			Type: "operation",
		}
		ensureShapeTraits(&shape).Put("smithy.api#documentation", hd.Comment)
		ensureShapeTraits(&shape).Put("smithy.api#http", httpTrait(path, hd.Method, expectedCode))
		if hd.Annotations != nil {
			if tags, ok := hd.Annotations["x_tags"]; ok {
				ensureShapeTraits(&shape).Put("smithy.api#tags", strings.Split(tags, ","))
			}
			if pagi, ok := hd.Annotations["x_paginated"]; ok {
				ensureShapeTraits(&shape).Put("smithy.api#paginated", paginatedTrait(pagi))
			}
		}
		switch hd.Method {
		case "GET":
			ensureShapeTraits(&shape).Put("smithy.api#readonly", true)
		case "PUT", "DELETE":
			ensureShapeTraits(&shape).Put("smithy.api#idempotent", true)
		}

		//if we have any inputs, define this
		if len(hd.Inputs) > 0 {
			shape.Input = &smithylib.ShapeRef{Target: prefix + name + inputSuffix}
			inShape := smithylib.Shape{
				Type:    "structure",
				Members: smithylib.NewMembers(),
			}
			ensureShapeTraits(&inShape).Put("smithy.api#input", true)
			for _, in := range hd.Inputs {
				mem := &smithylib.Member{
					Target: typeReferenceByName(ns, in.Type),
				}
				if in.Path {
					ensureMemberTraits(mem).Put("smithy.api#httpLabel", true)
					ensureMemberTraits(mem).Put("smithy.api#required", true)
				} else if in.Query != "" {
					ensureMemberTraits(mem).Put("smithy.api#httpQuery", in.Query)
				} else if in.Header != "" {
					ensureMemberTraits(mem).Put("smithy.api#httpHeader", in.Header)
				} else {
					ensureMemberTraits(mem).Put("smithy.api#httpPayload", true)
				}
				inShape.Members.Put(in.Name, mem)
			}
			ast.Shapes.Put(shape.Input.Target, &inShape)
		}

		if len(hd.Expected.Outputs) > 0 {
			//if we have any outputs, define this
			shape.Output = &smithylib.ShapeRef{Target: prefix + name + outputSuffix}
			outShape := smithylib.Shape{
				Type:    "structure",
				Members: smithylib.NewMembers(),
			}
			ensureShapeTraits(&outShape).Put("smithy.api#output", true)
			for _, out := range hd.Expected.Outputs {
				mem := &smithylib.Member{
					Target: typeReferenceByName(ns, out.Type),
				}
				if out.Header != "" {
					ensureMemberTraits(mem).Put("smithy.api#httpHeader", out.Header)
				} else {
					ensureMemberTraits(mem).Put("smithy.api#httpPayload", true)
				}
				outShape.Members.Put(out.Name, mem)
			}
			ast.Shapes.Put(shape.Output.Target, &outShape)
		}

		//if we have any exceptions, define them
		if len(hd.Exceptions) > 0 {
			for _, e := range hd.Exceptions {
				//so, these are *wrappers* for error resources in smithy
				//i.e. I need to generate *another* type with a field marked with @httpPayload attribute
				//So, this is the "NotFoundErrorContent" type that smithy->openapi produces. If I specify it,
				//then smithy reference tooling does not generate anything else.
				//typical pattern to use:
				// type Error Struct { message String }
				// type NotFoundError { error Error } //this is the wrapper for the operation
				//this produces (assuming NotFoundError is referenced in an 'except' response):
				// structure Error { message: String }
				// @error("client")
				// @httpError(404)
				// structure NotFoundError {
				//   @httpPayload
				//   error: Error
				// }
				//to do: modify the sadl model to support headers in exception responses
				em := &smithylib.ShapeRef{Target: ns + "#" + e.Type}
				shape.Errors = append(shape.Errors, em)
				if tmp := ast.Shapes.Get(em.Target); tmp != nil {
					ensureShapeTraits(tmp).Put("smithy.api#httpError", e.Status)
					ensureShapeTraits(tmp).Put("smithy.api#error", httpErrorCategory(e.Status))
					//check that the shape has a single member that is a struct. If so, mark it as payload
					if tmp.Members != nil && tmp.Members.Length() == 1 {
						for _, k := range tmp.Members.Keys() {
							m := tmp.Members.Get(k)
							ensureMemberTraits(m).Put("smithy.api#httpPayload", true)
						}
					}
				} else {
					return nil, fmt.Errorf("Cannot find shape for error declaration type %q", e.Type)
				}
			}
		}
		ast.Shapes.Put(prefix+name, &shape)
		ops = append(ops, &smithylib.ShapeRef{
			Target: prefix + name,
		})

	}
	for _, od := range model.Operations {
		shape := smithylib.Shape{
			Type: "operation",
		}
		name := capitalize(od.Name)
		if len(od.Inputs) > 0 {
			shape.Input = &smithylib.ShapeRef{Target: prefix + name + inputSuffix}
			inShape := smithylib.Shape{
				Type:    "structure",
				Members: smithylib.NewMembers(),
			}
			//		inShape.Documentation = "[autogenerated for operation '" + name + "']"
			for _, in := range od.Inputs {
				mem := &smithylib.Member{
					Target: typeReferenceByName(ns, in.Type),
				}
				inShape.Members.Put(in.Name, mem)
			}
			ast.Shapes.Put(shape.Input.Target, &inShape)
		}
		if len(od.Outputs) > 0 {
			shape.Output = &smithylib.ShapeRef{Target: prefix + name + outputSuffix}
			outShape := smithylib.Shape{
				Type:    "structure",
				Members: smithylib.NewMembers(),
			}
			//		outShape.Documentation = "[autogenerated for operation '" + name + "']"
			for _, out := range od.Outputs {
				mem := &smithylib.Member{
					Target: typeReferenceByName(ns, out.Type),
				}
				outShape.Members.Put(out.Name, mem)
			}
			ast.Shapes.Put(shape.Output.Target, &outShape)
		}
		if len(od.Exceptions) > 0 {
			var excs []string
			for _, exc := range od.Exceptions {
				excs = append(excs, exc)
			}
			//?
		}

		ensureShapeTraits(&shape).Put("smithy.api#documentation", od.Comment)
		ast.Shapes.Put(prefix+name, &shape)
		ops = append(ops, &smithylib.ShapeRef{
			Target: prefix + name,
		})
	}

	if len(ops) > 0 {
		service := &smithylib.Shape{
			Type:       "service",
			Version:    model.Version,
			Operations: ops,
		}
		if service.Version == "" {
			service.Version = "0.0" //Smithy requires a version on a service
		}
		if model.Comment != "" {
			ensureShapeTraits(service).Put("smithy.api#documentation", model.Comment)
		}
		serviceName := sadl.Capitalize(model.Name)
		ast.Shapes.Put(prefix+serviceName, service)
	}
	if len(model.Examples) > 0 {
		examplesFromSADL(ns, ast, model)
	}
	return ast, nil
}

func sadlExamplesForAction(model *sadl.Model, hdef *sadl.HttpDef) []map[string]interface{} {
	opName := sadl.Capitalize(hdef.Name)
	reqType := opName + "Request"
	resType := opName + "Response"
	namedExamples := make(map[string]map[string]interface{}, 0)
	//each named example should be a pair of req/res, or req/exc
	for _, ex := range model.Examples {
		if ex.Target == reqType {
			c := ex.Comment
			e := make(map[string]interface{}, 0)
			e["input"] = ex.Example.(map[string]interface{})
			if c != "" {
				e["documentation"] = c
			}
			namedExamples[ex.Name] = e
		}
	}
	for _, ex := range model.Examples {
		if ex.Target != reqType {
			if data, ok := namedExamples[ex.Name]; ok {
				if ex.Target == resType {
					data["output"] = ex.Example.(map[string]interface{})
				} else {
					for _, exc := range hdef.Exceptions {
						if exc.Type == ex.Target {
							tmp := make(map[string]interface{}, 0)
							tmp["error"] = ex.Example.(map[string]interface{})
							tmp["shapeId"] = ex.Target
							data["error"] = tmp
							break
						}
					}
				}
			}
		}
	}
	result := make([]map[string]interface{}, 0)
	for k, v := range namedExamples {
		v["title"] = k
		result = append(result, v)
	}
	return result
}

func examplesFromSADL(ns string, ast *smithylib.AST, model *sadl.Model) {
	for _, hact := range model.Http {
		name := capitalize(hact.Name)
		if ns != "" {
			name = ns + "#" + name
		}
		examples := sadlExamplesForAction(model, hact)
		shape := ast.Shapes.Get(name)
		if shape == nil {
			continue
		}
		if len(examples) > 0 {
			ensureShapeTraits(shape).Put("smithy.api#examples", examples)
		}
	}
}

func typeReference(ns string, ts *sadl.TypeSpec) string {
	return typeReferenceByName(ns, ts.Type)
}

func typeReferenceByName(ns string, name string) string {
	switch name {
	case "Bool":
		return "smithy.api#Boolean"
	case "Int8":
		return "smithy.api#Byte"
	case "Int16":
		return "smithy.api#Short"
	case "Int32":
		return "smithy.api#Integer"
	case "Int64":
		return "smithy.api#Long"
	case "Float32":
		return "smithy.api#Float"
	case "Float64":
		return "smithy.api#Double"
	case "Decimal":
		return "smithy.api#BigDecimal"
	case "Timestamp":
		return "smithy.api#Timestamp"
	case "UUID":
		return "smithy.api#String" //Smithy doesn't have UUID
	case "Bytes":
		return "smithy.api#Blob"
	case "String":
		return "smithy.api#String"
	case "Array":
		return "smithy.api#List"
	case "Map":
		return "smithy.api#Map"
	case "Struct":
		return "smithy.api#Document" //naked struct only.
	default:
		return ns + "#" + name
	}
}

func getAnnotation(annos map[string]string, key string) string {
	if annos != nil {
		if s, ok := annos[key]; ok {
			if s == "" {
				s = "true"
			}
			return s
		}
	}
	return ""
}

func listTypeReference(model *sadl.Model, ns string, shapes *smithylib.Shapes, prefix string, fd *sadl.StructFieldDef) string {
	ftype := capitalize(prefix) + capitalize(fd.Name)
	td := model.FindType(ftype)
	if td != nil {
		fmt.Printf("Inline defs not allowed, synthesize %q to refer to: %s\n", ftype, sadl.Pretty(fd))
		panic("Already have one with that name!!!")
	}
	ltype := "list"
	if getAnnotation(fd.Annotations, "x_unique") == "true" {
		ltype = "set"
	}
	shape := smithylib.Shape{
		Type: ltype,
	}
	shape.Member = &smithylib.Member{
		Target: typeReferenceByName(ns, fd.Items),
	}
	ensureShapeTraits(&shape).Put("smithy.api#documentation", "[autogenerated for field '"+fd.Name+"' in struct '"+prefix+"']")
	shapes.Put(ns+"#"+ftype, &shape)
	return ftype
}

func mapTypeReference(model *sadl.Model, ns string, shapes *smithylib.Shapes, prefix string, fd *sadl.StructFieldDef) string {
	ftype := capitalize(prefix) + capitalize(fd.Name)
	td := model.FindType(ftype)
	if td != nil {
		fmt.Printf("Inline defs not allowed, synthesize %q to refer to: %s\n", ftype, sadl.Pretty(fd))
		panic("Already have one with that name!!!")
	}
	shape := smithylib.Shape{
		Type: "map",
	}
	shape.Key = &smithylib.Member{
		Target: typeReferenceByName(ns, fd.Keys),
	}
	shape.Value = &smithylib.Member{
		Target: typeReferenceByName(ns, fd.Items),
	}
	ensureShapeTraits(&shape).Put("smithy.api#documentation", "[autogenerated for field '"+fd.Name+"' in struct '"+prefix+"']")
	shapes.Put(ns+"#"+ftype, &shape)
	return ftype
}

func enumTypeReference(model *sadl.Model, ns string, shapes *smithylib.Shapes, prefix string, fd *sadl.StructFieldDef) string {
	ftype := capitalize(prefix) + capitalize(fd.Name)
	td := model.FindType(ftype)
	if td != nil {
		fmt.Printf("Inline defs not allowed, synthesize %q to refer to: %s\n", ftype, sadl.Pretty(fd))
		panic("Already have one with that name!!!")
	}
	ltype := "string"
	shape := smithylib.Shape{
		Type: ltype,
	}
	shape.Member = &smithylib.Member{
		Target: typeReferenceByName(ns, fd.Items),
	}
	ensureShapeTraits(&shape).Put("smithy.api#enum", enumTrait(&fd.TypeSpec))
	ensureShapeTraits(&shape).Put("smithy.api#documentation", "[autogenerated for field '"+fd.Name+"' in struct '"+prefix+"']")
	shapes.Put(ns+"#"+ftype, &shape)
	return ftype
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[0:1]) + s[1:]
}

func ensureShapeTraits(shape *smithylib.Shape) *data.Object {
	if shape.Traits == nil {
		shape.Traits = data.NewObject()
	}
	return shape.Traits
}

func ensureMemberTraits(member *smithylib.Member) *data.Object {
	if member.Traits == nil {
		member.Traits = data.NewObject()
	}
	return member.Traits
}

func defineShapeFromTypeSpec(model *sadl.Model, ns string, shapes *smithylib.Shapes, ts *sadl.TypeSpec, name string, comment string, annos map[string]string) error {
	var shape smithylib.Shape
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
		shape = shapeFromArray(model, ns, shapes, name, ts, annos)
	case "Union":
		shape = shapeFromUnion(model, ns, shapes, name, ts)
	case "Map":
		shape = shapeFromMap(model, ns, shapes, name, ts)
	case "UUID":
		shape = *uuidShape()
	default:
		fmt.Println("So far:", sadl.Pretty(model))
		panic("handle this type:" + sadl.Pretty(ts))
	}
	if comment != "" {
		ensureShapeTraits(&shape).Put("smithy.api#documentation", comment)
	}
	if annos != nil {
		for k, v := range annos {
			switch k {
			case "x_tags":
				ensureShapeTraits(&shape).Put("smithy.api#tags", strings.Split(v, ","))
			case "x_sensitive":
				ensureShapeTraits(&shape).Put("smithy.api#sensitive", true)
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
					ensureShapeTraits(&shape).Put("smithy.api#deprecated", dep)
				}
			}
		}
	}
	shapes.Put(ns+"#"+name, &shape)
	return nil
}

func shapeFromArray(model *sadl.Model, ns string, shapes *smithylib.Shapes, tname string, ts *sadl.TypeSpec, annos map[string]string) smithylib.Shape {
	member := smithylib.Member{
		Target: EnsureNamespaced(ns, typeReferenceByName(ns, ts.Items)),
	}
	ltype := "list"
	if getAnnotation(annos, "x_unique") == "true" {
		ltype = "set"
	}
	shape := smithylib.Shape{
		Type:   ltype,
		Member: &member,
	}
	l := lengthTrait(ts.MinSize, ts.MaxSize)
	if l != nil {
		ensureShapeTraits(&shape).Put("smithy.api#length", l)
	}
	return shape
}

func shapeFromMap(model *sadl.Model, ns string, shapes *smithylib.Shapes, tname string, ts *sadl.TypeSpec) smithylib.Shape {
	key := smithylib.Member{
		Target: EnsureNamespaced(ns, typeReferenceByName(ns, ts.Keys)),
	}
	value := smithylib.Member{
		Target: EnsureNamespaced(ns, typeReferenceByName(ns, ts.Items)),
	}
	shape := smithylib.Shape{
		Type:  "map",
		Key:   &key,
		Value: &value,
	}
	//	l := lengthTrait(ts.MinSize, ts.MaxSize)
	//	if l != nil {
	//		ensureShapeTraits(&shape)["smithy.api#length"] = l
	//	}
	return shape
}

func shapeFromStruct(model *sadl.Model, ns string, shapes *smithylib.Shapes, tname string, ts *sadl.TypeSpec) smithylib.Shape {
	shape := smithylib.Shape{
		Type: "structure",
	}
	members := smithylib.NewMembers()
	for _, fd := range ts.Fields {
		ftype := typeReference(ns, &fd.TypeSpec)
		switch ftype {
		case "List":
			ftype = listTypeReference(model, ns, shapes, tname, fd)
		case "Map":
			ftype = mapTypeReference(model, ns, shapes, tname, fd)
		case "Enum":
			ftype = enumTypeReference(model, ns, shapes, tname, fd)
		}
		member := &smithylib.Member{
			Target: ftype,
		}
		if fd.Required {
			ensureMemberTraits(member).Put("smithy.api#required", true)
		}
		members.Put(fd.Name, member)
	}
	shape.Members = members
	return shape
}

func shapeFromUnion(model *sadl.Model, ns string, shapes *smithylib.Shapes, tname string, ts *sadl.TypeSpec) smithylib.Shape {
	shape := smithylib.Shape{
		Type: "union",
	}
	members := smithylib.NewMembers()
	for _, vd := range ts.Variants { //todo: modify SADL to make unions more like structs
		//		fd := model.FindType(vtype.Type)
		//		ftype := typeReference(&fd.TypeSpec)
		member := &smithylib.Member{
			Target: EnsureNamespaced(ns, vd.Type),
		}
		ensureMemberTraits(member).Put("smithy.api#documentation", vd.Comment)
		members.Put(vd.Name, member)
	}
	shape.Members = members
	return shape
}

func shapeFromString(ts *sadl.TypeSpec) smithylib.Shape {
	shape := smithylib.Shape{
		Type: "string",
	}
	l := lengthTrait(ts.MinSize, ts.MaxSize)
	if l != nil {
		ensureShapeTraits(&shape).Put("smithy.api#length", l)
	}
	if ts.Pattern != "" {
		ensureShapeTraits(&shape).Put("smithy.api#pattern", ts.Pattern)
	}
	if len(ts.Values) > 0 {
		ensureShapeTraits(&shape).Put("smithy.api#enum", enumTrait(ts))
	}
	return shape
}

func shapeFromNumber(ts *sadl.TypeSpec) smithylib.Shape {
	shape := smithylib.Shape{}
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
		ensureShapeTraits(&shape).Put("smithy.api#range", rangeTrait(ts.Min, ts.Max))
	}
	return shape
}

func uuidShape() *smithylib.Shape {
	shape := smithylib.Shape{
		Type: "string",
	}
	ensureShapeTraits(&shape).Put("smithy.api#pattern", "([a-f0-9]{8}(-[a-f0-9]{4}){4}[a-f0-9]{8})")
	return &shape
}

func shapeFromEnum(ts *sadl.TypeSpec) smithylib.Shape {
	shape := smithylib.Shape{
		Type: "enum",
	}
	mems := smithylib.NewMembers()
	for _, el := range ts.Elements {
		mem := &smithylib.Member{
			Target: "smithy.api#Unit",
		}
		if el.Annotations != nil {
			if val, ok := el.Annotations["x_enumValue"]; ok {
				ensureMemberTraits(mem).Put("smithy.api#enumValue", val)
			}
		}
		mems.Put(el.Symbol, mem)
	}
	shape.Members = mems
	return shape
}

func enumTrait(ts *sadl.TypeSpec) []interface{} {
	e := make([]interface{}, 0)
	if len(ts.Elements) > 0 {
		for _, eds := range ts.Elements {
			ei := make(map[string]interface{}, 0)
			ei["value"] = eds.Symbol
			ei["name"] = eds.Symbol
			if eds.Comment != "" {
				ei["documentation"] = eds.Comment
			}
			//ei["tags"] == ?
			e = append(e, ei)
		}
	} else if len(ts.Values) > 0 {
		for _, s := range ts.Values {
			ei := make(map[string]interface{}, 0)
			ei["value"] = s
			e = append(e, ei)
		}
	}
	return e
}

func paginatedTrait(sval string) map[string]interface{} {
	lst := strings.Split(sval, ",")
	m := make(map[string]interface{}, 0)
	for _, item := range lst {
		kv := strings.Split(item, "=")
		m[kv[0]] = kv[1]
	}
	return m
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

func EnsureNamespaced(ns, name string) string {
	switch name {
	case "Boolean", "Byte", "Short", "Integer", "Long", "Float", "Double", "BigInteger", "BigDecimal":
		return name
	case "Blob", "String", "Timestamp", "UUID", "Enum":
		return name
	case "List", "Map", "Set", "Document", "Structure", "Union":
		return name
	}
	if strings.Index(name, "#") < 0 {
		return ns + "#" + name
	}
	return name
}
