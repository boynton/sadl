package oas2

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/oas/oas3"
	"github.com/ghodss/yaml"
)

type OpenAPI struct {
	Extensions          map[string]interface{}     `json:"-"`
	ID                  string                     `json:"id,omitempty"`
	Consumes            []string                   `json:"consumes,omitempty"`
	Produces            []string                   `json:"produces,omitempty"`
	Schemes             []string                   `json:"schemes,omitempty"`
	Swagger             string                     `json:"swagger,omitempty"`
	Info                *Info                      `json:"info,omitempty"`
	Host                string                     `json:"host,omitempty"`
	BasePath            string                     `json:"basePath,omitempty"`
	Paths               map[string]*PathItem       `json:"paths,omitempty"`
	Definitions         map[string]*Schema         `json:"definitions,omitempty"`
	Parameters          map[string]*Parameter      `json:"parameters,omitempty"`
	Responses           map[string]*Response       `json:"responses,omitempty"`
	SecurityDefinitions map[string]*SecurityScheme `json:"securityDefinitions,omitempty"`
	Security            []map[string][]string      `json:"security,omitempty"`
	Tags                []Tag                      `json:"tags,omitempty"`
	ExternalDocs        *ExternalDocumentation     `json:"externalDocs,omitempty"`
}

type Info struct {
	Extensions     map[string]interface{} `json:"-"`
	Title          string                 `json:"title,omitempty"`
	Description    string                 `json:"description,omitempty"`
	TermsOfService string                 `json:"termsOfService,omitempty"`
	Contact        *ContactInfo           `json:"contact,omitempty"`
	License        *License               `json:"license,omitempty"`
	Version        string                 `json:"version,omitempty"`
}

type ContactInfo struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

type License struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

type PathItem struct {
	Extensions map[string]interface{} `json:"-"`
	Get        *Operation             `json:"get,omitempty"`
	Put        *Operation             `json:"put,omitempty"`
	Post       *Operation             `json:"post,omitempty"`
	Delete     *Operation             `json:"delete,omitempty"`
	Options    *Operation             `json:"options,omitempty"`
	Head       *Operation             `json:"head,omitempty"`
	Patch      *Operation             `json:"patch,omitempty"`
	Parameters []Parameter            `json:"parameters,omitempty"`
}

