package openapi

import (
	"fmt"
	"os"
	"strings"

	"github.com/boynton/sadl"
)

func Export(model *sadl.Model, conf *sadl.Data) error {
	gen := NewGenerator(model, conf)
	doc, err := gen.ExportToOAS3()
	if err != nil {
		return err
	}
	fmt.Println(sadl.Pretty(doc))
	return nil
}

type Generator struct {
	sadl.Generator
	Model *sadl.Model
}

func NewGenerator(model *sadl.Model, conf *sadl.Data) *Generator {
	gen := &Generator{}
	gen.Config = conf
	gen.Model = model
	return gen
}

func (gen *Generator) ExportToOAS3() (*Model, error) {
	model := gen.Model
	oas := &Model{
		OpenAPI: "3.0.0",
		Info:    &Info{},
	}
	comment := model.Comment
	oas.Info.Description = comment
	oas.Info.Title = model.Name
	oas.Info.Version = model.Version
	if oas.Info.Version == "" {
		oas.Info.Version = "dev"
	}
	if model.Annotations != nil {
		if url, ok := model.Annotations["x_server"]; ok {
			oas.Servers = append(oas.Servers, &Server{URL: url})
		}
		var license License
		if lname, ok := model.Annotations["x_license_name"]; ok {
			license.Name = lname
		}
		if lurl, ok := model.Annotations["x_license_url"]; ok {
			license.URL = lurl
		}
		if license.URL != "" || license.Name != "" {
			oas.Info.License = &license
		}
	}
	oas.Components = &Components{}
	oas.Components.Schemas = make(map[string]*Schema, 0)
	for _, td := range model.Types {
		otd, err := gen.exportTypeDef(td)
		if err != nil {
			return nil, err
		}
		oas.Components.Schemas[td.Name] = otd
	}
	//Paths
	oas.Paths = make(map[string]*PathItem, 0)
	for _, hdef := range model.Http {
		var pi *PathItem
		p := model.Base + hdef.Path
		i := strings.Index(p, "?")
		if i >= 0 {
			p = p[:i]
		}
		if prev, ok := oas.Paths[p]; ok {
			pi = prev
		} else {
			pi = &PathItem{
				//Extensions
				//Summary
				//Description
				//Servers
				//Parameters
			}
			oas.Paths[p] = pi
		}
		//note: the first tag is always the resource name for the action
		op := &Operation{
			OperationId: hdef.Name,
			Summary:     hdef.Comment,
			Tags:        []string{hdef.Resource},
			//Parameters
			//Body
			//Responses
			//Callbacks
			//Security
		}
		if len(hdef.Annotations) > 0 {
			for k, v := range hdef.Annotations {
				if k == "x_tags" {
					for _, t := range strings.Split(v, ",") {
						op.Tags = append(op.Tags, t)
					}
				}
			}
		}
		switch hdef.Method {
		case "GET":
			if pi.Get != nil {
				return nil, fmt.Errorf("Duplicate HTTP Method spec (%s %s)", hdef.Method, p)
			}
			pi.Get = op
		case "PUT":
			if pi.Put != nil {
				return nil, fmt.Errorf("Duplicate HTTP Method spec (%s %s)", hdef.Method, p)
			}
			pi.Put = op
		case "DELETE":
			if pi.Delete != nil {
				return nil, fmt.Errorf("Duplicate HTTP Method spec (%s %s)", hdef.Method, p)
			}
			pi.Delete = op
		case "POST":
			if pi.Post != nil {
				return nil, fmt.Errorf("Duplicate HTTP Method spec (%s %s)", hdef.Method, p)
			}
			pi.Post = op
		}
		for _, in := range hdef.Inputs {
			param := &Parameter{
				Name:        in.Name,
				Description: in.Comment,
			}
			r := in.Required
			if in.Path {
				param.In = "path"
				param.Name = in.Name
				r = true
			} else if in.Query != "" {
				param.In = "query"
				param.Name = in.Query
			} else if in.Header != "" {
				param.In = "header"
				param.Name = in.Header
			} else { //body
				body := &RequestBody{
					Description: in.Comment,
					Required:    true,
					Content:     make(map[string]*MediaType, 0),
				}
				tr, err := gen.oasSchema(&in.TypeSpec, "")
				if err != nil {
					return nil, err
				}
				body.Content["application/json"] = &MediaType{
					Schema: tr,
				}
				op.RequestBody = body
				continue
			}
			param.Required = r
			tr, err := gen.oasSchema(&in.TypeSpec, "")
			if err != nil {
				return nil, err
			}
			param.Schema = tr
			op.Parameters = append(op.Parameters, param)
		}
		responses := make(map[string]*Response, 0)
		op.Responses = responses
		content := make(map[string]*MediaType)
		comment := hdef.Expected.Comment
		if comment == "" {
			comment = "Expected response"
		}
		resp := &Response{
			Description: comment,
			Content:     content,
		}
		var headers map[string]*Header
		for _, param := range hdef.Expected.Outputs {
			if param.Header != "" {
				pschema, err := gen.oasSchema(&param.TypeSpec, "")
				if err != nil {
					return nil, err
				}
				header := &Header{
					Description: param.Comment,
					Schema:      pschema,
				}
				if headers == nil {
					headers = make(map[string]*Header, 0)
				}
				headers[param.Header] = header
			} else { //body
				tr, err := gen.oasSchema(&param.TypeSpec, "")
				if err != nil {
					return nil, err
				}
				mt := &MediaType{
					Schema: tr,
				}
				content["application/json"] = mt
			}
		}
		if len(headers) > 0 {
			resp.Headers = headers
		}
		key := fmt.Sprint(hdef.Expected.Status)
		responses[key] = resp
		for _, out := range hdef.Exceptions {
			content := make(map[string]*MediaType)
			comment := out.Comment
			if comment == "" {
				comment = "Exceptional response"
			}
			resp := &Response{
				Description: comment,
				Content:     content,
			}
			tr := &Schema{
				Ref: "#/components/schemas/" + out.Type,
			}
			mt := &MediaType{
				Schema: tr,
			}
			content["application/json"] = mt
			key := "default"
			if out.Status != 0 {
				key = fmt.Sprint(out.Status)
			}
			responses[key] = resp
		}
	}
	//Examples
	for _, ed := range model.Examples {
		//ok, for now: just "example", not "examples", i.e. no name. And, parameter examples override the matching type example
		//media types (the operation body) can also have examples, and also overrides the schema example
		//SO: do not use  oas.Components.Examples just yet
		//the simple case of an example for a type
		if sch, ok := oas.Components.Schemas[ed.Target]; ok {
			sch.Example = ed.Example
		} else {
			//todo: handle operation request and response parameters
			// -> oas allows "examples" on a MediaType (for bodies) and on Parameter (for path/query/header params)
			// ! so, my "request" and "response" example encapsulates all of those, along with error responses.
			// I need to unpack my object and assign the oas items that way. Interestingly, my packaged example may be hard to use?
			// I like the idea of have a complete request object (headers, path, query, and payload) all encapsulated nicely.
			//   -> seems like I could use it in the API somehow. It really is what you need to abstract from the transport. Like RPC.
			//todo: walk the compound name, install at the element if supported.
			if strings.HasSuffix(ed.Target, "Request") {
				hdefName := sadl.Uncapitalize(ed.Target[:len(ed.Target)-7])
				op := gen.FindOperation(oas, hdefName)
				if op != nil {
					for k, v := range ed.Example.(map[string]interface{}) {
						if k == "body" {
							tmp := op.RequestBody.Content["application/json"]
							if ed.Name == "" {
								tmp.Example = v
							} else {
								if tmp.Examples == nil {
									tmp.Examples = make(map[string]*Example, 0)
								}
								tmp.Examples[ed.Name] = &Example{
									Value: v,
								}
							}
						} else {
							for _, param := range op.Parameters {
								if param.Name == k {
									if ed.Name == "" {
										param.Example = v
									} else {
										if param.Examples == nil {
											param.Examples = make(map[string]*Example, 0)
										}
										param.Examples[ed.Name] = &Example{
											Value: v,
										}
									}
								}
							}
						}
					}
				}
			} else if strings.HasSuffix(ed.Target, "Response") {
				hdefName := sadl.Uncapitalize(ed.Target[:len(ed.Target)-8])
				op := gen.FindOperation(oas, hdefName)
				if op != nil {
					for k, v := range ed.Example.(map[string]interface{}) {
						if k == "body" {
							hact := model.FindHttp(hdefName)
							//FIXME: somehow the example name must include the status to use here
							//FIXME: currently the parser only handles Request and Response, not Exceptxxx
							sstatus := fmt.Sprintf("%v", hact.Expected.Status)
							tmp := op.Responses[sstatus].Content["application/json"]
							if ed.Name == "" {
								tmp.Example = v
							} else {
								if tmp.Examples == nil {
									tmp.Examples = make(map[string]*Example, 0)
								}
								tmp.Examples[ed.Name] = &Example{
									Value: v,
								}
							}

						} else {
							for _, param := range op.Parameters {
								if param.Name == k {
									if ed.Name == "" {
										param.Example = v
									} else {
										if param.Examples == nil {
											param.Examples = make(map[string]*Example, 0)
										}
										param.Examples[ed.Name] = &Example{
											Value: v,
										}
									}
								}
							}
						}
					}
				} else {
					panic("no")
				}
			} else {
				fmt.Fprintf(os.Stderr, "[warning: example not exported for %q\n", ed.Target)
			}
		}
	}
	return oas, nil
}

