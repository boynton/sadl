package openapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/boynton/sadl"
	"github.com/ghodss/yaml"
)

func IsValidSwaggerFile(path string) bool {
	_, err := LoadSwagger(path)
	return err == nil
}

func LoadSwagger(path string) (*Model, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Cannot read OpenAPI file: %v\n", err)
	}
	var v2 map[string]interface{}
	ext := filepath.Ext(path)
	if ext == ".yaml" {
		err = yaml.Unmarshal(data, &v2)
	} else {
		err = json.Unmarshal(data, &v2)
	}
	if err != nil {
		return nil, err
	}
	return toV3(sadl.AsData(v2))
}

func toV3(v2 *sadl.Data) (*Model, error) {
	var defConsumes []string
	var defProduces []string
	model := &Model{OpenAPI: "3.0.0"}
	if v2.Has("consumes") {
		defConsumes = v2.GetStringArray("consumes")
	}
	if v2.Has("produces") {
		defProduces = v2.GetStringArray("produces")
	}
	if v2.GetString("swagger") == "2.0" {
		if v2.Has("externalDocs") {
			model.ExternalDocs = toV3ExternalDocs(v2.GetData("externalDocs"))
		}
		model.Info = toV3Info(v2.GetData("info"))
		model.Components = toV3Components(v2)
		model.Paths = toV3Paths(model, v2.GetData("paths"), defConsumes, defProduces)
		model.Servers = toV3Servers(v2)
		return model, nil
	}
	return nil, fmt.Errorf("Not valid Swagger file")
}

func toV3ExternalDocs(v2 *sadl.Data) *ExternalDocumentation {
	return &ExternalDocumentation{
		Description: v2.GetString("description"),
		URL:         v2.GetString("url"),
	}
}

func toV3Paths(model *Model, v2 *sadl.Data, defConsumes []string, defProduces []string) map[string]*PathItem {
	paths := make(map[string]*PathItem, 0)
	for k, v := range v2.AsMap() {
		vd := sadl.AsData(v)
		pi := &PathItem{
			Get:    toV3Operation(model, vd.GetData("get"), defConsumes, defProduces),
			Delete: toV3Operation(model, vd.GetData("delete"), defConsumes, defProduces),
			Put:    toV3Operation(model, vd.GetData("put"), defConsumes, defProduces),
			Post:   toV3Operation(model, vd.GetData("post"), defConsumes, defProduces),
			Patch:  toV3Operation(model, vd.GetData("patch"), defConsumes, defProduces),
			Head:   toV3Operation(model, vd.GetData("head"), defConsumes, defProduces),
			//etc
		}
		paths[k] = pi
	}
	return paths
}

func toV3Operation(model *Model, v2 *sadl.Data, defConsumes []string, defProduces []string) *Operation {
	if v2.IsNil() {
		return nil
	}
	op := &Operation{
		OperationId: v2.GetString("operationId"),
		Tags:        v2.GetStringArray("tags"),
		Summary:     v2.GetString("summary"),
		Description: v2.GetString("description"),
	}
	//responses
	consumes := v2.GetStringArray("consumes")
	if consumes == nil {
		consumes = defConsumes
	}
	var params []*Parameter
	for _, param := range v2.GetArray("parameters") {
		paramd := sadl.AsData(param)
		if paramd.GetString("in") == "body" {
			op.RequestBody = &RequestBody{
				Description: paramd.GetString("description"),
				Required:    paramd.GetBool("required"),
				Content:     toV3Content(consumes, paramd.Get("schema")),
			}
		} else {
			var tp interface{}
			tp = paramd.GetString("type")
			if tp == "" {
				tp = paramd.Get("schema")
			}
			p := &Parameter{
				In:       paramd.GetString("in"),
				Name:     paramd.GetString("name"),
				Required: paramd.GetBool("required"),
				Schema:   toV3Schema(tp),
			}
			params = append(params, p)
		}
	}
	op.Parameters = params
	produces := v2.GetStringArray("produces")
	if produces == nil {
		produces = defProduces
	}
	v2Responses := v2.GetMap("responses")

	if len(v2Responses) > 0 {
		responses := make(map[string]*Response, 0)
		for k, v := range v2Responses {
			responses[k] = toV3Response(produces, sadl.AsData(v))
		}
		op.Responses = responses
	}
	return op
}

func toV3Response(mediaTypes []string, v2 *sadl.Data) *Response {
	return &Response{
		Description: v2.GetString("description"),
		Headers:     toV3Headers(v2.GetMap("headers")),
		Content:     toV3Content(mediaTypes, v2.Get("schema")),
	}
}