type Operation struct {
	Extensions   map[string]interface{} `json:"-"`
	Description  string                 `json:"description,omitempty"`
	Consumes     []string               `json:"consumes,omitempty"`
	Produces     []string               `json:"produces,omitempty"`
	Schemes      []string               `json:"schemes,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	Summary      string                 `json:"summary,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty"`
	ID           string                 `json:"operationId,omitempty"`
	Deprecated   bool                   `json:"deprecated,omitempty"`
	Security     []map[string][]string  `json:"security,omitempty"`
	Parameters   []Parameter            `json:"parameters,omitempty"`
	Responses    *Responses             `json:"responses,omitempty"`
}

type ExternalDocumentation struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
}

type Parameter struct {
	Extensions      map[string]interface{} `json:"-"`
	Description     string                 `json:"description,omitempty"`
	Name            string                 `json:"name,omitempty"`
	In              string                 `json:"in,omitempty"`
	Required        bool                   `json:"required,omitempty"`
	Schema          *Schema                `json:"schema,omitempty"`
	AllowEmptyValue bool                   `json:"allowEmptyValue,omitempty"`
}

type StringOrArray []string

func (s *StringOrArray) UnmarshalJSON(data []byte) error {
	var first byte
	if len(data) > 1 {
		first = data[0]
	}

	if first == '[' {
		var parsed []string
		if err := json.Unmarshal(data, &parsed); err != nil {
			return err
		}
		*s = StringOrArray(parsed)
		return nil
	}

	var single interface{}
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	if single == nil {
		return nil
	}
	switch v := single.(type) {
	case string:
		*s = StringOrArray([]string{v})
		return nil
	default:
		return fmt.Errorf("only string or array is allowed, not %T", single)
	}
}

type SchemaOrArray struct {
	Schema  *Schema
	Schemas []Schema
}

func (s *SchemaOrArray) UnmarshalJSON(data []byte) error {
	var nw SchemaOrArray
	var first byte
	if len(data) > 1 {
		first = data[0]
	}
	if first == '{' {
		var sch Schema
		if err := json.Unmarshal(data, &sch); err != nil {
			return err
		}
		nw.Schema = &sch
	}
	if first == '[' {
		if err := json.Unmarshal(data, &nw.Schemas); err != nil {
			return err
		}
	}
	*s = nw
	return nil
}

type SchemaOrStringArray struct {
	Schema   *Schema
	Property []string
}

func (s *SchemaOrStringArray) UnmarshalJSON(data []byte) error {
	var first byte
	if len(data) > 1 {
		first = data[0]
	}
	var nw SchemaOrStringArray
	if first == '{' {
		var sch Schema
		if err := json.Unmarshal(data, &sch); err != nil {
			return err
		}
		nw.Schema = &sch
	}
	if first == '[' {
		if err := json.Unmarshal(data, &nw.Property); err != nil {
			return err
		}
	}
	*s = nw
	return nil
}

type SchemaOrBool struct {
	Allows bool
	Schema *Schema
}

func (s *SchemaOrBool) UnmarshalJSON(data []byte) error {
	var nw SchemaOrBool
	if len(data) >= 4 {
		if data[0] == '{' {
			var sch Schema
			if err := json.Unmarshal(data, &sch); err != nil {
				return err
			}
			nw.Schema = &sch
		}
		nw.Allows = !(data[0] == 'f' && data[1] == 'a' && data[2] == 'l' && data[3] == 's' && data[4] == 'e')
	}
	*s = nw
	return nil
}

type Dependencies map[string]SchemaOrStringArray

type Schema struct {
	Extensions           map[string]interface{} `json:"-"`
	ID                   string                 `json:"id,omitempty"`
	Ref                  string                 `json:"$ref,omitempty"`
	Schema               string                 `json:"-"` //{"$schema": string(r)} //FIXME
	Description          string                 `json:"description,omitempty"`
	Type                 StringOrArray          `json:"type,omitempty"`
	Format               string                 `json:"format,omitempty"`
	Title                string                 `json:"title,omitempty"`
	Default              interface{}            `json:"default,omitempty"`
	Maximum              *float64               `json:"maximum,omitempty"`
	ExclusiveMaximum     bool                   `json:"exclusiveMaximum,omitempty"`
	Minimum              *float64               `json:"minimum,omitempty"`
	ExclusiveMinimum     bool                   `json:"exclusiveMinimum,omitempty"`
	MaxLength            *int64                 `json:"maxLength,omitempty"`
	MinLength            *int64                 `json:"minLength,omitempty"`
	Pattern              string                 `json:"pattern,omitempty"`
	MaxItems             *int64                 `json:"maxItems,omitempty"`
	MinItems             *int64                 `json:"minItems,omitempty"`
	UniqueItems          bool                   `json:"uniqueItems,omitempty"`
	MultipleOf           *float64               `json:"multipleOf,omitempty"`
	Enum                 []interface{}          `json:"enum,omitempty"`
	MaxProperties        *int64                 `json:"maxProperties,omitempty"`
	MinProperties        *int64                 `json:"minProperties,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Items                *SchemaOrArray         `json:"items,omitempty"`
	AllOf                []*Schema              `json:"allOf,omitempty"`
	OneOf                []*Schema              `json:"oneOf,omitempty"`
	AnyOf                []*Schema              `json:"anyOf,omitempty"`
	Not                  *Schema                `json:"not,omitempty"`
	Properties           map[string]*Schema     `json:"properties,omitempty"`
	AdditionalProperties *SchemaOrBool          `json:"additionalProperties,omitempty"`
	PatternProperties    map[string]*Schema     `json:"patternProperties,omitempty"`
	Dependencies         Dependencies           `json:"dependencies,omitempty"`
	AdditionalItems      *SchemaOrBool          `json:"additionalItems,omitempty"`
	Definitions          map[string]*Schema     `json:"definitions,omitempty"`
	Discriminator        string                 `json:"discriminator,omitempty"` //swagger extension
	ReadOnly             bool                   `json:"readOnly,omitempty"`      //swagger extension
	XML                  *XMLObject             `json:"xml,omitempty"`           //swagger extension
	ExternalDocs         *ExternalDocumentation `json:"externalDocs,omitempty"`  //swagger extension
	Example              interface{}            `json:"example,omitempty"`       //swagger extension
}

