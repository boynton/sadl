package oas

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/oas/oas2"
	"github.com/boynton/sadl/oas/oas3"
	"github.com/ghodss/yaml"
)

type Oas struct {
	V3 *oas3.OpenAPI
}

func (oas *Oas) MarshalJSON() ([]byte, error) {
	return json.Marshal(oas.V3)
}

func DetermineVersion(data []byte, format string) (string, error) {
	var raw map[string]interface{}
	var err error
	switch format {
	case "json":
		err = json.Unmarshal(data, &raw)
	case "yaml":
		err = yaml.Unmarshal(data, &raw)
	default:
		err = fmt.Errorf("Unsupported file format: %q. Only \"json\" and \"yaml\" are supported.", format)
	}
	if err != nil {
		return "", err
	}
	if v, ok := raw["openapi"]; ok {
		if s, ok := v.(string); ok {
			return s, nil
		}
	}
	if v, ok := raw["swagger"]; ok {
		if s, ok := v.(string); ok {
			return s, nil
		}
	}
	return "", fmt.Errorf("Cannot find an 'openapi' in the specified %s file to determine the version", format)
}

func Parse(data []byte, format string) (*Oas, error) {
	version, err := DetermineVersion(data, format)
	if err != nil {
		return nil, err
	}
	oas := &Oas{}
	if strings.HasPrefix(version, "3.") {
		oas.V3, err = oas3.Parse(data, format)
		return oas, nil
	} else if strings.HasPrefix(version, "2.") {
		v2, err := oas2.Parse(data, format)
		if err == nil {
			oas.V3, err = oas2.ConvertToV3(v2)
		}
		return oas, err
	}
	return nil, fmt.Errorf("Unsupported version of OpenAPI Spec: %s", version)
}

var examples []*sadl.ExampleDef

var methods = []string{"GET", "PUT", "POST", "DELETE", "DELETE", "HEAD"} //to do: "PATCH", "OPTIONS", "TRACE"

func (oas *Oas) ToSadl(name string) (*sadl.Model, error) {
	comment := oas.V3.Info.Title
	if sadl.IsSymbol(comment) {
		name = comment
	}
	if oas.V3.Info.Description != "" {
		comment = comment + " - " + oas.V3.Info.Description
	}
	schema := &sadl.Schema{
		Name:    name,
		Comment: comment,
		Version: oas.V3.Info.Version,
	}
	for name, oasSchema := range oas.V3.Components.Schemas {
		var ts sadl.TypeSpec
		var err error
		comment := ""
		tname := oasTypeRef(oasSchema)
		if tname != "" {
			if oasDef, ok := oas.V3.Components.Schemas[tname]; ok {
				ts, err = convertOasType(tname, oasDef) //doesn't handle N levels
			} else {
				panic("hmm")
			}
		} else {
			ts, err = convertOasType(name, oasSchema)
			comment = oasSchema.Description
		}

		if err != nil {
			return nil, err
		}
		td := &sadl.TypeDef{
			TypeSpec: ts,
			Name:     name,
			Comment:  comment,
			//annotations
		}
		schema.Types = append(schema.Types, td)
	}

	httpBindings := true
	actions := false
	for tmpl, path := range oas.V3.Paths {
		for _, method := range methods {
			op := getPathOperation(path, method)
			if op != nil {
				if strings.HasPrefix(tmpl, "x-") {
					continue
				}
				if actions {
					act, err := convertOasPathToAction(schema, op, method)
					if err != nil {
						return nil, err
					}
					schema.Actions = append(schema.Actions, act)
				}
				if httpBindings {
					hact, err := convertOasPath(tmpl, op, method)
					if err != nil {
						return nil, err
					}
					schema.Http = append(schema.Http, hact)
					//fmt.Println(tmpl, sadl.Pretty(path))
				}
			}
		}
	}
	for _, server := range oas.V3.Servers {
		if schema.Annotations == nil {
			schema.Annotations = make(map[string]string, 0)
		}
		schema.Annotations["x_server"] = server.URL
	}
	if oas.V3.Info.License != nil {
		if schema.Annotations == nil {
			schema.Annotations = make(map[string]string, 0)
		}
		if oas.V3.Info.License.Name != "" {
			schema.Annotations["x_license_name"] = oas.V3.Info.License.Name
		}
		if oas.V3.Info.License.URL != "" {
			schema.Annotations["x_license_url"] = oas.V3.Info.License.URL
		}
	}

	schema.Examples = examples
	return sadl.NewModel(schema)
}

