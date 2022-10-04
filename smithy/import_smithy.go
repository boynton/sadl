package smithy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/boynton/data"
	"github.com/boynton/sadl"
	smithylib "github.com/boynton/smithy"
)

const UnspecifiedNamespace = "example"
const UnspecifiedVersion = "0.0"

// set to true to prevent enum traits that have only values from ever becoming actul enum objects.
var StringValuesNeverEnum bool = false

func IsValidFile(path string) bool {
	if strings.HasSuffix(path, ".smithy") {
		return true
	}
	if strings.HasSuffix(path, ".json") {
		_, err := smithylib.LoadAST(path)
		return err == nil
	}
	return false
}

func Import(paths []string, conf *sadl.Data) (*sadl.Model, error) {
	var err error
	name := conf.GetString("name")
	namespace := conf.GetString("namespace")
	if namespace == "" {
		conf.Put("namespace", UnspecifiedNamespace)
	}
	var tags []string //fir filtering, if non-nil
	model, err := AssembleModel(paths, tags)
	if err != nil {
		return nil, err
	}
	conf.Put("name", name)
	return ToSadl(model, conf)
}

/*
func (model *Model) addShape(absoluteName string, shape *Shape) {
	prefix := model.namespace + "#"
	prefixLen := len(prefix)
	if strings.HasPrefix(absoluteName, prefix) {
		if _, ok := model.ast.Shapes[absoluteName]; !ok {
			model.ast.Shapes[absoluteName] = shape
		}
		kk := absoluteName[prefixLen:]
		model.shapes[kk] = shape
		if shape.Type == "operation" {
			if shape.Input != nil {
				fmt.Println(sadl.Pretty(shape), prefixLen)
				iok := shape.Input.Target[prefixLen:]
				model.ioParams[iok] = shape.Input.Target
			}
			if shape.Output != nil {
				iok := shape.Output.Target[prefixLen:]
				model.ioParams[iok] = shape.Output.Target
			}
		}
	}
}

func NewModel(ast *AST) *Model {
	model := &Model{
		ast: ast,
	}
	model.shapes = make(map[string]*Shape, 0)
	model.ioParams = make(map[string]string, 0)
	model.namespace, model.name, model.version = ast.NamespaceAndServiceVersion()

	if model.name == "" {
		s := sadl.GetString(ast.Metadata, "name")
		if s != "" {
			model.name = s
		}
	}

	for k, v := range ast.Shapes {
		model.addShape(k, v)
	}
	return model
}
*/

type Importer struct {
	name      string
	namespace string
	ast       *smithylib.AST
	ioParams  map[string]*smithylib.Shape
	schema    *sadl.Schema
}

func ToSadl(ast *smithylib.AST, conf *sadl.Data) (*sadl.Model, error) {
	i := &Importer{
		ast:      ast,
		ioParams: make(map[string]*smithylib.Shape, 0),
	}
	name := conf.GetString("name")
	if name != "" {
		i.name = name
	} else {
		s := ast.Metadata.GetString("name")
		if s != "" {
			i.name = s
			//			delete(ast.Metadata, "name")
		}
	}
	service := conf.GetString("service")
	namespace := conf.GetString("namespace")

	if namespace != UnspecifiedNamespace {
		i.namespace = namespace
	}
	annos := make(map[string]string, 0)

	if ast.Metadata != nil {
		for _, k := range ast.Metadata.Keys() {
			if k != "name" {
				s := sadl.AsString(ast.Metadata.Get(k))
				if k == "base" {
					annos[k] = s
				} else {
					annos["x_"+k] = s
				}
			}
		}
	}
	//	annos["x_smithy_version"] = model.ast.Version
	schema := &sadl.Schema{
		Name:      name,
		Namespace: namespace,
		//		Version:     ast.Smithy,
		Annotations: annos,
	}
	if schema.Namespace == UnspecifiedNamespace {
		schema.Namespace = ""
	}
	if schema.Version == UnspecifiedVersion {
		schema.Version = ""
	}
	i.schema = schema

	haveService := ""
	for _, shapeName := range ast.Shapes.Keys() {
		shapeDef := ast.Shapes.Get(shapeName)
		if shapeDef.Type == "service" {
			if service != "" {
				if shapeName != service {
					continue
				}
			} else {
				if haveService != "" {
					return nil, fmt.Errorf("SADL only supports one service per model (%s, %s)", haveService, shapeName)
				}
			}
			haveService = shapeName
		} else if shapeDef.Type == "operation" {
			if shapeDef.Input != nil {
				i.ioParams[shapeDef.Input.Target] = i.ast.GetShape(shapeDef.Input.Target)
			}
			if shapeDef.Output != nil {
				i.ioParams[shapeDef.Output.Target] = i.ast.GetShape(shapeDef.Output.Target)
			}
		}
	}

	for _, k := range ast.Shapes.Keys() {
		v := ast.Shapes.Get(k)
		if v.Type != "service" || k == haveService {
			i.importShape(k, v)
		}
	}
	return sadl.NewModel(schema)
}

