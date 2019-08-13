package oas

import (
	//	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/boynton/sadl"
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
	}
	oas.Info.Title = model.Comment //how we import it.
	oas.Info.Version = model.Version
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
	//Paths
	oas.Paths = make(map[string]*oas3.PathItem, 0)
	for _, hdef := range model.Http {
		var pi *oas3.PathItem
		p := hdef.Path
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
		op := &oas3.Operation{
			OperationID: hdef.Name,
			Summary:     hdef.Comment,
			//Parameters
			//Body
			//Responses
			//Callbacks
			//Security
		}
		if len(hdef.Annotations) > 0 {
			for k, v := range hdef.Annotations {
				if k == "x_tags" {
					op.Tags = strings.Split(v, ",")
				}
			}
		}
		switch hdef.Method {
		case "GET":
			if pi.Get != nil {
				return nil, fmt.Errorf("Duplicate HTTP Method spec (%s %s)", hdef.Method, p)
			}
			pi.Get = op
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
				r = true
			} else if in.Query != "" {
				param.In = "query"
			}
			param.Required = r
			tr, err := gen.oasTypeRef(&in.TypeSpec)
			if err != nil {
				return nil, err
			}
			param.Schema = tr
			op.Parameters = append(op.Parameters, param)
		}
		responses := make(map[string]*oas3.Response, 0)
		op.Responses = responses
		content := make(map[string]*oas3.MediaType)
		resp := &oas3.Response{
			Description: hdef.Expected.Comment,
			Content:     content,
		}
		var headers map[string]*oas3.Header
		for _, param := range hdef.Expected.Outputs {
			if param.Name == "body" {
				tr, err := gen.oasTypeRef(&param.TypeSpec)
				if err != nil {
					return nil, err
				}
				mt := &oas3.MediaType{
					Schema: tr,
				}
				content["application/json"] = mt
			} else if param.Header != "" {
				pschema, err := gen.oasTypeRef(&param.TypeSpec)
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
			}
		}
		if len(headers) > 0 {
			resp.Headers = headers
		}
		key := fmt.Sprint(hdef.Expected.Status)
		responses[key] = resp
		for _, out := range hdef.Exceptions {
			content := make(map[string]*oas3.MediaType)
			resp := &oas3.Response{
				Description: out.Comment,
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
	}
	return nil, fmt.Errorf("Implement export of this type: %q", td.Type)
}

func (gen *Generator) exportStructTypeDef(td *sadl.TypeDef) (*oas3.Schema, error) {
	otd := &oas3.Schema{
		Description: td.Comment,
	}
	var required []string
	properties := make(map[string]*oas3.Schema, 0)
	for _, fd := range td.Fields {
		if fd.Required {
			required = append(required, fd.Name)
		}
		tr, err := gen.oasTypeRef(&fd.TypeSpec)
		if err != nil {
			return nil, err
		}
		properties[fd.Name] = tr

	}
	otd.Required = required
	otd.Properties = properties
	return otd, nil
}

func (gen *Generator) exportArrayTypeDef(td *sadl.TypeDef) (*oas3.Schema, error) {
	otd, err := gen.oasTypeRef(&td.TypeSpec)
	if err != nil {
		return nil, err
	}
	otd.Description = td.Comment
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

func (gen *Generator) oasTypeRef(td *sadl.TypeSpec) (*oas3.Schema, error) {
	switch td.Type {
	case "Int32", "Int16", "Int8", "Int64", "Float32", "Float64", "Decimal":
		stype, sformat, scomment := oasNumericEquivalent(td.Type)
		return &oas3.Schema{
			Type:        stype,
			Format:      sformat,
			Description: scomment,
		}, nil
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
		//restrictions
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
	case "Array":
		tr := &oas3.Schema{
			Type: "array",
			Items: &oas3.Schema{
				Ref: "#/components/schemas/" + td.Items,
			},
		}
		return tr, nil
	default:
		return &oas3.Schema{
			Ref: "#/components/schemas/" + td.Type,
		}, nil
	}
}