func oasTypeRef(oasSchema *oas3.Schema) string {
	if oasSchema.Ref != "" {
		if strings.HasPrefix(oasSchema.Ref, "#/components/schemas/") {
			return oasSchema.Ref[len("#/components/schemas/"):]
		}
		return oasSchema.Ref //?
	}
	return ""
}

func convertOasType(name string, oasSchema *oas3.Schema) (sadl.TypeSpec, error) {
	var err error
	var ts sadl.TypeSpec
	switch oasSchema.Type {
	case "boolean":
		ts.Type = "Bool"
	case "string":
		if oasSchema.Enum != nil {
			//OAS defines element *descriptions* as the values, not symbolic identifiers.
			//so we look for the case where all values look like identifiers, and call that an enum. Else a strings with accepted "values"
			//perhaps the spirit of JSON Schema enums are just values, not what I think of as "enums", i.e. "a set of named values", per wikipedia.
			//still, with symbolic values, perhaps the intent is to use proper enums, if only JSON Schema had them.
			wantEnums := true //set to true to opportunistically try to make then real enums. If false, everything is a "value" of a string instead
			isEnum := wantEnums
			var values []string
			for _, val := range oasSchema.Enum {
				if s, ok := val.(string); ok {
					values = append(values, s)
					if !sadl.IsSymbol(s) {
						isEnum = false
					}
				} else {
					return ts, fmt.Errorf("Error in OAS source: string enum value is not a string: %v", val)
				}
			}
			if isEnum {
				ts.Type = "Enum"
				for _, sym := range values {
					el := &sadl.EnumElementDef{
						Symbol: sym,
					}
					ts.Elements = append(ts.Elements, el)
				}
			} else {
				ts.Type = "String"
				ts.Values = values
			}
		} else {
			ts.Type = "String"
		}
		if ts.Type == "String" {
			if oasSchema.Example != nil {
				ex := &sadl.ExampleDef{
					Target:  name,
					Example: oasSchema.Example,
				}
				examples = append(examples, ex)
			}
			ts.Pattern = oasSchema.Pattern
			if oasSchema.MinLength > 0 {
				tmpMin := int64(oasSchema.MinLength)
				ts.MinSize = &tmpMin
			}
			if oasSchema.MaxLength != nil {
				tmpMax := int64(*oasSchema.MaxLength)
				ts.MaxSize = &tmpMax
			}
			if oasSchema.Format != "" {
				fmt.Println("NYI: String 'format':", oasSchema.Format)
			}
		}
	case "array":
		ts.Type = "Array"
		if oasSchema.Items != nil {
			if oasSchema.Items.Ref != "" {
				ts.Items = oasTypeRef(oasSchema.Items)
			} else {
				its, err := convertOasType(name+".Items", oasSchema.Items)
				if err == nil {
					ts.Items = its.Type
				}
			}
		}
		//minsize, maxsize
		//comment
	case "number":
		ts.Type = "Decimal"
		if oasSchema.Min != nil {
			ts.Min = sadl.NewDecimal(*oasSchema.Min)
		}
		if oasSchema.Max != nil {
			ts.Max = sadl.NewDecimal(*oasSchema.Max)
		}
	case "integer":
		switch oasSchema.Format {
		case "int8":
			ts.Type = "Int8"
		case "int16":
			ts.Type = "Int16"
		case "int32":
			ts.Type = "Int32"
		case "int64":
			ts.Type = "Int64"
		default:
			ts.Type = "Int64"
		}
		if oasSchema.Min != nil {
			ts.Min = sadl.NewDecimal(*oasSchema.Min)
		}
		if oasSchema.Max != nil {
			ts.Max = sadl.NewDecimal(*oasSchema.Max)
		}
	case "", "object":
		if oasSchema.Properties != nil {
			ts.Type = "Struct"
			req := oasSchema.Required
			for fname, fschema := range oasSchema.Properties {
				fd := &sadl.StructFieldDef{
					Name:    fname,
					Comment: fschema.Description,
				}
				if containsString(req, fname) {
					fd.Required = true
				}
				fd.Type = oasTypeRef(fschema)
				if fd.Type == "" {
					fd.TypeSpec, err = convertOasType(name+"."+fname, fschema)
				}
				ts.Fields = append(ts.Fields, fd)
			}
		}
	default:
		fmt.Printf("oas type is %q\n", oasSchema.Type)
		panic("oas type not handled")
	}
	return ts, err
}

func containsString(lst []string, val string) bool {
	for _, s := range lst {
		if s == val {
			return true
		}
	}
	return false
}

func capitalize(s string) string {
	return strings.ToUpper(s[0:1]) + s[1:]
}