func (i *Importer) importShape(shapeName string, shapeDef *smithylib.Shape) {
	if _, ok := i.ioParams[shapeName]; ok {
		return
	}
	switch shapeDef.Type {
	case "byte", "short", "integer", "long", "float", "double", "bigInteger", "bigDecimal":
		i.importNumericShape(shapeDef.Type, shapeName, shapeDef)
	case "string":
		i.importStringShape(shapeName, shapeDef)
	case "timestamp":
		i.importTimestampShape(shapeName, shapeDef)
	case "boolean":
		i.importBooleanShape(shapeName, shapeDef)
	case "list":
		i.importListShape(shapeName, shapeDef, false)
	case "set":
		i.importListShape(shapeName, shapeDef, true)
	case "map":
		i.importMapShape(shapeName, shapeDef)
	case "structure":
		i.importStructureShape(shapeName, shapeDef)
	case "union":
		i.importUnionShape(shapeName, shapeDef)
	case "blob":
		i.importBlobShape(shapeName, shapeDef)
	case "enum":
		i.importEnumShape(shapeName, shapeDef)
	case "service":
		//FIX schema.Name = shapeName
	case "operation":
		i.importOperationShape(shapeName, shapeDef)
	case "resource":
		i.importResourceShape(shapeName, shapeDef)
	default:
		fmt.Println("fix me, unhandled shape type: " + shapeDef.Type)
		panic("whoa")
	}
}

func escapeComment(doc string) string {
	lines := strings.Split(doc, "\n")
	if len(lines) == 1 {
		return doc
	}
	for i, line := range lines {
		lines[i] = strings.Trim(line, " ")
	}
	//	return strings.Replace(strings.Replace(doc, "\n", " ", -1), "  ", " ", -1)
	return strings.Join(lines, "\n")
}

func (i *Importer) importNumericShape(smithyType string, shapeName string, shape *smithylib.Shape) {
	sadlName := stripNamespace(shapeName)
	td := &sadl.TypeDef{
		Name:        sadlName,
		Comment:     escapeComment(shape.Traits.GetString("smithy.api#documentation")),
		Annotations: i.importTraitsAsAnnotations(nil, shape.Traits),
	}
	switch smithyType {
	case "byte":
		td.Type = "Int8"
	case "short":
		td.Type = "Int16"
	case "integer":
		td.Type = "Int32"
	case "long":
		td.Type = "Int64"
	case "float":
		td.Type = "Float32"
	case "double":
		td.Type = "Float64"
	case "bigInteger":
		td.Type = "Decimal"
		td.Annotations = WithAnnotation(td.Annotations, "x_integer", "true")
	case "bigDecimal":
		td.Type = "Decimal"
	default:
		panic("whoops")
	}
	if l := shape.Traits.GetMap("smithy.api#range"); l != nil {
		tmp := sadl.GetDecimal(l, "min")
		if tmp != nil {
			td.Min = tmp
		}
		tmp = sadl.GetDecimal(l, "max")
		if tmp != nil {
			td.Max = tmp
		}
	}
	i.schema.Types = append(i.schema.Types, td)
}

