package smithy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/util"
)

func Import(path string, conf map[string]interface{}) (*sadl.Model, error) {
	var ast *AST
	var err error
	name := getString(conf, "name")
	if name == "" {
		conf["name"] = nameFromPath(path)
	}
	namespace := getString(conf, "namespace")
	if namespace == "" {
		conf["namespace"] = UnspecifiedNamespace
	}
	if strings.HasSuffix(path, ".json") {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("Cannot read source file: %v\n", err)
		}
		//smithy AST
		err = json.Unmarshal(data, &ast)
		if err != nil {
			return nil, fmt.Errorf("Cannot parse Smithy AST file: %v\n", err)
		}
		if ast.Version == "" {
			return nil, fmt.Errorf("Cannot parse Smithy AST file: %v\n", err)
		}
	} else {
		//parse Smithy IDL
		ast, err = parse(path)
		if err != nil {
			return nil, err
		}
	}
	return NewModel(ast).ToSadl(conf)
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

func (model *Model) getShape(name string) *Shape {
	return model.shapes[name]
}

func NewModel(ast *AST) *Model {
	model := &Model{
		ast: ast,
	}
	model.shapes = make(map[string]*Shape, 0)
	model.ioParams = make(map[string]string, 0)
	model.namespace, model.name, model.version = ast.NamespaceAndServiceVersion()

	if model.name == "" {
		s := getString(ast.Metadata, "name")
		if s != "" {
			model.name = s
		}
	}

	prefix := model.namespace + "#"
	prefixLen := len(prefix)
	for k, v := range ast.Shapes {
		if strings.HasPrefix(k, prefix) {
			kk := k[prefixLen:]
			model.shapes[kk] = v
			if v.Type == "operation" {
				if v.Input != nil {
					iok := v.Input.Target[prefixLen:]
					model.ioParams[iok] = v.Input.Target
				}
				if v.Output != nil {
					iok := v.Output.Target[prefixLen:]
					model.ioParams[iok] = v.Output.Target
				}
			}
		}
	}
	return model
}