func uncapitalize(s string) string {
	return strings.ToLower(s[0:1]) + s[1:]
}

func makeIdentifier(text string) string {
	reg, _ := regexp.Compile("[^a-zA-Z_][^a-zA-Z_0-9]*")
	return reg.ReplaceAllString(text, "")
}

func convertOasPathToAction(schema *sadl.Schema, op *oas3.Operation, method string) (*sadl.ActionDef, error) {
	name := op.OperationID
	synthesizedName := guessOperationName(op, method)
	if name == "" {
		if synthesizedName == "" {
			synthesizedName = method + "Something" //!
		}
		name = synthesizedName
	}
	act := &sadl.ActionDef{
		Name:    name,
		Comment: op.Summary,
	}
	//need a single input. Generate the op.OperationID
	reqTypeName := capitalize(name) + "Request"

	td := &sadl.TypeDef{
		Name: reqTypeName,
		TypeSpec: sadl.TypeSpec{
			Type: "Struct",
		},
	}
	for _, param := range op.Parameters {
		name := makeIdentifier(param.Name)
		fd := &sadl.StructFieldDef{
			Name:     name,
			Comment:  param.Description,
			Required: param.Required,
		}
		if param.Schema != nil {
			fd.Type = oasTypeRef(param.Schema)
			if fd.Type == "" {
				fd.Type = sadlPrimitiveType(param.Schema.Type)
				if fd.Type == "Array" {
					if param.Schema.Items == nil {
						fd.Items = "Any"
					} else {
						//fmt.Println("item type:", sadl.Pretty(param.Schema.Items))
						//					if param.Schema.Items.Ref != "" {
						//					} else {
						schref := param.Schema.Items
						switch schref.Type {
						case "string":
							fd.Items = "String"
						default:
							fd.Items = "Any"
						}
						//					}
					}
				}
				if param.Schema.Enum != nil {
					for _, val := range param.Schema.Enum {
						if s, ok := val.(string); ok {
							fd.Values = append(fd.Values, s)
						} else {
							return nil, fmt.Errorf("String enum values are not strings: %v", param.Schema.Enum)
						}
					}
				}
			}
		}
		if fd.Type == "Struct" {
			panic("whoops, that isn't right")
		} else {
			if fd.Type == "Array" {
				if fd.Items == "" {
					panic("nope")
				}
			}
		}
		td.Fields = append(td.Fields, fd)
	}

	td2 := findTypeDef(schema, reqTypeName)
	if td2 == nil {
		schema.Types = append(schema.Types, td)
	} else {
		fmt.Println(reqTypeName, "already defined as", sadl.Pretty(td2), "Would have replaced with ", sadl.Pretty(td))
	}
	act.Input = reqTypeName

	//need the output and exceptions now
	expectedStatus := guessDefaultResponseCode(op)
	expectedType := ""
	if expectedStatus != "" {
		resp := op.Responses[expectedStatus]
		if resp == nil {
			resp = op.Responses["default"]
		}
		expectedType = responseTypeName(resp)
		act.Output = expectedType
	}
	resTypeName := capitalize(name) + "Response"
	if findTypeDef(schema, resTypeName) == nil { //ugh
		td = &sadl.TypeDef{
			Name: resTypeName,
			TypeSpec: sadl.TypeSpec{
				Type: "Struct",
			},
		}
		fd := &sadl.StructFieldDef{
			Name: uncapitalize(resTypeName),
		}
		fd.Type = act.Output
		td.Fields = append(td.Fields, fd)
		//header responses?
		schema.Types = append(schema.Types, td)
	}
	act.Output = resTypeName
	excepts := make(map[string][]int32, 0)
	for status, param := range op.Responses {
		if status != expectedStatus {
			respType := responseTypeName(param)
			if respType != expectedType {
				code, _ := strconv.Atoi(status)
				lst, _ := excepts[respType]
				excepts[respType] = append(lst, int32(code))
			}
		}
	}
	for etype, _ := range excepts {
		act.Exceptions = append(act.Exceptions, etype)
	}
	return act, nil
}