type Responses struct {
	Extensions          map[string]interface{} `json:"-"`
	Default             *Response
	StatusCodeResponses map[int]Response
}

type Response struct {
	Extensions  map[string]interface{} `json:"-"`
	Ref         string                 `json:"$ref,omitempty"`
	Description string                 `json:"description,omitempty"`
	Schema      *Schema                `json:"schema,omitempty"`
	Headers     map[string]Header      `json:"headers,omitempty"`
	Examples    map[string]interface{} `json:"examples,omitempty"`
}

type Header struct {
	CommonValidations
	SimpleSchema
	Description string `json:"description,omitempty"`
}

type SimpleSchema struct {
	Type             string      `json:"type,omitempty"`
	Format           string      `json:"format,omitempty"`
	Items            *Items      `json:"items,omitempty"`
	CollectionFormat string      `json:"collectionFormat,omitempty"`
	Default          interface{} `json:"default,omitempty"`
	Example          interface{} `json:"example,omitempty"`
}

type CommonValidations struct {
	Maximum          *float64      `json:"maximum,omitempty"`
	ExclusiveMaximum bool          `json:"exclusiveMaximum,omitempty"`
	Minimum          *float64      `json:"minimum,omitempty"`
	ExclusiveMinimum bool          `json:"exclusiveMinimum,omitempty"`
	MaxLength        *int64        `json:"maxLength,omitempty"`
	MinLength        *int64        `json:"minLength,omitempty"`
	Pattern          string        `json:"pattern,omitempty"`
	MaxItems         *int64        `json:"maxItems,omitempty"`
	MinItems         *int64        `json:"minItems,omitempty"`
	UniqueItems      bool          `json:"uniqueItems,omitempty"`
	MultipleOf       *float64      `json:"multipleOf,omitempty"`
	Enum             []interface{} `json:"enum,omitempty"`
}

type Items struct {
	Extensions map[string]interface{} `json:"-"`
	Ref        string                 `json:"$ref,omitempty"`
	CommonValidations
	SimpleSchema
}

type SecurityScheme interface{} //fixme

type Tag struct {
	Extensions   map[string]interface{} `json:"-"`
	Description  string                 `json:"description,omitempty"`
	Name         string                 `json:"name,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty"`
}

type XMLObject interface{} //who cares

func Parse(data []byte, format string) (*OpenAPI, error) {
	var err error
	v2 := &OpenAPI{}
	if format == "yaml" {
		err = yaml.Unmarshal(data, &v2)
	} else {
		err = json.Unmarshal(data, &v2)
	}
	return v2, err
}

func ConvertToV3(v2 *OpenAPI) (*oas3.OpenAPI, error) {
	fmt.Println(sadl.Pretty(v2))
	v3 := &oas3.OpenAPI{
		Components: &oas3.Components{},
	}
	v3.Components.Schemas = make(map[string]*oas3.Schema, 0)
	for name, val := range v2.Definitions {
		val3, err := convertSchema(name, val)
		if err != nil {
			return nil, err
		}
		v3.Components.Schemas[name] = val3
	}
	v3.Paths = make(map[string]*oas3.PathItem, 0)
	for tmpl, pathItem := range v2.Paths {
		path, err := convertPath(pathItem)
		if err != nil {
			return nil, err
		}
		v3.Paths[tmpl] = path
	}
	//to do: the actions
	return v3, nil
}

func convertPath(v2Path *PathItem) (*oas3.PathItem, error) {
	v3Path := &oas3.PathItem{}
	v3Path.Extensions = v2Path.Extensions
	return v3Path, nil
}