func (i *Importer) importStringShape(shapeName string, shape *smithylib.Shape) {
	sadlName := stripNamespace(shapeName)
	if sadlName == "UUID" {
		fmt.Println("UUID shape name:", shapeName)
		//UUID is already a builtin SADL type
		return
	}
	if i.importEnum(shapeName, shape) {
		return
	}
	td := &sadl.TypeDef{
		Name:    sadlName,
		Comment: escapeComment(shape.Traits.GetString("smithy.api#documentation")),
	}
	pat := shape.Traits.GetString("smithy.api#pattern")
	if pat == "([a-f0-9]{8}(-[a-f0-9]{4}){4}[a-f0-9]{8})" {
		td.Type = "UUID"
	} else {
		td.Type = "String"
		td.Pattern = pat
		if l := shape.Traits.GetMap("smithy.api#length"); l != nil {
			tmp := sadl.GetInt64(l, "min")
			if tmp != 0 {
				td.MinSize = &tmp
			}
			tmp = sadl.GetInt64(l, "max")
			if tmp != 0 {
				td.MaxSize = &tmp
			}
		}
		lst := shape.Traits.GetArray("smithy.api#enum")
		if lst != nil {
			var values []string
			for _, v := range lst {
				m := sadl.AsMap(v)
				s := sadl.GetString(m, "value")
				values = append(values, s)
			}
			td.Values = values
		}
	}
	i.schema.Types = append(i.schema.Types, td)
}

func isSmithyRecommendedEnumName(s string) bool {
	if s == "" {
		return false
	}
	for i, ch := range s {
		if i == 0 {
			if !sadl.IsUppercaseLetter(ch) {
				return false
			}
		} else {
			if !(sadl.IsUppercaseLetter(ch) || sadl.IsDigit(ch) || ch == '_') {
				return false
			}
		}
	}
	return true
}

func (i *Importer) importEnum(shapeName string, shape *smithylib.Shape) bool {
	lst := shape.Traits.GetArray("smithy.api#enum")
	if lst == nil {
		return false
	}
	var elements []*sadl.EnumElementDef
	isEnum := false
	couldBeEnum := false
	for _, v := range lst {
		m := sadl.AsMap(v)
		sym := sadl.GetString(m, "name")
		if sym == "" {
			if StringValuesNeverEnum {
				return false
			}
			sym = sadl.GetString(m, "value")
			if !isSmithyRecommendedEnumName(sym) {
				return false
			}
		}
		element := &sadl.EnumElementDef{
			Symbol:  sym,
			Comment: sadl.GetString(m, "documentation"),
		}
		//tags -> annotations?
		elements = append(elements, element)
	}
	if !isEnum {
		if !couldBeEnum { //might want a preference on this, if the values happen to follow symbol rules now, but really are values
			return false
		}
	}
	sadlName := stripNamespace(shapeName)
	td := &sadl.TypeDef{
		Name:    sadlName,
		Comment: escapeComment(shape.Traits.GetString("smithy.api#documentation")),
	}
	td.Type = "Enum"
	td.Elements = elements
	i.schema.Types = append(i.schema.Types, td)
	return true
}

func (i *Importer) importTimestampShape(shapeName string, shape *smithylib.Shape) {
	sadlName := stripNamespace(shapeName)
	td := &sadl.TypeDef{
		Name:        sadlName,
		Comment:     escapeComment(shape.Traits.GetString("smithy.api#documentation")),
		Annotations: i.importTraitsAsAnnotations(nil, shape.Traits),
	}
	td.Type = "Timestamp"
	i.schema.Types = append(i.schema.Types, td)
}

func (i *Importer) importBlobShape(shapeName string, shape *smithylib.Shape) {
	sadlName := stripNamespace(shapeName)
	td := &sadl.TypeDef{
		Name:    sadlName,
		Comment: escapeComment(shape.Traits.GetString("smithy.api#documentation")),
	}
	td.Type = "Bytes"
	if l := shape.Traits.GetMap("smithy.api#length"); l != nil {
		tmp := sadl.GetInt64(l, "min")
		if tmp != 0 {
			td.MinSize = &tmp
		}
		tmp = sadl.GetInt64(l, "max")
		if tmp != 0 {
			td.MaxSize = &tmp
		}
	}
	i.schema.Types = append(i.schema.Types, td)
}