func convertOasPath(path string, op *oas3.Operation, method string) (*sadl.HttpDef, error) {
	hact := &sadl.HttpDef{
		Name:    op.OperationID,
		Path:    path,
		Method:  method,
		Comment: op.Summary,
	}
	if len(op.Tags) > 0 {
		hact.Annotations = make(map[string]string, 0)
		
		tmp := ""
		for _, tag := range op.Tags {
			if tmp == "" {
				tmp = tag
			} else {
				tmp = tmp + "," + tag
			}
		}
		hact.Annotations["x_tags"] = tmp
	}

	var queries []string
	for _, param := range op.Parameters {
		name := makeIdentifier(param.Name)
		spec := &sadl.HttpParamSpec{
			StructFieldDef: sadl.StructFieldDef{
				Name:     name,
				Comment:  param.Description,
				Required: param.Required,
			},
		}
		//FIXME: the p.Name is the "source name". There is no formal parameter name defined, although for pathparam, the p.Name acts like it.
		//So, a formal param name (for codegen) must be synthesized, and must be an Indentifier
		switch param.In {
		case "query":
			spec.Query = param.Name
			queries = append(queries, param.Name+"={"+name+"}")
		case "path":
			spec.Path = true
			if strings.Index(path, "{"+name+"}") < 0 {
				fmt.Println("WARNING: path param is not in path template:", path, name)
				panic("here")
			}
		case "header":
			spec.Header = param.Name
		case "cookie":
			return nil, fmt.Errorf("Cookie params NYI: %v", sadl.AsString(param))
		}
		spec.Type = oasTypeRef(param.Schema)
		if spec.Type == "" {
			spec.Type = sadlPrimitiveType(param.Schema.Type)
			if spec.Type == "Array" {
				if param.Schema.Items == nil {
					spec.Items = "Any"
				} else {
					schref := param.Schema.Items
					switch schref.Type {
					case "string":
						spec.Items = "String"
					default:
						spec.Items = "Any"
					}
				}
			}
			if spec.Type == "Struct" {
				panic("Whoops, that can't be right")
			}
			if param.Schema.Enum != nil {
				for _, val := range param.Schema.Enum {
					if s, ok := val.(string); ok {
						spec.Values = append(spec.Values, s)
					} else {
						return nil, fmt.Errorf("String enum values are not strings: %v", param.Schema.Enum)
					}
				}
			}
		} else {
		}
		hact.Inputs = append(hact.Inputs, spec)
	}
	if len(queries) > 0 {
		hact.Path = hact.Path + "?" + strings.Join(queries, "&")
	}
	if hact.Method == "POST" || hact.Method == "PUT" || hact.Method == "PATCH" {
		if op.RequestBody != nil {
			for contentType, mediadef := range op.RequestBody.Content {
				if contentType == "application/json" { //hack
					bodyType := oasTypeRef(mediadef.Schema)
					if bodyType != "" {
						spec := &sadl.HttpParamSpec{
							StructFieldDef: sadl.StructFieldDef{
								TypeSpec: sadl.TypeSpec{
									Type: bodyType,
								},
								Comment:  op.RequestBody.Description,
								Name:     "body",
								Required: op.RequestBody.Required,
							},
						}
						hact.Inputs = append(hact.Inputs, spec)
					}
				}
			}
		}
	}
	//expected: if 200 is in the list, use that
	//else: if 201 is in the list, use that
	//else: ? find a likely candidate.
	var expectedStatus string = "default"
	for status, _ := range op.Responses {
		if strings.HasPrefix(status, "2") || strings.HasPrefix(status, "3") {
			expectedStatus = status
			break
		}
	}
	//	if expectedStatus == "default" {
	//		expectedStatus = "200" //?
	//	}
	if expectedStatus != "" {
		eparam := op.Responses[expectedStatus]
		var err error
		code := 200
		if expectedStatus != "default" && strings.Index(expectedStatus, "X") < 0 {
			code, err = strconv.Atoi(expectedStatus)
			if err != nil {
				return nil, err
			}
		}
		ex := &sadl.HttpExpectedSpec{
			Status:  int32(code),
			Comment: eparam.Description,
		}
		for header, def := range eparam.Headers {
			param := &sadl.HttpParamSpec{}
			param.Header = header
			param.Comment = def.Description
			s := param.Header
			//most app-defined headers start with "x-" or "X-". Strip that off for a more reasonable variable name.
			if strings.HasPrefix(param.Header, "x-") || strings.HasPrefix(param.Header, "X-") {
				s = s[2:]
			}
			param.Name = makeIdentifier(s)
			schref := def.Schema
			if schref != nil {
				if schref.Ref != "" {
					param.Type = oasTypeRef(schref)
				} else {
					param.TypeSpec, err = convertOasType(hact.Name+".Expected."+param.Name, schref)
				}
				ex.Outputs = append(ex.Outputs, param)
			}
		}
		for contentType, mediadef := range eparam.Content {
			if contentType == "application/json" { //hack
				result := &sadl.HttpParamSpec{}
				result.Name = "body"
				schref := mediadef.Schema
				if schref != nil {
					if schref.Ref != "" {
						result.Type = oasTypeRef(schref)
					} else {
						result.TypeSpec, err = convertOasType(hact.Name+".Expected.payload", schref)
					}
					ex.Outputs = append(ex.Outputs, result)
				} else {
					fmt.Println("HTTP Action has no expected result type:", sadl.Pretty(eparam))
				}
			}
		}
		hact.Expected = ex
	}
	for status, param := range op.Responses {
		if status != expectedStatus {
			//the status can be "default", or "4XX" (where 'X' is a wildcard) or "404". If the latter, it takes precedence.
			//for SADL, not specifying the response is a bug. So "default" will be turned into "500". The wildcards
			if status == "default" {
				status = "0"
			} else if strings.Index(status, "X") >= 0 {
				panic("wildcard response codes not supported")
			}
			code, err := strconv.Atoi(status)
			if err != nil {
				return nil, fmt.Errorf("Invalid status code: %q", status)
			}
			ex := &sadl.HttpExceptionSpec{
				Status:  int32(code),
				Comment: param.Description,
			}
			//FIXME: sadl should allow response headers for exceptions, also.
			for contentType, mediadef := range param.Content {
				if contentType == "application/json" { //hack
					schref := mediadef.Schema
					if schref != nil {
						if schref.Ref != "" {
							ex.Type = oasTypeRef(schref)
						} else {
							panic("inline response types not yet supported")
						}
						break
					}
				}
			}
			hact.Exceptions = append(hact.Exceptions, ex)
		}
	}
	//tags: add `x-tags="one,two"` annotation
	return hact, nil
}

