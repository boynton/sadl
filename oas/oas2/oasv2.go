package oas2

import (
	"encoding/json"
	"fmt"
	"strings"

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
	OperationId string                 `json:"operationId,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Consumes     []string               `json:"consumes,omitempty"`
	Produces     []string               `json:"produces,omitempty"`
	Schemes      []string               `json:"schemes,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	Summary      string                 `json:"summary,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty"`
	Deprecated   bool                   `json:"deprecated,omitempty"`
	Security     []map[string][]string  `json:"security,omitempty"`
	Parameters   []*Parameter            `json:"parameters,omitempty"`
	Responses           map[string]*Response       `json:"responses,omitempty"`
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
	AllowEmptyValue bool                   `json:"allowEmptyValue,omitempty"`
	Schema
}

type StringOrArray []string

func (s StringOrArray) MarshalJSON() ([]byte, error) {
	str := ""
	if len(s) == 1 {
		str = s[0]
	} else {
		arr := make([]string, 0, len(s))
		for _, i := range s {
			arr = append(arr, i)
		}
		return json.Marshal(arr)
	}
	return json.Marshal(str)
}

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

func (s *SchemaOrArray) MarshalJSON() ([]byte, error) {
	if s.Schemas != nil {
		return json.Marshal(s.Schemas)
	}
	return json.Marshal(s.Schema)
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

type Response struct {
	Extensions  map[string]interface{} `json:"-"`
	Ref         string                 `json:"$ref,omitempty"`
	Description string                 `json:"description,omitempty"`
	Schema      *Schema                `json:"schema,omitempty"`
	Headers     map[string]Header      `json:"headers,omitempty"`
	Examples    map[string]interface{} `json:"examples,omitempty"`
}

type Header struct {
//	CommonValidations
	//	SimpleSchema
	Schema
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

func OasError(format string, args ...interface{}) error {
	return fmt.Errorf("(OAS v2) - "+format, args...)
}

func ConvertToV3(v2 *OpenAPI) (*oas3.OpenAPI, error) {
	v3 := &oas3.OpenAPI{
		Components: &oas3.Components{},
	}
	if v2.Info == nil {
		return nil, OasError("Missing required field 'info' in Swaggger object")
	}
	v3.Info = convertInfo(v2.Info)
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
	return v3, nil
}

func convertInfo(v2Info *Info) *oas3.Info {
	info := &oas3.Info{
		Title:       v2Info.Title,
		Description: v2Info.Description,
		Version:     v2Info.Version,
		Extensions:  v2Info.Extensions,
	}
	if v2Info.Contact != nil {
		info.Contact = &oas3.Contact{
			Email: v2Info.Contact.Email,
			Name:  v2Info.Contact.Name,
			URL:   v2Info.Contact.URL,
		}
	}
	return info
}

func convertOperation(op2 *Operation) (*oas3.Operation, error) {
	op3 := &oas3.Operation{
		Extensions: op2.Extensions,
		OperationId: op2.OperationId,
		Description: op2.Description,
		Tags: op2.Tags,
	}
	var params []*oas3.Parameter
	for _, param2 := range op2.Parameters {
		param3, err := convertParam(param2)
		if err != nil {
			return nil, err
		}
		params = append(params, param3)
	}
	op3.Parameters = params

	responses := make(map[string]*oas3.Response, 0)
	for scode, resp2 := range op2.Responses {
		resp3 := &oas3.Response {
			Description: resp2.Description,
		}
		if len(resp2.Headers) > 0 {
			headers := make(map[string]*oas3.Header, 0)
			for k, v2 := range resp2.Headers {
				v3 := &oas3.Header{
					Description: v2.Description,
				}
				tmp, err := convertSchema("", &v2.Schema)
				if err != nil {
					return nil, err
				}
				v3.Schema = tmp
				headers[k] = v3
			}
			resp3.Headers = headers
		}
		tmp, err := convertSchema("", resp2.Schema)
		if err != nil {
			return nil, err
		}
		resp3.Content = make(oas3.Content, 0)
		resp3.Content["application/json"] = &oas3.MediaType{
			Schema: tmp,
		}
		responses[scode] = resp3
	}
	if len(responses) > 0 {
		op3.Responses = responses
	}
	return op3, nil
}


func convertParam(p2 *Parameter) (*oas3.Parameter, error) {
	schema, err := convertSchema(p2.Name, &p2.Schema)
	if err != nil {
		return nil, err
	}
	p3 := &oas3.Parameter{
		Extensions: p2.Extensions,
		Description: p2.Description,
		Name: p2.Name,
		In: p2.In,
		Required: p2.Required,
		Schema: schema,
		AllowEmptyValue: p2.AllowEmptyValue,
	}
	//p3.Style
	//p3.Explode
	//p3.AllowReserved
	//p3.Deprecated
	//p3.Example
	//p3.Examples
	//p3.Content
	return p3, nil
}

func convertPath(v2Path *PathItem) (*oas3.PathItem, error) {
	v3Path := &oas3.PathItem{
		Extensions: v2Path.Extensions,
	}
	var err error
	if v2Path.Get != nil {
		v3Path.Get, err = convertOperation(v2Path.Get)
	} else if v2Path.Put != nil {
		v3Path.Put, err = convertOperation(v2Path.Put)
	} else if v2Path.Post != nil {
		v3Path.Post, err = convertOperation(v2Path.Post)
	} else if v2Path.Put != nil {
		v3Path.Put, err = convertOperation(v2Path.Put)
	} else if v2Path.Delete != nil {
		v3Path.Delete, err = convertOperation(v2Path.Delete)
	} else if v2Path.Options != nil {
		v3Path.Options, err = convertOperation(v2Path.Options)
	} else if v2Path.Head != nil {
		v3Path.Head, err = convertOperation(v2Path.Head)
	} else if v2Path.Patch != nil {
		v3Path.Patch, err = convertOperation(v2Path.Patch)
	}
	return v3Path, err
}

func convertSchema(xname string, v2 *Schema) (*oas3.Schema, error) {
	var err error
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
			if v2.Enum != nil {
				v3.Enum = v2.Enum
			}
			v3.Pattern = v2.Pattern
			if v2.MinLength != nil {
				v3.MinLength = uint64(*v2.MinLength)
			}
			if v2.MaxLength != nil {
				tmp := uint64(*v2.MaxLength)
				v3.MaxLength = &tmp
			}
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
				v3.Required = v2.Required
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
			return nil, fmt.Errorf("Unrecognized OAS type: %q", stype)
		}
	}
	return v3, nil
}