func (i *Importer) importBooleanShape(shapeName string, shape *smithylib.Shape) {
	sadlName := stripNamespace(shapeName)
	td := &sadl.TypeDef{
		Name:        sadlName,
		Comment:     escapeComment(shape.Traits.GetString("smithy.api#documentation")),
		Annotations: i.importTraitsAsAnnotations(nil, shape.Traits),
	}
	td.Type = "Boolean"
	i.schema.Types = append(i.schema.Types, td)
}

func (i *Importer) importTraitsAsAnnotations(annos map[string]string, traits *data.Object) map[string]string {
	for _, k := range traits.Keys() {
		v := traits.Get(k)
		switch k {
		case "smithy.api#error":
			annos = WithAnnotation(annos, "x_"+stripNamespace(k), sadl.AsString(v))
		case "smithy.api#httpError":
			annos = WithAnnotation(annos, "x_"+stripNamespace(k), fmt.Sprintf("%v", v))
		case "smithy.api#httpPayload", "smithy.api#httpLabel", "smithy.api#httpQuery", "smithy.api#httpHeader":
			/* ignore, implicit in SADL */
		case "smithy.api#required", "smithy.api#documentation", "smithy.api#range", "smithy.api#length":
			/* ignore, implicit in SADL */
		case "smithy.api#tags":
			annos = WithAnnotation(annos, "x_"+stripNamespace(k), strings.Join(sadl.AsStringArray(v), ","))
		case "smithy.api#readonly", "smithy.api#idempotent", "smithy.api#sensitive", "smithy.api#box":
			//			annos = WithAnnotation(annos, "x_"+stripNamespace(k), "true")
		case "smithy.api#http":
			/* ignore, handled elsewhere */
		case "smithy.api#timestampFormat", "smithy.api#enumValue":
			annos = WithAnnotation(annos, "x_"+stripNamespace(k), sadl.AsString(v))
		case "smithy.api#deprecated":
			//message
			//since
			dv := sadl.AsMap(v)
			annos = WithAnnotation(annos, "x_deprecated", "true")
			msg := sadl.GetString(dv, "message")
			if msg != "" {
				annos = WithAnnotation(annos, "x_deprecated_message", msg)
			}
			since := sadl.GetString(dv, "since")
			if since != "" {
				annos = WithAnnotation(annos, "x_deprecated_since", since)
			}
		case "smithy.api#paginated":
			dv := sadl.AsMap(v)
			inputToken := sadl.AsString(dv["inputToken"])
			outputToken := sadl.AsString(dv["outputToken"])
			pageSize := sadl.AsString(dv["pageSize"])
			items := sadl.AsString(dv["items"])
			s := fmt.Sprintf("inputToken=%s,outputToken=%s,pageSize=%s,items=%s", inputToken, outputToken, pageSize, items)
			annos = WithAnnotation(annos, "x_paginated", s)
		case "aws.protocols#restJson1":
			//ignore
		case "smithy.api#examples":
			//ignore for now
		default:
			tshape := i.ast.GetShape(k)
			if tshape != nil {
				tm := tshape.Traits.GetMap("smithy.api#trait")
				if tm != nil {
					fmt.Println("tshape:", k, tm)
					//FIX ME
					//I don't really have a place to put non-string annotations (note how I stuff string annos into "x_name").
					//So I really need arbitrary annotations values to do this. Note also: need multiple ns support!
				} else {
					fmt.Println("Unhandled trait:", k, " =", sadl.Pretty(v))
					panic("here: " + k)
				}
			} else {
				if strings.HasPrefix(k, "smithy.api#") {
					fmt.Println("Unhandled trait:", k, " =", sadl.Pretty(v))
					panic("here: " + k)
				} //else ignore
			}
		}
	}
	return annos
}

func (i *Importer) importEnumShape(shapeName string, shape *smithylib.Shape) {
	sadlName := stripNamespace(shapeName)
	td := &sadl.TypeDef{
		Name:        sadlName,
		Comment:     escapeComment(shape.Traits.GetString("smithy.api#documentation")),
		Annotations: i.importTraitsAsAnnotations(nil, shape.Traits),
	}
	td.Type = "Enum"
	//	prefix := model.namespace + "#"
	var elements []*sadl.EnumElementDef
	for _, memberName := range shape.Members.Keys() {
		member := shape.Members.Get(memberName)
		ee := &sadl.EnumElementDef{
			Symbol:      memberName,
			Comment:     escapeComment(member.Traits.GetString("smithy.api#documentation")),
			Annotations: i.importTraitsAsAnnotations(nil, member.Traits),
		}
		elements = append(elements, ee)
	}
	td.Type = "Enum"
	td.Elements = elements
	i.schema.Types = append(i.schema.Types, td)
}