func (gen *Generator) FindOperation(model *Model, opId string) *Operation {
	for _, pathItem := range model.Paths {
		var op *Operation
		if pathItem.Get != nil {
			op = pathItem.Get
		} else if pathItem.Put != nil {
			op = pathItem.Put
		} else if pathItem.Delete != nil {
			op = pathItem.Delete
		} else if pathItem.Post != nil {
			op = pathItem.Post
		} else if pathItem.Head != nil {
			op = pathItem.Head
		} else if pathItem.Patch != nil {
			op = pathItem.Patch
		} else {
			panic("fix me")
		}
		if op.OperationId == opId {
			return op
		}
	}
	return nil
}

func (gen *Generator) exportTypeDef(td *sadl.TypeDef) (*Schema, error) {
	switch td.Type {
	case "Struct":
		return gen.exportStructTypeDef(td)
	case "Array":
		return gen.exportArrayTypeDef(td)
	case "Map":
		return gen.exportMapTypeDef(td)
	case "String":
		return gen.exportStringTypeDef(td)
	case "Bool":
		otd := &Schema{
			Description: td.Comment,
			Type:        "boolean",
		}
		return otd, nil
	case "Enum":
		return gen.exportEnumTypeDef(td)
	case "Int8", "Int16", "Int32", "Int64", "Float32", "Float64", "Decimal":
		stype, sformat, scomment := oasNumericEquivalent(td.Type)
		otd := &Schema{
			Description: scomment,
			Type:        stype,
			Format:      sformat,
		}

		if td.Min != nil {
			v := td.Min.AsFloat64()
			otd.Min = &v
		}
		if td.Max != nil {
			v := td.Max.AsFloat64()
			otd.Max = &v
		}
		return otd, nil
	case "UUID":
		tmp, err := gen.exportStringTypeDef(td)
		if err != nil {
			return nil, err
		}
		tmp.Format = "uuid"
		return tmp, nil
	case "Union":
		return gen.exportUnionTypeDef(td)
	}
	//etc
	return nil, fmt.Errorf("Implement export of this type: %q", td.Type)
}

