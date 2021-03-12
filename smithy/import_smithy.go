package smithy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/boynton/sadl"
)

//set to true to prevent enum traits that have only values from ever becoming actul enum objects.
var StringValuesNeverEnum bool = false

func IsValidFile(path string) bool {
	if strings.HasSuffix(path, ".smithy") {
		return true
	}
	if strings.HasSuffix(path, ".json") {
		_, err := loadAST(path)
		return err == nil
	}
	return false
}

func loadAST(path string) (*AST, error) {
	var ast *AST
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Cannot read smithy AST file: %v\n", err)
	}
	err = json.Unmarshal(data, &ast)
	if err != nil {
		return nil, fmt.Errorf("Cannot parse Smithy AST file: %v\n", err)
	}
	if ast.Version == "" {
		return nil, fmt.Errorf("Cannot parse Smithy AST file: %v\n", err)
	}
	return ast, nil
}

func Import(paths []string, conf *sadl.Data) (*sadl.Model, error) {
	var model *Model
	var err error
	name := conf.GetString("name")
	namespace := conf.GetString("namespace")
	if namespace == "" {
		conf.Put("namespace", UnspecifiedNamespace)
		//		conf["namespace"] = UnspecifiedNamespace
	}
	for _, path := range paths {
		if name == "" {
			name = nameFromPath(path)
		}
		var ast *AST
		if strings.HasSuffix(path, ".json") {
			ast, err = loadAST(path)
		} else {
			//parse Smithy IDL
			ast, err = parse(path)
		}
		if err != nil {
			return nil, err
		}
		mod := NewModel(ast)
		if model == nil {
			model = mod
		} else {
			err = model.Merge(mod)
			if err != nil {
				return nil, err
			}
		}
	}
	conf.Put("name", name)
	return model.ToSadl(conf)
}

func nameFromPath(path string) string {
	name := path
	n := strings.LastIndex(name, "/")
	if n >= 0 {
		name = name[n+1:]
	}
	n = strings.LastIndex(name, ".")
	if n >= 0 {
		name = name[:n]
		name = strings.Replace(name, ".", "_", -1)
	}
	return name
}

type Model struct {
	ast       *AST
	name      string
	namespace string //the primary one, anyway. There may be multiple namespaces
	version   string //of the service
	shapes    map[string]*Shape
	ioParams  map[string]string
}

//maybe filter by tag top include only parts?
func (model *Model) Merge(another *Model) error {
	if model.ast.Version != another.ast.Version {
		return fmt.Errorf("cannot merge models of different Smithy versions")
	}
	if another.ast.Metadata != nil {
		if model.ast.Metadata == nil {
			model.ast.Metadata = another.ast.Metadata
		} else {
			for k, v2 := range another.ast.Metadata {
				if v1, ok := model.ast.Metadata[k]; ok {
					if a1, ok := v1.([]interface{}); ok {
						if a2, ok := v2.([]interface{}); ok {
							for _, v := range a2 {
								a1 = append(a1, v)
							}
							model.ast.Metadata[k] = a1
						} else {
							return fmt.Errorf("Cannot merge models: metadata %q conflict", k)
						}
					}
					if !sadl.Equivalent(v1, v2) {
						return fmt.Errorf("Cannot merge models: metadata %q conflict", k)
					}
				}
			}
		}
	}
	for shapeName, shape2 := range another.shapes {
		shape1 := model.getShape(shapeName)
		//todo: shape ID conflicts in merge should be observed. Despite case sensitive names, prevent case-insensitive dups
		absName := model.namespace + "#" + shapeName
		if shape1 == nil {
			model.addShape(absName, shape2)
		} else {
			return fmt.Errorf("Cannot merge models: %s is a duplicate shape", absName)
		}
	}
	//multiple services should cause an error
	return nil
}

func (model *Model) getShape(name string) *Shape {
	return model.shapes[name]
}

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