func (i *Importer) importUnionShape(shapeName string, shape *smithylib.Shape) {
	sadlName := stripNamespace(shapeName)
	td := &sadl.TypeDef{
		Name:        sadlName,
		Comment:     escapeComment(shape.Traits.GetString("smithy.api#documentation")),
		Annotations: i.importTraitsAsAnnotations(nil, shape.Traits),
	}
	td.Type = "Union"
	//	prefix := model.namespace + "#"
	for _, memberName := range shape.Members.Keys() {
		member := shape.Members.Get(memberName)
		//		if memberName != member.Target {
		vd := &sadl.UnionVariantDef{
			Name:        memberName,
			Comment:     escapeComment(member.Traits.GetString("smithy.api#documentation")),
			Annotations: i.importTraitsAsAnnotations(nil, member.Traits),
		}
		vd.Type = i.shapeRefToTypeRef(member.Target)
		/*
			m := member.Target
			if strings.HasPrefix(m, prefix) {
				m = member.Target[len(prefix):]
			}
			if m != memberName {
				fmt.Printf("member: %q, target: %q, m: %q\n", memberName, member.Target, m)
				panic("fixme: named union variants")
			}
		*/
		//		}
		//		td.Variants = append(td.Variants, model.shapeRefToTypeRef(schema, member.Target))
		td.Variants = append(td.Variants, vd)
	}
	i.schema.Types = append(i.schema.Types, td)
}

func (i *Importer) importStructureShape(shapeName string, shape *smithylib.Shape) {
	//unless...no httpTraits, in which case is should equivalent to a payload description
	if _, ok := i.ioParams[shapeName]; ok {
		shape := i.ast.GetShape(shapeName)
		for _, k := range shape.Members.Keys() {
			fval := shape.Members.Get(k)
			if fval.Traits.GetBool("smithy.api#httpLabel") {
				return
			}
			if fval.Traits.GetBool("smithy.api#httpPayload") {
				return
			}
			if fval.Traits.GetString("smithy.api#httpQuery") != "" {
				return
			}
		}
	}
	sadlName := stripNamespace(shapeName)
	td := &sadl.TypeDef{
		Name:        sadlName,
		Comment:     escapeComment(shape.Traits.GetString("smithy.api#documentation")),
		Annotations: i.importTraitsAsAnnotations(nil, shape.Traits),
	}
	td.Type = "Struct"
	var fields []*sadl.StructFieldDef
	for _, memberName := range shape.Members.Keys() {
		member := shape.Members.Get(memberName)
		fd := &sadl.StructFieldDef{
			Name:        memberName,
			Comment:     escapeComment(member.Traits.GetString("smithy.api#documentation")),
			Annotations: i.importTraitsAsAnnotations(nil, member.Traits),
			Required:    member.Traits.GetBool("smithy.api#required"),
		}
		fd.Type = i.shapeRefToTypeRef(member.Target)
		if member.Traits.GetBool("smithy.api#required") {
			fd.Required = true
		}
		fields = append(fields, fd)
	}
	td.Fields = fields
	i.schema.Types = append(i.schema.Types, td)
}

func (i *Importer) importListShape(shapeName string, shape *smithylib.Shape, unique bool) {
	sadlName := stripNamespace(shapeName)
	td := &sadl.TypeDef{
		Name:    sadlName,
		Comment: escapeComment(shape.Traits.GetString("smithy.api#documentation")),
	}
	td.Type = "Array"
	td.Items = i.shapeRefToTypeRef(shape.Member.Target)
	tmp := shape.Traits.GetInt64("smithy.api#min")
	if tmp != 0 {
		td.MinSize = &tmp
	}
	tmp = shape.Traits.GetInt64("smithy.api#max")
	if tmp != 0 {
		td.MaxSize = &tmp
	}
	if unique {
		td.Annotations = WithAnnotation(td.Annotations, "x_unique", "true")
	}
	i.schema.Types = append(i.schema.Types, td)
}