func toV3Headers(v2 map[string]interface{}) map[string]*Header {
	if len(v2) == 0 {
		return nil
	}
	headers := make(map[string]*Header, 0)
	for k, v := range v2 {
		vd := sadl.AsData(v)
		headers[k] = &Header{
			Description: vd.GetString("description"),
			Schema:      toV3Schema(vd.GetString("type")),
		}
	}
	return headers
}

func toV3Content(mediaTypes []string, v2 interface{}) Content {
	if v2 == nil {
		return nil
	}
	schema := toV3Schema(v2)
	content := make(map[string]*MediaType, 0)
	for _, mt := range mediaTypes {
		md := &MediaType{
			Schema: schema,
			//Example, Examples
			//Encoding
		}
		content[mt] = md
	}
	return content
}

func toV3Components(v2 *sadl.Data) *Components {
	v2Schemas := v2.GetData("definitions")
	schemas := make(map[string]*Schema, 0)
	for k, v := range v2Schemas.AsMap() {
		schemas[k] = toV3Schema(v)
	}
	return &Components{
		Schemas: schemas,
	}
}

func toV3Schema(v2Any interface{}) *Schema {
	schema := &Schema{}
	if v2String, ok := v2Any.(string); ok {
		schema.Type = v2String
		return schema
	}
	v2 := sadl.AsData(v2Any)
	schema.Description = v2.GetString("description")
	schema.Type = v2.GetString("type")
	schema.Required = v2.GetStringArray("required")
	switch schema.Type {
	case "object":
		schema.Properties = make(map[string]*Schema, 0)
		for k, v := range v2.GetData("properties").AsMap() {
			schema.Properties[k] = toV3Schema(v)
		}
	case "array":
		schema.Items = toV3Schema(v2.Get("items"))
	case "string":
		schema.Format = v2.GetString("format")
		schema.Pattern = v2.GetString("pattern")
		if v2.Has("maxLength") {
			n := uint64(v2.GetInt64("maxLength"))
			schema.MaxLength = &n
		}
		if v2.Has("minLength") {
			n := uint64(v2.GetInt64("minLength"))
			schema.MinLength = n
		}
		//values
		if v2.Has("enum") {
			var en []interface{}
			for _, e := range v2.GetStringArray("enum") {
				en = append(en, e)
			}
			schema.Enum = en
		}
	case "integer", "number":
		schema.Format = v2.GetString("format")
		if v2.Has("min") {
			f := v2.GetFloat64("min")
			schema.Min = &f
		}
		if v2.Has("max") {
			f := v2.GetFloat64("max")
			schema.Max = &f
		}
	case "boolean":
		//
	default:
		if v2.Has("$ref") {
			schema.Ref = strings.Replace(v2.GetString("$ref"), "#/definitions/", "#/components/schemas/", -1)
		} else {
			fmt.Printf("type: %q\n%s\n->%v\n", schema.Type, sadl.Pretty(v2), v2.Get("$ref"))
			panic("fix this schema")
		}
	}
	return schema
}

func toV3Servers(v2 *sadl.Data) []*Server {
	var servers []*Server
	if v2.Has("host") {
		bp := v2.GetString("basePath")
		server := &Server{
			URL: "https://" + v2.GetString("host") + bp,
		}
		servers = append(servers, server)
	}
	return servers
}

func toV3Info(v2 *sadl.Data) *Info {
	return &Info{
		Title:          v2.GetString("title"),
		Description:    v2.GetString("description"),
		TermsOfService: v2.GetString("termsOfService"),
		Contact:        toV3Contact(v2.GetData("contact")),
		License:        toV3License(v2.GetData("license")),
		Version:        v2.GetString("version"),
	}
}

func toV3Contact(v2 *sadl.Data) *Contact {
	return &Contact{
		Name:  v2.GetString("name"),
		URL:   v2.GetString("url"),
		Email: v2.GetString("email"),
	}
}

func toV3License(v2 *sadl.Data) *License {
	return &License{
		Name: v2.GetString("name"),
		URL:  v2.GetString("url"),
	}
}

func ImportSwagger(paths []string, conf *sadl.Data) (*sadl.Model, error) {
	if len(paths) != 1 {
		return nil, fmt.Errorf("Cannot merge multiple Swagger files")
	}
	path := paths[0]
	name := path
	n := strings.LastIndex(name, "/")
	//	format := ""
	if n >= 0 {
		name = name[n+1:]
	}
	n = strings.LastIndex(name, ".")
	if n >= 0 {
		//		format = name[n+1:]
		name = name[:n]
		name = strings.Replace(name, ".", "_", -1)
	}
	name = strings.Replace(name, "-", "_", -1)
	oas3, err := LoadSwagger(path)
	if err != nil {
		return nil, err
	}
	model, err := oas3.ToSadl(name)
	if err != nil {
		return nil, fmt.Errorf("Cannot convert to SADL: %v\n", err)
	}
	//err = model.ConvertInlineEnums()
	return model, err
}