func (model *Model) ToSadl(conf *sadl.Data) (*sadl.Model, error) {
	name := conf.GetString("name")
	if name != "" {
		model.name = name
	} else {
		s := sadl.GetString(model.ast.Metadata, "name")
		if s != "" {
			model.name = s
			delete(model.ast.Metadata, "name")
		}
	}
	service := conf.GetString("service")
	namespace := conf.GetString("namespace")
	if namespace != UnspecifiedNamespace {
		model.namespace = namespace
	}

	annos := make(map[string]string, 0)

	if model.ast.Metadata != nil {
		for k, v := range model.ast.Metadata {
			if k != "name" {
				annos["x_"+k] = sadl.ToString(v) //fix this, should not need to be a string
			}
		}
	}
	//	annos["x_smithy_version"] = model.ast.Version
	schema := &sadl.Schema{
		Name:        model.name,
		Namespace:   model.namespace,
		Version:     model.version,
		Annotations: annos,
	}
	if schema.Namespace == UnspecifiedNamespace {
		schema.Namespace = ""
	}
	if schema.Version == UnspecifiedVersion {
		schema.Version = ""
	}

	haveService := ""
	for shapeName, shapeDef := range model.shapes {
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
		}
	}

	for k, v := range model.shapes {
		if v.Type != "service" || k == haveService {
			model.importShape(schema, k, v)
		}
	}
	return sadl.NewModel(schema)
}

func (model *Model) importShape(schema *sadl.Schema, shapeName string, shapeDef *Shape) {
	switch shapeDef.Type {
	case "byte", "short", "integer", "long", "float", "double", "bigInteger", "bigDecimal":
		model.importNumericShape(schema, shapeDef.Type, shapeName, shapeDef)
	case "string":
		model.importStringShape(schema, shapeName, shapeDef)
	case "timestamp":
		model.importTimestampShape(schema, shapeName, shapeDef)
	case "boolean":
		model.importBooleanShape(schema, shapeName, shapeDef)
	case "list":
		model.importListShape(schema, shapeName, shapeDef, false)
	case "set":
		model.importListShape(schema, shapeName, shapeDef, true)
	case "map":
		model.importMapShape(schema, shapeName, shapeDef)
	case "structure":
		model.importStructureShape(schema, shapeName, shapeDef)
	case "union":
		model.importUnionShape(schema, shapeName, shapeDef)
	case "blob":
		model.importBlobShape(schema, shapeName, shapeDef)
	case "service":
		schema.Name = shapeName
	case "operation":
		model.importOperationShape(schema, shapeName, shapeDef)
	case "resource":
		model.importResourceShape(schema, shapeName, shapeDef)
	default:
		fmt.Println("fix me, unhandled shape type: " + shapeDef.Type)
		panic("whoa")
	}
}