func (i *Importer) importMapShape(shapeName string, shape *smithylib.Shape) {
	sadlName := stripNamespace(shapeName)
	td := &sadl.TypeDef{
		Name:    sadlName,
		Comment: escapeComment(shape.Traits.GetString("smithy.api#documentation")),
	}
	td.Type = "Map"
	td.Keys = i.shapeRefToTypeRef(shape.Key.Target)
	td.Items = i.shapeRefToTypeRef(shape.Value.Target)
	tmp := shape.Traits.GetInt64("smithy.api#min")
	if tmp != 0 {
		td.MinSize = &tmp
	}
	tmp = shape.Traits.GetInt64("smithy.api#max")
	if tmp != 0 {
		td.MaxSize = &tmp
	}
	i.schema.Types = append(i.schema.Types, td)
}

func (i *Importer) ensureLocalNamespace(id string) string {
	if strings.Index(id, "#") < 0 {
		return id //already local
	}
	prefix := i.namespace + "#"
	if strings.HasPrefix(id, prefix) {
		return id[len(prefix):]
	}
	return ""
}

func (i *Importer) shapeRefToTypeRef(shapeRef string) string {
	typeRef := shapeRef
	switch typeRef {
	case "smithy.api#Blob", "Blob":
		return "Bytes"
	case "smithy.api#Boolean", "Boolean":
		return "Bool"
	case "smithy.api#String", "String":
		return "String"
	case "smithy.api#Byte", "Byte":
		return "Int8"
	case "smithy.api#Short", "Short":
		return "Int16"
	case "smithy.api#Integer", "Integer":
		return "Int32"
	case "smithy.api#Long", "Long":
		return "Int64"
	case "smithy.api#Float", "Float":
		return "Float32"
	case "smithy.api#Double", "Double":
		return "Float64"
	case "smithy.api#BigInteger", "BigInteger":
		return "Decimal" //lossy!
	case "smithy.api#BigDecimal", "BigDecimal":
		return "Decimal"
	case "smithy.api#Timestamp", "Timestamp":
		return "Timestamp"
	case "smithy.api#Document", "Document":
		return "Struct" //todo: introduce a separate type for open structs.
	default:
		if true {
			typeRef = stripNamespace(typeRef)
		} else {
			ltype := i.ensureLocalNamespace(typeRef)
			if ltype == "" {

				panic("external namespace type reference not supported: " + typeRef)
			}
			typeRef = ltype
		}
	}
	return typeRef
}

func (i *Importer) importResourceShape(shapeName string, shape *smithylib.Shape) {
	//to do: preserve the resource info as tags on the operations
}