func getPathOperation(oasPathItem *oas3.PathItem, method string) *oas3.Operation {
	switch method {
	case "GET":
		return oasPathItem.Get
	case "PUT":
		return oasPathItem.Put
	case "POST":
		return oasPathItem.Post
	case "DELETE":
		return oasPathItem.Delete
	case "HEAD":
		return oasPathItem.Head
		/*
			case "PATCH":
				return oasPathItem.Patch
			case "OPTIONS":
				return oasPathItem.Options
			case "TRACE":
				return oasPathItem.Trace
			case "CONNECT":
				return oasPathItem.Connect
		*/
	}
	return nil
}

func guessOperationName(op *oas3.Operation, method string) string {
	defaultStatus := guessDefaultResponseCode(op)
	switch method {
	case "GET":
		resp := op.Responses[defaultStatus]
		if resp == nil {
			resp = op.Responses["default"]
		}
		for contentType, mediadef := range resp.Content {
			if contentType == "application/json" {
				schref := mediadef.Schema
				if schref != nil {
					if schref.Ref != "" {
						entityType := oasTypeRef(schref)
						return entityType
					} else {
						entityType := sadlPrimitiveType(schref.Type)
						if entityType == "Array" {
							itemType := schref.Items
							if itemType.Ref != "" {
								itemTypeName := oasTypeRef(itemType)
								entityType = "ArrayOf" + itemTypeName
							}
						}
						return entityType
					}
				} else {
					fmt.Println("HTTP Action has no expected result type:", sadl.Pretty(resp))
				}
			}
		}
	}
	return ""
}

func sadlPrimitiveType(name string) string {
	switch name {
	case "string":
		return "String"
	case "number":
		return "Decimal"
	case "integer":
		return "Int32"
	case "array":
		return "Array"
	case "object":
		return "Struct"
	case "boolean":
		return "Bool"
	default:
		fmt.Println("sadlPrimitiveType for:", name)
		panic("what?")
	}
}

func findTypeDef(schema *sadl.Schema, name string) *sadl.TypeDef {
	for _, td := range schema.Types {
		if td.Name == name {
			return td
		}
	}
	return nil
}

func guessDefaultResponseCode(op *oas3.Operation) string {
	for status, _ := range op.Responses {
		if strings.HasPrefix(status, "2") || strings.HasPrefix(status, "3") {
			//kind of an arbitrary choice: the first one we encounter, and this is random order, too.
			return status
		}
	}
	return "200" //!
}

func responseTypeName(resp *oas3.Response) string {
	for contentType, mediadef := range resp.Content {
		if contentType == "application/json" { //hack
			schref := mediadef.Schema
			if schref != nil {
				if schref.Ref != "" {
					return oasTypeRef(schref)
				} else {
					ts, err := convertOasType("", schref)
					if err == nil {
						return ts.Type //fixme
					}
				}
			}
		}
	}
	return ""
}
