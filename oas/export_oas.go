package oas

import (
	//	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/oas/oas2"
	"github.com/boynton/sadl/oas/oas3"
	//	"github.com/ghodss/yaml"
)

type Generator struct {
	sadl.Generator
}

func NewGenerator(model *sadl.Model, outdir string) *Generator {
	gen := &Generator{
		Generator: sadl.Generator{
			Model:  model,
			OutDir: outdir,
		},
	}
	pdir := filepath.Join(outdir)
	err := os.MkdirAll(pdir, 0755)
	if err != nil {
		gen.Err = err
	}
	return gen
}

func (gen *Generator) ExportToOAS3() (*oas3.OpenAPI, error) {
	model := gen.Model
	oas := &oas3.OpenAPI{
		OpenAPI: "3.0.0",
		Info:    &oas3.Info{},
	}
	comment := model.Comment
	if comment == "" {
		comment = model.Name
	}
	oas.Info.Title = comment
	oas.Info.Version = model.Version
	if oas.Info.Version == "" {
		oas.Info.Version = "dev"
	}
	if model.Annotations != nil {
		if url, ok := model.Annotations["x_server"]; ok {
			oas.Servers = append(oas.Servers, &oas3.Server{URL: url})
		}
		var license oas3.License
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
	oas.Components = &oas3.Components{}
	oas.Components.Schemas = make(map[string]*oas3.Schema, 0)
	for _, td := range model.Types {
		otd, err := gen.exportTypeDef(td)
		if err != nil {
			return nil, err
		}
		oas.Components.Schemas[td.Name] = otd
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
			//todo: walk the compound name, install at the element if supported.
			//todo: handle operation request and response parameters
			fmt.Fprintf(os.Stderr, "[warning: example not exported for %q\n", ed.Target)
		}
	}
	//Paths
	oas.Paths = make(map[string]*oas3.PathItem, 0)
	for _, hdef := range model.Http {
		var pi *oas3.PathItem
		p := model.Base + hdef.Path
		i := strings.Index(p, "?")
		if i >= 0 {
			p = p[:i]
		}
		if prev, ok := oas.Paths[p]; ok {
			pi = prev
		} else {
			pi = &oas3.PathItem{
				//Extensions
				//Summary
				//Description
				//Servers
				//Parameters
			}
			oas.Paths[p] = pi
		}
		//note: the first tag is always the resource name for the action
		op := &oas3.Operation{
			OperationID: hdef.Name,
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
			param := &oas3.Parameter{
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
				body := &oas3.RequestBody{
					Description: in.Comment,
					Required:    true,
					Content:     make(map[string]*oas3.MediaType, 0),
				}
				tr, err := gen.oasSchema(&in.TypeSpec, "")
				if err != nil {
					return nil, err
				}
				body.Content["application/json"] = &oas3.MediaType{
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
		responses := make(map[string]*oas3.Response, 0)
		op.Responses = responses
		content := make(map[string]*oas3.MediaType)
		comment := hdef.Expected.Comment
		if comment == "" {
			comment = "Expected response"
		}
		resp := &oas3.Response{
			Description: comment,
			Content:     content,
		}
		var headers map[string]*oas3.Header
		for _, param := range hdef.Expected.Outputs {
			if param.Header != "" {
				pschema, err := gen.oasSchema(&param.TypeSpec, "")
				if err != nil {
					return nil, err
				}
				header := &oas3.Header{
					Description: param.Comment,
					Schema:      pschema,
				}
				if headers == nil {
					headers = make(map[string]*oas3.Header, 0)
				}
				headers[param.Header] = header
			} else { //body
				tr, err := gen.oasSchema(&param.TypeSpec, "")
				if err != nil {
					return nil, err
				}
				mt := &oas3.MediaType{
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
			content := make(map[string]*oas3.MediaType)
			comment := out.Comment
			if comment == "" {
				comment = "Exceptional response"
			}
			resp := &oas3.Response{
				Description: comment,
				Content:     content,
			}
			tr := &oas3.Schema{
				Ref: "#/components/schemas/" + out.Type,
			}
			mt := &oas3.MediaType{
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
	return oas, nil
}

func (gen *Generator) exportTypeDef(td *sadl.TypeDef) (*oas3.Schema, error) {
	switch td.Type {
	case "Struct":
		return gen.exportStructTypeDef(td)
	case "Array":
		return gen.exportArrayTypeDef(td)
	case "Map":
		return gen.exportMapTypeDef(td)
	case "String":
		return gen.exportStringTypeDef(td)
	case "Enum":
		return gen.exportEnumTypeDef(td)
	case "Int8", "Int16", "Int32", "Int64", "Float32", "Float64", "Decimal":
		stype, sformat, scomment := oasNumericEquivalent(td.Type)
		otd := &oas3.Schema{
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
	}
	//etc
	return nil, fmt.Errorf("Implement export of this type: %q", td.Type)
}

func (gen *Generator) exportStructTypeDef(td *sadl.TypeDef) (*oas3.Schema, error) {
	schema := &oas3.Schema{
		Type:        "object",
		Description: td.Comment,
	}
	var required []string
	properties := make(map[string]*oas3.Schema, 0)
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

func (gen *Generator) exportArrayTypeDef(td *sadl.TypeDef) (*oas3.Schema, error) {
	itd := gen.Model.FindType(td.Items)
	itemSchema, err := gen.oasSchema(&itd.TypeSpec, td.Items)
	if err != nil {
		return nil, err
	}
	schema := &oas3.Schema{
		Type:        "array",
		Description: td.Comment,
		Items:       itemSchema,
	}
	return schema, nil
}

func (gen *Generator) exportMapTypeDef(td *sadl.TypeDef) (*oas3.Schema, error) {
	return &oas3.Schema{
		Type:        "object",
		Description: td.Comment,
		AdditionalProperties: &oas3.Schema{
			Type: td.Items,
		},
	}, nil
}

func (gen *Generator) exportStringTypeDef(td *sadl.TypeDef) (*oas3.Schema, error) {
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

func (gen *Generator) exportEnumTypeDef(td *sadl.TypeDef) (*oas3.Schema, error) {
	otd := &oas3.Schema{
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

func (gen *Generator) oasSchema(td *sadl.TypeSpec, name string) (*oas3.Schema, error) {
	//	if name != "" {
	//		return &oas3.Schema{
	//			Ref: "#/components/schemas/" + name,
	//		}, nil
	//	}
	switch td.Type {
	case "Int32", "Int16", "Int8", "Int64", "Float32", "Float64", "Decimal":
		if td.Type == "Float64" && name != "" {
			panic("here? " + name + " -> " + sadl.Pretty(td))
		}
		stype, sformat, scomment := oasNumericEquivalent(td.Type)
		sch := &oas3.Schema{
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
		tr := &oas3.Schema{
			Type:   "string",
			Format: "byte",
		}
		//restrictions
		return tr, nil
	case "String":
		tr := &oas3.Schema{
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
		tr := &oas3.Schema{
			Type:   "string",
			Format: "date-time",
		}
		//restrictions
		return tr, nil
	case "UnitValue":
		tr := &oas3.Schema{
			Type: "string",
			//(not standard) Format: "unitvalue",
			Description: "UnitValue",
		}
		return tr, nil
	case "UUID":
		tr := &oas3.Schema{
			Type: "string",
			//(not standard) Format: "uuid",
			Description: "UUID",
		}
		return tr, nil
	case "Enum":
		return &oas3.Schema{
			Ref: "#/components/schemas/" + name,
		}, nil
	case "Array":
		itd := gen.Model.FindType(td.Items)
		itemSchema, err := gen.oasSchema(&itd.TypeSpec, td.Items)
		if err != nil {
			return nil, err
		}
		tr := &oas3.Schema{
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
		otd := &oas3.Schema{
			Type:                 "object",
			AdditionalProperties: itemSchema,
		}
		return otd, nil
	case "Struct":
		if name != "" {
			return &oas3.Schema{
				Ref: "#/components/schemas/" + name,
			}, nil
		} else {
			sch := &oas3.Schema{
				Type: "object",
			}
			f := make(map[string]*oas3.Schema, 0)
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
		return &oas3.Schema{
			Ref: "#/components/schemas/" + td.Type,
		}, nil
	}
}

func (gen *Generator) ExportToOAS2() (*oas2.OpenAPI, error) {
	v3, err := gen.ExportToOAS3()
	if err != nil {
		return nil, err
	}
	return oas2.ConvertFromV3(v3)
}