func (i *Importer) importOperationShape(shapeName string, shape *smithylib.Shape) {
	var method, uri string
	var code int
	if ht := shape.Traits.GetObject("smithy.api#http"); ht != nil {
		method = ht.GetString("method")
		uri = ht.GetString("uri")
		code = ht.GetInt("code")
	}
	var inType string
	var inShapeName string
	if shape.Input != nil {
		inShapeName = shape.Input.Target
		inType = i.shapeRefToTypeRef(inShapeName)
	}
	var outType string
	var outShapeName string
	if shape.Output != nil {
		outShapeName = shape.Output.Target
		outType = i.shapeRefToTypeRef(outShapeName)
	}
	reqType := sadl.Capitalize(shapeName) + inputSuffix
	resType := sadl.Capitalize(shapeName) + outputSuffix
	if opexes := shape.Traits.GetArray("smithy.api#examples"); opexes != nil {
		for _, opexr := range opexes {
			if opex, ok := opexr.(map[string]interface{}); ok {
				title := sadl.GetString(opex, "title")
				if input, ok := opex["input"]; ok {
					ex := &sadl.ExampleDef{
						Target:  reqType,
						Name:    title,
						Comment: sadl.GetString(opex, "documentation"),
						Example: input,
					}
					i.schema.Examples = append(i.schema.Examples, ex)
				}
				if output, ok := opex["output"]; ok {
					ex := &sadl.ExampleDef{
						Target:  resType,
						Name:    title,
						Example: output,
					}
					i.schema.Examples = append(i.schema.Examples, ex)
				}
			}
		}
	}

	sadlName := stripNamespace(shapeName)
	if method == "" {
		odef := &sadl.OperationDef{
			Name:        sadl.Uncapitalize(sadlName),
			Comment:     escapeComment(shape.Traits.GetString("smithy.api#documentation")),
			Annotations: i.importTraitsAsAnnotations(nil, shape.Traits),
		}
		if shape.Input != nil {
			inStruct := i.ast.GetShape(inShapeName)
			var inputs []*sadl.OperationInput
			for _, fname := range inStruct.Members.Keys() {
				fval := inStruct.Members.Get(fname)
				in := &sadl.OperationInput{}
				in.Name = fname
				in.Type = i.shapeRefToTypeRef(fval.Target)
				in.Required = fval.Traits.GetBool("smithy.api#required")
				inputs = append(inputs, in)
			}
			odef.Inputs = inputs
		}
		if shape.Output != nil {
			outStruct := i.ast.GetShape(outShapeName)
			var outputs []*sadl.OperationOutput
			for _, fname := range outStruct.Members.Keys() {
				out := &sadl.OperationOutput{}
				fval := outStruct.Members.Get(fname)
				out.Name = fname
				out.Type = i.shapeRefToTypeRef(fval.Target)
				outputs = append(outputs, out)
			}
			odef.Outputs = outputs
		}
		if shape.Errors != nil {
			//fix me
			for _, err := range shape.Errors {
				odef.Exceptions = append(odef.Exceptions, i.shapeRefToTypeRef(err.Target))
			}
		}
		i.schema.Operations = append(i.schema.Operations, odef)
		//the req/resp types are getting dropped
		return
	}
	hdef := &sadl.HttpDef{
		Method:      method,
		Path:        uri,
		Name:        sadl.Uncapitalize(sadlName),
		Comment:     escapeComment(shape.Traits.GetString("smithy.api#documentation")),
		Annotations: i.importTraitsAsAnnotations(nil, shape.Traits),
	}
	if code == 0 {
		code = 200
	}
	if shape.Input != nil {
		inStruct := i.ast.GetShape(inShapeName)
		qs := ""
		payloadMember := ""
		hasLabel := false
		hasQuery := false
		if inStruct != nil {
			for _, fname := range inStruct.Members.Keys() {
				fval := inStruct.Members.Get(fname)
				if fval.Traits.GetBool("smithy.api#httpPayload") {
					payloadMember = fname
				} else if fval.Traits.GetBool("smithy.api#httpLabel") {
					hasLabel = true
				} else if fval.Traits.GetBool("smithy.api#httpQuery") {
					hasQuery = true
				}
			}
			if hasLabel || hasQuery || payloadMember != "" {
				//the input might *have* a body
				for _, fname := range inStruct.Members.Keys() {
					fval := inStruct.Members.Get(fname)
					in := &sadl.HttpParamSpec{}
					in.Name = fname
					in.Type = i.shapeRefToTypeRef(fval.Target)
					in.Required = fval.Traits.GetBool("smithy.api#required")
					in.Query = fval.Traits.GetString("smithy.api#httpQuery")
					if in.Query != "" {
						if qs == "" {
							qs = "?"
						} else {
							qs = qs + "&"
						}
						qs = qs + fname + "={" + fname + "}"
					}
					in.Header = fval.Traits.GetString("smithy.api#httpHeader")
					in.Path = fval.Traits.GetBool("smithy.api#httpLabel")
					hdef.Inputs = append(hdef.Inputs, in)
				}
			} else {
				//the input *is* a body. Generate a name for it.
				in := &sadl.HttpParamSpec{}
				in.Name = "body"
				in.Type = inType
				hdef.Inputs = append(hdef.Inputs, in)
			}
			hdef.Path = hdef.Path + qs
		}
	} else {
		//		fmt.Println("no input:", data.Pretty(shape))
	}

	expected := &sadl.HttpExpectedSpec{
		Status: int32(code),
	}
	if shape.Output != nil {
		outStruct := i.ast.GetShape(outShapeName)
		//SADL: each output is a header or a (singular) payload.
		//Smithy: the output struct is the result payload, unless a field is marked as payload, which allows other fields
		//to be marked as header.
		outBodyField := ""
		hasLabel := false
		for _, fname := range outStruct.Members.Keys() {
			fval := outStruct.Members.Get(fname)
			if fval.Traits.GetBool("smithy.api#httpPayload") {
				outBodyField = fname
			} else if fval.Traits.GetBool("smithy.api#httpLabel") {
				hasLabel = true
			}
		}
		if outBodyField == "" && !hasLabel {
			//the entire output structure is the payload, no headers possible
			out := &sadl.HttpParamSpec{}
			out.Name = "body"
			out.Type = i.shapeRefToTypeRef(outType)
			expected.Outputs = append(expected.Outputs, out)
		} else {
			for _, fname := range outStruct.Members.Keys() {
				fval := outStruct.Members.Get(fname)
				out := &sadl.HttpParamSpec{}
				out.Name = fname
				out.Type = i.shapeRefToTypeRef(fval.Target)
				out.Required = fval.Traits.GetBool("smithy.api#required")
				out.Header = fval.Traits.GetString("smithy.api#httpHeader")
				out.Query = fval.Traits.GetString("smithy.api#httpQuery")
				out.Path = fval.Traits.GetBool("smithy.api#httpLabel")
				expected.Outputs = append(expected.Outputs, out)
			}
		}
	}
	if shape.Errors != nil {
		for _, etype := range shape.Errors {
			eShapeName := etype.Target
			eStruct := i.ast.GetShape(eShapeName)
			eType := i.shapeRefToTypeRef(eShapeName)
			if eStruct == nil {
				panic("error type not found: " + eShapeName)
			}
			exc := &sadl.HttpExceptionSpec{}
			exc.Type = eType
			exc.Status = int32(eStruct.Traits.GetInt("smithy.api#httpError"))
			exc.Comment = escapeComment(eStruct.Traits.GetString("smithy.api#documentation"))
			//preserve other traits as annotations?
			hdef.Exceptions = append(hdef.Exceptions, exc)
		}
	}
	//Comment string
	//Annotations map[string]string
	hdef.Expected = expected
	i.schema.Http = append(i.schema.Http, hdef)
}