func (model *Model) escapeComment(doc string) string {
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

func (model *Model) importNumericShape(schema *sadl.Schema, smithyType string, shapeName string, shape *Shape) {
	td := &sadl.TypeDef{
		Name:        shapeName,
		Comment:     model.escapeComment(sadl.GetString(shape.Traits, "smithy.api#documentation")),
		Annotations: model.importTraitsAsAnnotations(nil, shape.Traits),
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
	if l := sadl.GetMap(shape.Traits, "smithy.api#range"); l != nil {
		tmp := sadl.GetDecimal(l, "min")
		if tmp != nil {
			td.Min = tmp
		}
		tmp = sadl.GetDecimal(l, "max")
		if tmp != nil {
			td.Max = tmp
		}
	}
	schema.Types = append(schema.Types, td)
}

func (model *Model) importStringShape(schema *sadl.Schema, shapeName string, shape *Shape) {
	if shapeName == "UUID" {
		//UUID is already a builtin SADL type
		return
	}
	if model.importEnum(schema, shapeName, shape) {
		return
	}
	td := &sadl.TypeDef{
		Name:    shapeName,
		Comment: model.escapeComment(sadl.GetString(shape.Traits, "smithy.api#documentation")),
	}
	pat := sadl.GetString(shape.Traits, "smithy.api#pattern")
	if pat == "([a-f0-9]{8}(-[a-f0-9]{4}){4}[a-f0-9]{8})" {
		td.Type = "UUID"
	} else {
		td.Type = "String"
		td.Pattern = pat
		if l := sadl.GetMap(shape.Traits, "smithy.api#length"); l != nil {
			tmp := sadl.GetInt64(l, "min")
			if tmp != 0 {
				td.MinSize = &tmp
			}
			tmp = sadl.GetInt64(l, "max")
			if tmp != 0 {
				td.MaxSize = &tmp
			}
		}
		lst := sadl.GetArray(shape.Traits, "smithy.api#enum")
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
	schema.Types = append(schema.Types, td)
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

func (model *Model) importEnum(schema *sadl.Schema, shapeName string, shape *Shape) bool {
	lst := sadl.GetArray(shape.Traits, "smithy.api#enum")
	if lst == nil {
		return false
	}
	var elements []*sadl.EnumElementDef
	isEnum := true
	couldBeEnum := true
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
	td := &sadl.TypeDef{
		Name:    shapeName,
		Comment: model.escapeComment(sadl.GetString(shape.Traits, "smithy.api#documentation")),
	}
	td.Type = "Enum"
	td.Elements = elements
	schema.Types = append(schema.Types, td)
	return true
}

func (model *Model) importTimestampShape(schema *sadl.Schema, shapeName string, shape *Shape) {
	td := &sadl.TypeDef{
		Name:        shapeName,
		Comment:     model.escapeComment(sadl.GetString(shape.Traits, "smithy.api#documentation")),
		Annotations: model.importTraitsAsAnnotations(nil, shape.Traits),
	}
	td.Type = "Timestamp"
	schema.Types = append(schema.Types, td)
}

func (model *Model) importBlobShape(schema *sadl.Schema, shapeName string, shape *Shape) {
	td := &sadl.TypeDef{
		Name:    shapeName,
		Comment: model.escapeComment(sadl.GetString(shape.Traits, "smithy.api#documentation")),
	}
	td.Type = "Bytes"
	if l := sadl.GetMap(shape.Traits, "smithy.api#length"); l != nil {
		tmp := sadl.GetInt64(l, "min")
		if tmp != 0 {
			td.MinSize = &tmp
		}
		tmp = sadl.GetInt64(l, "max")
		if tmp != 0 {
			td.MaxSize = &tmp
		}
	}
	schema.Types = append(schema.Types, td)
}

func (model *Model) importBooleanShape(schema *sadl.Schema, shapeName string, shape *Shape) {
	td := &sadl.TypeDef{
		Name:        shapeName,
		Comment:     model.escapeComment(sadl.GetString(shape.Traits, "smithy.api#documentation")),
		Annotations: model.importTraitsAsAnnotations(nil, shape.Traits),
	}
	td.Type = "Boolean"
	schema.Types = append(schema.Types, td)
}

func (model *Model) importTraitsAsAnnotations(annos map[string]string, traits map[string]interface{}) map[string]string {
	for k, v := range traits {
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
		case "smithy.api#readonly", "smithy.api#idempotent":
			//			annos = WithAnnotation(annos, "x_"+stripNamespace(k), "true")
		case "smithy.api#http":
			/* ignore, handled elsewhere */
		case "smithy.api#timestampFormat":
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
		case "aws.protocols#restJson1":
			//ignore
		case "smithy.api#examples":
			//ignore
		default:
			fmt.Println("Unhandled trait:", k, " =", sadl.Pretty(v))
			panic("here: " + k)
		}
	}
	return annos
}

func (model *Model) importUnionShape(schema *sadl.Schema, shapeName string, shape *Shape) {
	td := &sadl.TypeDef{
		Name:        shapeName,
		Comment:     model.escapeComment(sadl.GetString(shape.Traits, "smithy.api#documentation")),
		Annotations: model.importTraitsAsAnnotations(nil, shape.Traits),
	}
	td.Type = "Union"
	//	prefix := model.namespace + "#"
	for memberName, member := range shape.Members { //this order is not deterministic, because map
		//		if memberName != member.Target {
		vd := &sadl.UnionVariantDef{
			Name:        memberName,
			Comment:     model.escapeComment(sadl.GetString(member.Traits, "smithy.api#documentation")),
			Annotations: model.importTraitsAsAnnotations(nil, member.Traits),
		}
		vd.Type = model.shapeRefToTypeRef(schema, member.Target)
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
	schema.Types = append(schema.Types, td)
}

func (model *Model) importStructureShape(schema *sadl.Schema, shapeName string, shape *Shape) {
	//unless...no httpTraits, in which case is should equivalent to a payload description
	if _, ok := model.ioParams[shapeName]; ok {
		shape := model.getShape(shapeName)
		for _, fval := range shape.Members { //this order is not deterministic, because map
			if sadl.GetBool(fval.Traits, "smithy.api#httpLabel") {
				return
			}
			if sadl.GetBool(fval.Traits, "smithy.api#httpPayload") {
				return
			}
			if sadl.GetString(fval.Traits, "smithy.api#httpQuery") != "" {
				return
			}
		}
	}
	td := &sadl.TypeDef{
		Name:        shapeName,
		Comment:     model.escapeComment(sadl.GetString(shape.Traits, "smithy.api#documentation")),
		Annotations: model.importTraitsAsAnnotations(nil, shape.Traits),
	}
	td.Type = "Struct"
	var fields []*sadl.StructFieldDef
	for memberName, member := range shape.Members { //this order is not deterministic, because map
		fd := &sadl.StructFieldDef{
			Name:        memberName,
			Comment:     model.escapeComment(sadl.GetString(member.Traits, "smithy.api#documentation")),
			Annotations: model.importTraitsAsAnnotations(nil, member.Traits),
			Required:    sadl.GetBool(member.Traits, "smithy.api#required"),
		}
		fd.Type = model.shapeRefToTypeRef(schema, member.Target)
		if sadl.GetBool(member.Traits, "smithy.api#required") {
			fd.Required = true
		}
		fields = append(fields, fd)
	}
	td.Fields = fields
	schema.Types = append(schema.Types, td)
}

func (model *Model) importListShape(schema *sadl.Schema, shapeName string, shape *Shape, unique bool) {
	td := &sadl.TypeDef{
		Name:    shapeName,
		Comment: model.escapeComment(sadl.GetString(shape.Traits, "smithy.api#documentation")),
	}
	td.Type = "Array"
	td.Items = model.shapeRefToTypeRef(schema, shape.Member.Target)
	tmp := sadl.GetInt64(shape.Traits, "smithy.api#min")
	if tmp != 0 {
		td.MinSize = &tmp
	}
	tmp = sadl.GetInt64(shape.Traits, "smithy.api#max")
	if tmp != 0 {
		td.MaxSize = &tmp
	}
	if unique {
		td.Annotations = WithAnnotation(td.Annotations, "x_unique", "true")
	}
	schema.Types = append(schema.Types, td)
}

func (model *Model) importMapShape(schema *sadl.Schema, shapeName string, shape *Shape) {
	td := &sadl.TypeDef{
		Name:    shapeName,
		Comment: model.escapeComment(sadl.GetString(shape.Traits, "smithy.api#documentation")),
	}
	td.Type = "Map"
	td.Keys = model.shapeRefToTypeRef(schema, shape.Key.Target)
	td.Items = model.shapeRefToTypeRef(schema, shape.Value.Target)
	tmp := sadl.GetInt64(shape.Traits, "smithy.api#min")
	if tmp != 0 {
		td.MinSize = &tmp
	}
	tmp = sadl.GetInt64(shape.Traits, "smithy.api#max")
	if tmp != 0 {
		td.MaxSize = &tmp
	}
	schema.Types = append(schema.Types, td)
}

func (model *Model) ensureLocalNamespace(id string) string {
	if strings.Index(id, "#") < 0 {
		return id //already local
	}
	prefix := model.namespace + "#"
	if strings.HasPrefix(id, prefix) {
		return id[len(prefix):]
	}
	return ""
}

func (model *Model) shapeRefToTypeRef(schema *sadl.Schema, shapeRef string) string {
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
		ltype := model.ensureLocalNamespace(typeRef)
		if ltype == "" {
			panic("external namespace type refr not supported: " + typeRef)
		}
		typeRef = ltype
	}
	return typeRef
}

func (model *Model) importResourceShape(schema *sadl.Schema, shapeName string, shape *Shape) {
	//to do: preserve the resource info as tags on the operations
}

func (model *Model) importOperationShape(schema *sadl.Schema, shapeName string, shape *Shape) {
	var method, uri string
	var code int
	if t, ok := shape.Traits["smithy.api#http"]; ok {
		switch ht := t.(type) {
		case map[string]interface{}:
			method = sadl.GetString(ht, "method")
			uri = sadl.GetString(ht, "uri")
			code = sadl.GetInt(ht, "code")
		default:
			fmt.Println("?", sadl.Pretty(shape))
		}
	}
	if method == "" {
		panic("non-http actions NYI")
	}
	var inType string
	if shape.Input != nil {
		inType = model.shapeRefToTypeRef(schema, shape.Input.Target)
	}
	var outType string
	if shape.Output != nil {
		outType = model.shapeRefToTypeRef(schema, shape.Output.Target)
	}
	reqType := sadl.Capitalize(shapeName) + "Request"
	resType := sadl.Capitalize(shapeName) + "Response"
	if t, ok := shape.Traits["smithy.api#examples"]; ok {
		if opexes, ok := t.([]interface{}); ok {
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
						schema.Examples = append(schema.Examples, ex)
					}
					if output, ok := opex["output"]; ok {
						ex := &sadl.ExampleDef{
							Target:  resType,
							Name:    title,
							Example: output,
						}
						schema.Examples = append(schema.Examples, ex)
					}
				}
			}
		}
	}

	hdef := &sadl.HttpDef{
		Method:      method,
		Path:        uri,
		Name:        sadl.Uncapitalize(shapeName),
		Comment:     model.escapeComment(sadl.GetString(shape.Traits, "smithy.api#documentation")),
		Annotations: model.importTraitsAsAnnotations(nil, shape.Traits),
	}
	if code == 0 {
		code = 200
	}
	if shape.Input != nil {
		inStruct := model.shapes[inType]
		qs := ""
		payloadMember := ""
		hasLabel := false
		hasQuery := false
		for fname, fval := range inStruct.Members { //this order is not deterministic, because map
			if sadl.GetBool(fval.Traits, "smithy.api#httpPayload") {
				payloadMember = fname
			} else if sadl.GetBool(fval.Traits, "smithy.api#httpLabel") {
				hasLabel = true
			} else if sadl.GetBool(fval.Traits, "smithy.api#httpQuery") {
				hasQuery = true
			}
		}
		if hasLabel || hasQuery || payloadMember != "" {
			//the input might *have* a body
			for fname, fval := range inStruct.Members { //this order is not deterministic, because map
				in := &sadl.HttpParamSpec{}
				in.Name = fname
				in.Type = model.shapeRefToTypeRef(schema, fval.Target)
				in.Required = sadl.GetBool(fval.Traits, "smithy.api#required")
				in.Query = sadl.GetString(fval.Traits, "smithy.api#httpQuery")
				if in.Query != "" {
					if qs == "" {
						qs = "?"
					} else {
						qs = qs + "&"
					}
					qs = qs + fname + "={" + fname + "}"
				}
				in.Header = sadl.GetString(fval.Traits, "smithy.api#httpHeader")
				in.Path = sadl.GetBool(fval.Traits, "smithy.api#httpLabel")
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

	expected := &sadl.HttpExpectedSpec{
		Status: int32(code),
	}
	if shape.Output != nil {
		outStruct := model.shapes[outType]
		//SADL: each output is a header or a (singular) payload.
		//Smithy: the output struct is the result payload, unless a field is marked as payload, which allows other fields
		//to be marked as header.
		outBodyField := ""
		hasLabel := false
		for fname, fval := range outStruct.Members { //this order is not deterministic, because map
			if sadl.GetBool(fval.Traits, "smithy.api#httpPayload") {
				outBodyField = fname
			} else if sadl.GetBool(fval.Traits, "smithy.api#httpLabel") {
				hasLabel = true
			}
		}
		if outBodyField == "" && !hasLabel {
			//the entire output structure is the payload, no headers possible
			out := &sadl.HttpParamSpec{}
			out.Name = "body"
			out.Type = model.shapeRefToTypeRef(schema, outType)
			expected.Outputs = append(expected.Outputs, out)
		} else {
			for fname, fval := range outStruct.Members { //this order is not deterministic, because map
				out := &sadl.HttpParamSpec{}
				out.Name = fname
				out.Type = model.shapeRefToTypeRef(schema, fval.Target)
				out.Required = sadl.GetBool(fval.Traits, "smithy.api#required")
				out.Header = sadl.GetString(fval.Traits, "smithy.api#httpHeader")
				out.Query = sadl.GetString(fval.Traits, "smithy.api#httpQuery")
				out.Path = sadl.GetBool(fval.Traits, "smithy.api#httpLabel")
				expected.Outputs = append(expected.Outputs, out)
			}
		}
	}
	if shape.Errors != nil {
		for _, etype := range shape.Errors {
			eType := model.shapeRefToTypeRef(schema, etype.Target)
			eStruct := model.shapes[eType]
			if eStruct == nil {
				panic("error type not found")
			}
			exc := &sadl.HttpExceptionSpec{}
			exc.Type = eType
			exc.Status = int32(sadl.GetInt(eStruct.Traits, "smithy.api#httpError"))
			exc.Comment = model.escapeComment(sadl.GetString(eStruct.Traits, "smithy.api#documentation"))
			//preserve other traits as annotations?
			hdef.Exceptions = append(hdef.Exceptions, exc)
		}
	}
	//Comment string
	//Annotations map[string]string
	hdef.Expected = expected
	schema.Http = append(schema.Http, hdef)
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