func ConvertFromV3(v3 *oas3.OpenAPI) (*OpenAPI, error) {
	v2 := &OpenAPI{
		Swagger: "2.0",
		Info: &Info{
			Title:          v3.Info.Title,
			Version:        v3.Info.Version,
			Description:    v3.Info.Description,
			TermsOfService: v3.Info.TermsOfService,
		},
	}
	if v3.Info.Contact != nil {
		v2.Info.Contact = &ContactInfo{
			Name:  v3.Info.Contact.Name,
			URL:   v3.Info.Contact.URL,
			Email: v3.Info.Contact.Email,
		}
	}
	if v3.Info.License != nil {
		v2.Info.License = &License{
			Name: v3.Info.License.Name,
			URL:  v3.Info.License.URL,
		}
	}
	v2.Definitions = make(map[string]*Schema, 0)
	for tname, schema := range v3.Components.Schemas {
		schema2, err := schemaFromV3(schema)
		if err != nil {
			return nil, err
		}
		v2.Definitions[tname] = schema2
	}
	return v2, nil
}

func refFromV3(ref string) (string, error) {
	prefix := "#/components/schemas/"
	if strings.HasPrefix(ref, prefix) {
		tname := ref[len(prefix):]
		return "#/definitions/" + tname, nil
	}
	return "", fmt.Errorf("Cannot convert reference to v2: %q", ref)
}

func schemaFromV3(schema3 *oas3.Schema) (*Schema, error) {
	var err error
	schema2 := &Schema{}
	if schema3.Ref != "" {
		schema2.Ref, err = refFromV3(schema3.Ref)
		if err != nil {
			return nil, err
		}
		return schema2, nil
	}
	schema2.Type = append(schema2.Type, schema3.Type) //?
	switch schema3.Type {
	case "string":
		schema2.Pattern = schema3.Pattern
		if schema3.MinLength > 0 {
			tmp := int64(schema3.MinLength)
			schema2.MinLength = &tmp
		}
		if schema3.MaxLength != nil {
			tmp := int64(*schema3.MaxLength)
			schema2.MaxLength = &tmp
		}
	case "array":
		itemType, err := schemaFromV3(schema3.Items)
		if err != nil {
			return nil, err
		}
		schema2.Items = &SchemaOrArray{
			Schema: itemType,
		}
	case "object":
		schema2.Required = schema3.Required
		schema2.Properties = make(map[string]*Schema, 0)
		for k, v := range schema3.Properties {
			schema2.Properties[k], err = schemaFromV3(v)
			if err != nil {
				return nil, err
			}
		}
	default:
		panic("fix me")
		//		return nil, fmt.Errorf("NYI")
	}
	return schema2, nil
}