func WithAnnotation(annos map[string]string, key string, value string) map[string]string {
	if value != "" {
		if annos == nil {
			annos = make(map[string]string, 0)
		}
		annos[key] = value
	}
	return annos
}

func stripNamespace(trait string) string {
	n := strings.Index(trait, "#")
	if n < 0 {
		return trait
	}
	return trait[n+1:]
}

func AssembleModel(paths []string, tags []string) (*smithylib.AST, error) {
	flatPathList, err := expandPaths(paths)
	if err != nil {
		return nil, err
	}
	assembly := &smithylib.AST{
		Smithy: "2",
	}
	for _, path := range flatPathList {
		var ast *smithylib.AST
		var err error
		ext := filepath.Ext(path)
		switch ext {
		case ".json":
			ast, err = smithylib.LoadAST(path)
		case ".smithy":
			ast, err = smithylib.Parse(path)
		default:
			return nil, fmt.Errorf("parse for file type %q not implemented", ext)
		}
		if err != nil {
			return nil, err
		}
		err = assembly.Merge(ast)
		if err != nil {
			return nil, err
		}
	}
	if len(tags) > 0 {
		assembly.Filter(tags)
	}
	err = assembly.Validate()
	if err != nil {
		return nil, err
	}
	return assembly, nil
}

var ImportFileExtensions = map[string][]string{
	".smithy": []string{"smithy"},
	".json":   []string{"smithy"},
}

func expandPaths(paths []string) ([]string, error) {
	var result []string
	for _, path := range paths {
		ext := filepath.Ext(path)
		if _, ok := ImportFileExtensions[ext]; ok {
			result = append(result, path)
		} else {
			fi, err := os.Stat(path)
			if err != nil {
				return nil, err
			}
			if fi.IsDir() {
				err = filepath.Walk(path, func(wpath string, info os.FileInfo, errIncoming error) error {
					if errIncoming != nil {
						return errIncoming
					}
					ext := filepath.Ext(wpath)
					if _, ok := ImportFileExtensions[ext]; ok {
						result = append(result, wpath)
					}
					return nil
				})
			}
		}
	}
	return result, nil
}