func convertSchema(xname string, v2 *Schema) (*oas3.Schema, error) {
	var err error
	//	fmt.Println("v2 type:", xname, sadl.Pretty(v2))
	v3 := &oas3.Schema{
		Description: v2.Description,
	}
	if v2.Ref != "" {
		v3.Ref = strings.Replace(v2.Ref, "#/definitions/", "#/components/schemas/", -1)
	}
	if v2.Type != nil && len(v2.Type) > 0 {
		if len(v2.Type) > 1 {
			panic("unions not yet handled")
		}
		stype := v2.Type[0]
		switch stype {
		case "string":
			v3.Type = "string"
			//todo restrictions
		case "array":
			v3.Type = "array"
			v3.Items, err = convertSchema("", v2.Items.Schema)
			if err != nil {
				return nil, err
			}
		case "object":
			v3.Type = "object"
			if v2.Properties != nil {
				schemas := make(map[string]*oas3.Schema, 0)
				for fname, fschema := range v2.Properties {
					v3schema, err := convertSchema(fname, fschema)
					if err != nil {
						return nil, err
					}
					schemas[fname] = v3schema
				}
				v3.Properties = schemas
			}
		case "boolean":
			v3.Type = "boolean"
		case "number":
			v3.Type = "number"
			v3.Format = v2.Format
		case "integer":
			v3.Type = "number"
			v3.Format = "int32"
		default:
			fmt.Println("FIX THIS:", sadl.Pretty(v2))
			panic("here")
		}
	}
	//	fmt.Println("--------->", sadl.Pretty(v3))
	return v3, nil
}

/*
//	v3, err := openapi2conv.ToV3Swagger(v2)
	if err != nil {
		return nil, err
	}

   for _, oasSchema := range v3.Components.Schemas {
		fixV2SchemaRef(oasSchema)
	}
	for _, item := range v3.Paths {
		for _, param := range item.Parameters {
			fixV2ParamRef(param)
		}
		fixV2Operation(item.Delete)
		fixV2Operation(item.Get)
		fixV2Operation(item.Head)
		fixV2Operation(item.Options)
		fixV2Operation(item.Patch)
		fixV2Operation(item.Post)
		fixV2Operation(item.Put)
		fixV2Operation(item.Trace)
	}
	return &Oas{v3: v3}, nil
}

func fixV2Operation(op *openapi3.Operation) {
	if op != nil {
		if op.RequestBody != nil {
			body := op.RequestBody
			body.Ref = strings.Replace(body.Ref, "#/definitions/", "#/components/schemas/", -1)
			if body.Value != nil {
				for _, tmp := range body.Value.Content {
					fixV2SchemaRef(tmp.Schema)
				}
			}
		}
		for _, resp := range op.Responses {
			if resp.Ref != "" {
				resp.Ref = strings.Replace(resp.Ref, "#/definitions/", "#/components/schemas/", -1)
			} else {
				for _, tmp := range resp.Value.Content {
					fixV2SchemaRef(tmp.Schema)
				}
			}
		}
	}
}

func fixV2ParamRef(sref *openapi3.ParameterRef) {
	if sref.Ref != "" {
		sref.Ref = strings.Replace(sref.Ref, "#/definitions/", "#/components/schemas/", -1)
	} else if sref.Value != nil {
		fixV2SchemaRef(sref.Value.Schema)
	}
}

func fixV2Schema(sch *openapi3.Schema) {
	if sch.Properties != nil {
		for _, prop := range sch.Properties {
			fixV2SchemaRef(prop)
		}
	} else if sch.Items != nil {
		fixV2SchemaRef(sch.Items)
	}
}

func fixV2SchemaRef(sref *openapi3.SchemaRef) {
	if sref.Ref != "" {
		sref.Ref = strings.Replace(sref.Ref, "#/definitions/", "#/components/schemas/", -1)
	} else if sref.Value != nil {
		fixV2Schema(sref.Value)
	}

}
*/