func (gen *Generator) exportStructTypeDef(td *sadl.TypeDef) (*Schema, error) {
	schema := &Schema{
		Type:        "object",
		Description: td.Comment,
	}
	var required []string
	properties := make(map[string]*Schema, 0)
	for _, fd := range td.Fields {
		if fd.Required {
			required = append(required, fd.Name)
		}
		tr, err := gen.oasSchema(&fd.TypeSpec, "")
		if err != nil {
			return nil, err
		}
		properties[fd.Name] = tr

	}
	schema.Required = required
	schema.Properties = properties
	return schema, nil
}

func (gen *Generator) exportUnionTypeDef(td *sadl.TypeDef) (*Schema, error) {
	schema := &Schema{
		Description: td.Comment,
	}
	for _, vd := range td.Variants {
		v := &Schema{
			Ref: "#/components/schemas/" + vd.Type,
		}
		schema.OneOf = append(schema.OneOf, v)
	}
	return schema, nil
}

func (gen *Generator) exportArrayTypeDef(td *sadl.TypeDef) (*Schema, error) {
	itd := gen.Model.FindType(td.Items)
	itemSchema, err := gen.oasSchema(&itd.TypeSpec, td.Items)
	if err != nil {
		return nil, err
	}
	schema := &Schema{
		Type:        "array",
		Description: td.Comment,
		Items:       itemSchema,
	}
	return schema, nil
}

func (gen *Generator) exportMapTypeDef(td *sadl.TypeDef) (*Schema, error) {
	itd := gen.Model.FindType(td.Items)
	itemSchema, err := gen.oasSchema(&itd.TypeSpec, "")
	if err != nil {
		return nil, err
	}
	return &Schema{
		Type:                 "object",
		Description:          td.Comment,
		AdditionalProperties: itemSchema,
	}, nil
}

func (gen *Generator) exportStringTypeDef(td *sadl.TypeDef) (*Schema, error) {
	otd, err := gen.oasSchema(&td.TypeSpec, td.Name)
	if err != nil {
		return nil, err
	}
	otd.Description = td.Comment
	if len(td.Values) > 0 {
		e := make([]interface{}, 0)
		for _, s := range td.Values {
			e = append(e, s)
		}
		otd.Enum = e
	}
	return otd, nil
}

func (gen *Generator) exportEnumTypeDef(td *sadl.TypeDef) (*Schema, error) {
	otd := &Schema{
		Type:        "string",
		Description: td.Comment,
	}
	e := make([]interface{}, 0)
	for _, el := range td.Elements {
		e = append(e, el.Symbol)
	}
	otd.Enum = e
	return otd, nil
}