func (model *Model) ToSadl(conf map[string]interface{}) (*sadl.Model, error) {
	name := getString(conf, "name")
	if name != "" {
		model.name = name
	} else {
		s := getString(model.ast.Metadata, "name")
		if s != "" {
			model.name = s
			delete(model.ast.Metadata, "name")
		}
	}
	namespace := getString(conf, "namespace")
	if namespace != UnspecifiedNamespace {
		model.namespace = namespace
	}

	annos := make(map[string]string, 0)

	if model.ast.Metadata != nil {
		for k, v := range model.ast.Metadata {
			if k != "name" {
				annos["x_"+k] = util.ToString(v) //fix this, should not need to be a string
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

	//	fmt.Println("shapes in our namespace:", util.Pretty(model.shapes))
	for k, v := range model.shapes {
		model.importShape(schema, k, v)
	}
	return sadl.NewModel(schema)
}

func (model *Model) importShape(schema *sadl.Schema, shapeName string, shapeDef *Shape) {
	switch shapeDef.Type {
	case "byte", "short", "integer", "long", "float", "double", "bigInteger", "bigDecimal":
		model.importNumericShape(schema, shapeDef.Type, shapeName, shapeDef)
	case "string":
		model.importStringShape(schema, shapeName, shapeDef)
	case "list":
		model.importListShape(schema, shapeName, shapeDef)
	case "structure":
		model.importStructureShape(schema, shapeName, shapeDef)
	case "union":
		model.importUnionShape(schema, shapeName, shapeDef)
	case "service":
		schema.Name = shapeName
	case "operation":
		model.importOperationShape(schema, shapeName, shapeDef)
	default:
		fmt.Println("fix me, unhandled shape type: " + shapeDef.Type)
		panic("whoa")
	}
}

func (model *Model) importNumericShape(schema *sadl.Schema, smithyType string, shapeName string, shape *Shape) {
	fmt.Println("importNumericShape:", smithyType)
	td := &sadl.TypeDef{
		Name:        shapeName,
		Comment:     getString(shape.Traits, "smithy.api#documentation"),
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
		td.Type = "Int65"
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
	if l := getStruct(shape.Traits, "smithy.api#range"); l != nil {
		tmp := getDecimal(l, "min")
		if tmp != nil {
			td.Min = tmp
		}
		tmp = getDecimal(l, "max")
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
		Comment: getString(shape.Traits, "smithy.api#documentation"),
	}
	td.Type = "String"
	td.Pattern = getString(shape.Traits, "smithy.api#pattern")
	if l := getStruct(shape.Traits, "smithy.api#length"); l != nil {
		tmp := getInt64(l, "min")
		if tmp != 0 {
			td.MinSize = &tmp
		}
		tmp = getInt64(l, "max")
		if tmp != 0 {
			td.MaxSize = &tmp
		}
	}
	lst := getArray(shape.Traits, "smithy.api#enum")
	if lst != nil {
		isActualEnum := true
		for _, v := range lst {
			m := asStruct(v)
			if get(m, "name") == nil {
				isActualEnum = false
				break
			}
		}
		if isActualEnum {
		}
		fmt.Println("enum:", util.Pretty(lst))
		var values []string
		for _, v := range lst {
			m := asStruct(v)
			fmt.Println("-->", getString(m, "value"), "<--")
			//			m, _ := v.(map[string]interface{})
			//			if _, ok := m["name"]; !ok {
			//				panic("should be a true enum, not a string with values")
			//			}
			//			s, _ := m["value"].(string)
			s := getString(m, "value")
			fmt.Println("  m, s", util.Pretty(m), s)
			values = append(values, s)
		}
		td.Values = values
	}
	//	}
	schema.Types = append(schema.Types, td)
}
func (model *Model) importEnum(schema *sadl.Schema, shapeName string, shape *Shape) bool {
	lst := getArray(shape.Traits, "smithy.api#enum")
	if lst == nil {
		return false
	}
	var elements []*sadl.EnumElementDef
	for _, v := range lst {
		m := asStruct(v)
		sym := getString(m, "name")
		if sym == "" {
			return false
		}
		element := &sadl.EnumElementDef{
			Symbol:  sym,
			Comment: getString(m, "documentation"),
		}
		//tags -> annotations?
		elements = append(elements, element)
	}
	td := &sadl.TypeDef{
		Name:    shapeName,
		Comment: getString(shape.Traits, "smithy.api#documentation"),
	}
	td.Type = "Enum"
	td.Elements = elements
	schema.Types = append(schema.Types, td)
	return true
}

func (model *Model) importTraitsAsAnnotations(annos map[string]string, traits map[string]interface{}) map[string]string {
	for k, v := range traits {
		switch k {
		case "smithy.api#error":
			annos = WithAnnotation(annos, "x_"+stripNamespace(k), asString(v))
		case "smithy.api#httpError":
			annos = WithAnnotation(annos, "x_"+stripNamespace(k), fmt.Sprintf("%v", v))
		case "smithy.api#httpPayload", "smithy.api#httpLabel", "smithy.api#httpQuery", "smithy.api#httpHeader":
			/* ignore, implicit in SADL */
		case "smithy.api#required", "smithy.api#documentation", "smithy.api#range", "smithy.api#length":
			/* ignore, implicit in SADL */
		default:
			fmt.Println("Unhandled struct member trait:", k, " =", v)
			panic("here")
		}
	}
	return annos
}

func (model *Model) importUnionShape(schema *sadl.Schema, shapeName string, shape *Shape) {
	td := &sadl.TypeDef{
		Name:        shapeName,
		Comment:     getString(shape.Traits, "smithy.api#documentation"),
		Annotations: model.importTraitsAsAnnotations(nil, shape.Traits),
	}
	td.Type = "Union"
	//	prefix := model.namespace + "#"
	for memberName, member := range shape.Members {
		//		if memberName != member.Target {
		vd := &sadl.UnionVariantDef{
			Name:        memberName,
			Comment:     getString(member.Traits, "smithy.api#documentation"),
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
	if _, ok := model.ioParams[shapeName]; ok {
		return
	}
	td := &sadl.TypeDef{
		Name:        shapeName,
		Comment:     getString(shape.Traits, "smithy.api#documentation"),
		Annotations: model.importTraitsAsAnnotations(nil, shape.Traits),
	}
	td.Type = "Struct"
	var fields []*sadl.StructFieldDef
	for memberName, member := range shape.Members {
		fd := &sadl.StructFieldDef{
			Name:        memberName,
			Comment:     getString(member.Traits, "smithy.api#documentation"),
			Annotations: model.importTraitsAsAnnotations(nil, member.Traits),
			Required:    getBool(member.Traits, "smithy.api#required"),
		}
		fd.Type = model.shapeRefToTypeRef(schema, member.Target)
		if getBool(member.Traits, "smithy.api#required") {
			fd.Required = true
		}
		fields = append(fields, fd)
	}
	td.Fields = fields
	schema.Types = append(schema.Types, td)
}

func (model *Model) importListShape(schema *sadl.Schema, shapeName string, shape *Shape) {
	td := &sadl.TypeDef{
		Name:    shapeName,
		Comment: getString(shape.Traits, "smithy.api#documentation"),
	}
	td.Type = "Array"
	td.Items = model.shapeRefToTypeRef(schema, shape.Member.Target)
	tmp := getInt64(shape.Traits, "smithy.api#min")
	if tmp != 0 {
		td.MinSize = &tmp
	}
	tmp = getInt64(shape.Traits, "smithy.api#max")
	if tmp != 0 {
		td.MaxSize = &tmp
	}
	schema.Types = append(schema.Types, td)
}

func (model *Model) shapeRefToTypeRef(schema *sadl.Schema, shapeRef string) string {
	prefix := model.namespace + "#"
	typeRef := shapeRef
	if strings.HasPrefix(typeRef, prefix) {
		typeRef = typeRef[len(prefix):]
	} else {
		switch typeRef {
		case "smithy.api#Blob", "Blob":
			return "Blob"
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
			panic("no:" + typeRef)
		}
	}
	//assume the type is defined
	return typeRef
}

func (model *Model) importOperationShape(schema *sadl.Schema, shapeName string, shape *Shape) {
	//	fmt.Println(shapeName, util.Pretty(shape))
	var method, uri string
	var code int
	if t, ok := shape.Traits["smithy.api#http"]; ok {
		switch ht := t.(type) {
		case map[string]interface{}:
			method = getString(ht, "method")
			uri = getString(ht, "uri")
			code = getInt(ht, "code")
			/*		case *HttpTrait:
					method = ht.Method
					uri = ht.Uri
					code = ht.Code
			*/
		default:
			fmt.Println("?", util.Pretty(shape))
		}
	}
	if method == "" {
		panic("non-http actions NYI")
	}

	hdef := &sadl.HttpDef{
		Method:  method,
		Path:    uri,
		Name:    shapeName,
		Comment: getString(shape.Traits, "smithy.api#documentation"),
	}
	if code == 0 {
		code = 200
	}
	if shape.Input != nil {
		inType := model.shapeRefToTypeRef(schema, shape.Input.Target)
		inStruct := model.shapes[inType]
		qs := ""
		for fname, fval := range inStruct.Members {
			in := &sadl.HttpParamSpec{}
			in.Name = fname
			in.Type = model.shapeRefToTypeRef(schema, fval.Target)
			if in.Type == "Integer" {
				panic("here")
			}
			in.Required = getBool(fval.Traits, "smithy.api#required")
			in.Query = getString(fval.Traits, "smithy.api#httpQuery")
			if in.Query != "" {
				if qs == "" {
					qs = "?"
				} else {
					qs = qs + "&"
				}
				qs = qs + fname + "={" + fname + "}"
			}
			in.Header = getString(fval.Traits, "smithy.api#httpHeader")
			in.Path = getBool(fval.Traits, "smithy.api#httpLabel")
			hdef.Inputs = append(hdef.Inputs, in)
		}
		hdef.Path = hdef.Path + qs
	}

	expected := &sadl.HttpExpectedSpec{
		Status: int32(code),
	}
	if shape.Output != nil {
		outType := model.shapeRefToTypeRef(schema, shape.Output.Target)
		outStruct := model.shapes[outType]
		//SADL: each output is a header or a (singular) payload.
		//Smithy: the output struct is the result payload, unless a field is marked as payload, which allows other fields
		//to be marked as header.
		outBodyField := ""
		for fname, fval := range outStruct.Members {
			if getBool(fval.Traits, "smithy.api#httpPayload") {
				outBodyField = fname
				break
			}
		}
		if outBodyField == "" {
			//the entire output structure is the payload, no headers possible
			out := &sadl.HttpParamSpec{}
			out.Name = "body"
			out.Type = model.shapeRefToTypeRef(schema, outType)
			expected.Outputs = append(expected.Outputs, out)
		} else {
			for fname, fval := range outStruct.Members {
				out := &sadl.HttpParamSpec{}
				out.Name = fname
				out.Type = model.shapeRefToTypeRef(schema, fval.Target)
				out.Required = getBool(fval.Traits, "smithy.api#required")
				out.Header = getString(fval.Traits, "smithy.api#httpHeader")
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
			exc.Status = int32(getInt(eStruct.Traits, "smithy.api#httpError"))
			exc.Comment = getString(eStruct.Traits, "smithy.api#documentation")
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