func oasNumericEquivalent(sadlTypeName string) (string, string, string) {
	//See https://github.com/json-schema-org/json-schema-spec/issues/563 for problems with incomplete set of format codes
	//Also https://github.com/OAI/OpenAPI-Specification/issues/845. This is long-running problems with oas.
	//I stick to the standard codes, but add a comment so "autoconversion" is not totally silent.
	switch sadlTypeName {
	case "Int8", "Int16":
		//return "integer", strings.ToLower(sadlTypeName), "" //preferred, but technically violates current swagger spec
		return "integer", "int32", sadlTypeName
	case "Int32":
		return "integer", "int32", ""
	case "Int64":
		return "integer", "int64", ""
	case "Float32":
		return "number", "float", ""
	case "Float64":
		return "number", "double", ""
	case "Decimal":
		//return "number", "decimal", "" //technically most correct, but various implementations cannot handle JSON bignums
		//return "string", "decimal", "" //technically most correct, but various implementations cannot handle JSON bignums
		return "number", "", sadlTypeName
	default:
		return "", "", ""
	}
}

func (gen *Generator) oasSchema(td *sadl.TypeSpec, name string) (*Schema, error) {
	//	if name != "" {
	//		return &Schema{
	//			Ref: "#/components/schemas/" + name,
	//		}, nil
	//	}
	switch td.Type {
	case "Bool":
		sch := &Schema{
			Type: "boolean",
		}
		return sch, nil
	case "Int32", "Int16", "Int8", "Int64", "Float32", "Float64", "Decimal":
		if td.Type == "Float64" && name != "" {
			panic("here? " + name + " -> " + sadl.Pretty(td))
		}
		stype, sformat, scomment := oasNumericEquivalent(td.Type)
		sch := &Schema{
			Type:        stype,
			Format:      sformat,
			Description: scomment,
		}
		if td.Min != nil {
			v := td.Min.AsFloat64()
			sch.Min = &v
		}
		if td.Max != nil {
			v := td.Max.AsFloat64()
			sch.Max = &v
		}
		return sch, nil
	case "Bytes":
		tr := &Schema{
			Type:   "string",
			Format: "byte",
		}
		//restrictions
		return tr, nil
	case "String":
		tr := &Schema{
			Type: "string",
		}
		if td.Pattern != "" {
			tr.Pattern = td.Pattern
		}
		if td.MinSize != nil {
			tr.MinLength = uint64(*td.MinSize)
		}
		if td.MaxSize != nil {
			tmp := uint64(*td.MaxSize)
			tr.MaxLength = &tmp
		}
		return tr, nil
	case "Timestamp":
		tr := &Schema{
			Type:   "string",
			Format: "date-time",
		}
		//restrictions
		return tr, nil
	case "UnitValue":
		tr := &Schema{
			Type: "string",
			//(not standard) Format: "unitvalue",
			Description: "UnitValue",
		}
		return tr, nil
	case "UUID":
		tr := &Schema{
			Type: "string",
			//(not standard) Format: "uuid",
			Description: "UUID",
		}
		return tr, nil
	case "Enum":
		return &Schema{
			Ref: "#/components/schemas/" + name,
		}, nil
	case "Array":
		itd := gen.Model.FindType(td.Items)
		itemSchema, err := gen.oasSchema(&itd.TypeSpec, td.Items)
		if err != nil {
			return nil, err
		}
		tr := &Schema{
			Type:  "array",
			Items: itemSchema,
		}
		return tr, nil
	case "Map":
		//note: keys are always strings
		itd := gen.Model.FindType(td.Items)
		itemSchema, err := gen.oasSchema(&itd.TypeSpec, "")
		if err != nil {
			return nil, err
		}
		otd := &Schema{
			Type:                 "object",
			AdditionalProperties: itemSchema,
		}
		return otd, nil
	case "Struct":
		if name != "" {
			return &Schema{
				Ref: "#/components/schemas/" + name,
			}, nil
		} else {
			sch := &Schema{
				Type: "object",
			}
			f := make(map[string]*Schema, 0)
			for _, fd := range td.Fields {
				fieldSchema, err := gen.oasSchema(&fd.TypeSpec, "")
				if err != nil {
					return nil, err
				}
				f[fd.Name] = fieldSchema
			}
			sch.Properties = f
			return sch, nil
		}
	default:
		return &Schema{
			Ref: "#/components/schemas/" + td.Type,
		}, nil
	}
}

/*
func (gen *Generator) ExportToOAS2() (*oas2.OpenAPI, error) {
	v3, err := gen.ExportToOAS3()
	if err != nil {
		return nil, err
	}
	return oas2.ConvertFromV3(v3)
}
*/
