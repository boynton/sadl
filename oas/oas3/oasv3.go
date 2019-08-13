package oas3

import (
	"encoding/json"

	"github.com/ghodss/yaml"
)

func Parse(data []byte, format string) (*OpenAPI, error) {
	var err error
	v3 := &OpenAPI{}
	if format == "yaml" {
		err = yaml.Unmarshal(data, &v3)
	} else {
		err = json.Unmarshal(data, &v3)
	}
	if err != nil {
		return nil, err
	}
	return v3, nil
}

type OpenAPI struct {
	Extensions   map[string]interface{} `json:"-"`
	OpenAPI      string                 `json:"openapi"`           // Required
	Info         Info                   `json:"info"`              // Required
	Servers      []*Server              `json:"servers,omitempty"` //?change
	Paths        map[string]*PathItem   `json:"paths,omitempty"`   //?change
	Components   *Components            `json:"components,omitempty"`
	Security     []SecurityRequirement  `json:"security,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty"`
}

type SecurityRequirement map[string][]string

type Info struct {
	Extensions     map[string]interface{} `json:"-"`
	Title          string                 `json:"title,omitempty"`
	Description    string                 `json:"description,omitempty"`
	TermsOfService string                 `json:"termsOfService,omitempty"`
	Contact        *Contact               `json:"contact,omitempty"`
	License        *License               `json:"license,omitempty"`
	Version        string                 `json:"version,omitempty"`
}

type Contact struct {
	Extensions map[string]interface{} `json:"-"`
	Name       string                 `json:"name,omitempty"`
	URL        string                 `json:"url,omitempty"`
	Email      string                 `json:"email,omitempty"`
}

type License struct {
	Extensions map[string]interface{} `json:"-"`
	Name       string                 `json:"name,omitempty"`
	URL        string                 `json:"url,omitempty"`
}

type Server struct {
	Extensions  map[string]interface{}     `json:"-"`
	URL         string                     `json:"url,omitempty"`
	Description string                     `json:"description,omitempty"`
	Variables   map[string]*ServerVariable `json:"variables,omitempty"`
}

type ServerVariable struct {
	Extensions  map[string]interface{} `json:"-"`
	Enum        []interface{}          `json:"enum,omitempty"`
	Default     interface{}            `json:"default,omitempty"`
	Description string                 `json:"description,omitempty"`
}

//type Paths struct {
//	Extensions map[string]interface{} `json:"-"`
//	Paths map[string]PathItem `json:"paths,omitempty"`
//}

type PathItem struct {
	Extensions  map[string]interface{} `json:"-"`
	Summary     string                 `json:"summary,omitempty"`
	Description string                 `json:"description,omitempty"`
	Connect     *Operation             `json:"connect,omitempty"`
	Delete      *Operation             `json:"delete,omitempty"`
	Get         *Operation             `json:"get,omitempty"`
	Head        *Operation             `json:"head,omitempty"`
	Options     *Operation             `json:"options,omitempty"`
	Patch       *Operation             `json:"patch,omitempty"`
	Post        *Operation             `json:"post,omitempty"`
	Put         *Operation             `json:"put,omitempty"`
	Trace       *Operation             `json:"trace,omitempty"`
	Servers     []*Server              `json:"servers,omitempty"`
	Parameters  []*Parameter           `json:"parameters,omitempty"`
}

type Operation struct {
	Extensions  map[string]interface{} `json:"-"`
	Tags        []string               `json:"tags,omitempty"`
	Summary     string                 `json:"summary,omitempty"`
	Description string                 `json:"description,omitempty"`
	OperationID string                 `json:"operationId,omitempty"`
	Parameters  []*Parameter           `json:"parameters,omitempty"`
	RequestBody *RequestBody           `json:"requestBody,omitempty"`
	Responses   map[string]*Response   `json:"responses,omitempty"`
	Callbacks   map[string]*Callback   `json:"callbacks,omitempty"`
	Deprecated  bool                   `json:"deprecated,omitempty"`
	Security    []SecurityRequirement  `json:"security,omitempty"`
	Servers     []*Server              `json:"servers,omitempty"`
}

type Parameter struct {
	Extensions      map[string]interface{} `json:"-"`
	Name            string                 `json:"name,omitempty"`
	In              string                 `json:"in,omitempty"`
	Description     string                 `json:"description,omitempty"`
	Style           string                 `json:"style,omitempty"`
	Explode         *bool                  `json:"explode,omitempty"`
	AllowEmptyValue bool                   `json:"allowEmptyValue,omitempty"`
	AllowReserved   bool                   `json:"allowReserved,omitempty"`
	Deprecated      bool                   `json:"deprecated,omitempty"`
	Required        bool                   `json:"required,omitempty"`
	Schema          *Schema                `json:"schema,omitempty"`
	Example         interface{}            `json:"example,omitempty"`
	Examples        map[string]*Example    `json:"examples,omitempty"`
	Content         Content                `json:"content,omitempty"`
}

type Response struct {
	Extensions  map[string]interface{} `json:"-"`
	Description string                 `json:"description,omitempty"`
	Headers     map[string]*Header     `json:"headers,omitempty"`
	Content     Content                `json:"content,omitempty"`
	Links       map[string]*Link       `json:"links,omitempty"`
}

type Link struct {
	Extensions  map[string]interface{} `json:"-"`
	Description string                 `json:"description,omitempty"`
	Href        string                 `json:"href,omitempty"`
	OperationID string                 `json:"operationId,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Headers     map[string]*Schema     `json:"headers,omitempty"`
}

type Content map[string]*MediaType

type MediaType struct {
	Extensions map[string]interface{} `json:"-"`
	Schema     *Schema                `json:"schema,omitempty"`
	Example    interface{}            `json:"example,omitempty"`
	Examples   map[string]*Example    `json:"examples,omitempty"`
	Encoding   map[string]*Encoding   `json:"encoding,omitempty"`
}

type Example struct {
	Extensions    map[string]interface{} `json:"-"`
	Summary       string                 `json:"summary,omitempty"`
	Description   string                 `json:"description,omitempty"`
	Value         interface{}            `json:"value,omitempty"`
	ExternalValue string                 `json:"externalValue,omitempty"`
}

type Encoding struct {
	Extensions    map[string]interface{} `json:"-"`
	ContentType   string                 `json:"contentType,omitempty"`
	Headers       map[string]*Header     `json:"headers,omitempty"`
	Style         string                 `json:"style,omitempty"`
	Explode       bool                   `json:"explode,omitempty"`
	AllowReserved bool                   `json:"allowReserved,omitempty"`
}

type Header struct {
	Extensions  map[string]interface{} `json:"-"`
	Description string                 `json:"description,omitempty"`
	Schema      *Schema                `json:"schema,omitempty"`
}

type Schema struct {
	Ref          string        `json:"$ref,omitempty"`
	OneOf        []*Schema     `json:"oneOf,omitempty"`
	AnyOf        []*Schema     `json:"anyOf,omitempty"`
	AllOf        []*Schema     `json:"allOf,omitempty"`
	Not          *Schema       `json:"not,omitempty"`
	Type         string        `json:"type,omitempty"`
	Format       string        `json:"format,omitempty"`
	Description  string        `json:"description,omitempty"`
	Enum         []interface{} `json:"enum,omitempty"`
	Default      interface{}   `json:"default,omitempty"`
	Example      interface{}   `json:"example,omitempty"`
	ExternalDocs interface{}   `json:"externalDocs,omitempty"`

	// Array-related, here for struct compactness
	UniqueItems bool `json:"uniqueItems,omitempty"`
	// Number-related, here for struct compactness
	ExclusiveMin bool `json:"exclusiveMinimum,omitempty"`
	ExclusiveMax bool `json:"exclusiveMaximum,omitempty"`
	// Properties
	Nullable  bool        `json:"nullable,omitempty"`
	ReadOnly  bool        `json:"readOnly,omitempty"`
	WriteOnly bool        `json:"writeOnly,omitempty"`
	XML       interface{} `json:"xml,omitempty"`

	// Number
	Min        *float64 `json:"minimum,omitempty"`
	Max        *float64 `json:"maximum,omitempty"`
	MultipleOf *float64 `json:"multipleOf,omitempty"`

	// String
	MinLength uint64  `json:"minLength,omitempty"`
	MaxLength *uint64 `json:"maxLength,omitempty"`
	Pattern   string  `json:"pattern,omitempty"`

	// Array
	MinItems uint64  `json:"minItems,omitempty"`
	MaxItems *uint64 `json:"maxItems,omitempty"`
	Items    *Schema `json:"items,omitempty"`

	// Object
	Required   []string           `json:"required,omitempty"`
	Properties map[string]*Schema `json:"properties,omitempty"`
	MinProps   uint64             `json:"minProperties,omitempty"`
	MaxProps   *uint64            `json:"maxProperties,omitempty"`
	//broken: AdditionalPropertiesAllowed *bool `json:"additionalProperties,omitEmpty"`
	AdditionalProperties *Schema        `json:"additionalProperties,omitempty"`
	Discriminator        *Discriminator `json:"discriminator,omitempty"`

	PatternProperties string `json:"patternProperties,omitempty"`
}

type Discriminator struct {
	Extensions   map[string]interface{} `json:"-"`
	PropertyName string                 `json:"propertyName"`
	Mapping      map[string]string      `json:"mapping,omitempty"`
}

type Components struct {
	Extensions      map[string]interface{}     `json:"-"`
	Schemas         map[string]*Schema         `json:"schemas,omitempty"`
	Parameters      map[string]*Parameter      `json:"parameters,omitempty"`
	Headers         map[string]*Header         `json:"headers,omitempty"`
	RequestBodies   map[string]*RequestBody    `json:"requestBodies,omitempty"`
	Responses       map[string]*Response       `json:"responses,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty"`
	Examples        map[string]*Example        `json:"examples,omitempty"`
	Tags            []*Tag                     `json:"tags,omitempty"`
	Links           map[string]*Link           `json:"links,omitempty"`
	Callbacks       map[string]*Callback       `json:"callbacks,omitempty"`
}

type SecurityScheme struct {
	Extensions   map[string]interface{} `json:"-"`
	Type         string                 `json:"type,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Name         string                 `json:"name,omitempty"`
	In           string                 `json:"in,omitempty"`
	Scheme       string                 `json:"scheme,omitempty"`
	BearerFormat string                 `json:"bearerFormat,omitempty"`
	Flows        *OAuthFlows            `json:"flows,omitempty"`
}

type OAuthFlows struct {
	Extensions        map[string]interface{} `json:"-"`
	Implicit          *OAuthFlow             `json:"implicit,omitempty"`
	Password          *OAuthFlow             `json:"password,omitempty"`
	ClientCredentials *OAuthFlow             `json:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow             `json:"authorizationCode,omitempty"`
}

type OAuthFlow struct {
	Extensions       map[string]interface{} `json:"-"`
	AuthorizationURL string                 `json:"authorizationUrl,omitempty"`
	TokenURL         string                 `json:"tokenUrl,omitempty"`
	RefreshURL       string                 `json:"refreshUrl,omitempty"`
	Scopes           map[string]string      `json:"scopes"`
}

type Callback map[string]*PathItem

type RequestBody struct {
	Extensions  map[string]interface{} `json:"-"`
	Description string                 `json:"description,omitempty"`
	Required    bool                   `json:"required,omitempty"`
	Content     Content                `json:"content,omitempty"`
}

type Tag struct {
	Name         string                 `json:"name,omitempty"`
	Description  string                 `json:"description,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty"`
}

type ExternalDocumentation struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
}

// example of preserving extension when unmarshaling.
/*
func (p *Paths) UnmarshalJSON(data []byte) error {
	var res map[string]json.RawMessage
	if err := json.Unmarshal(data, &res); err != nil {
		return err
	}
	for k, v := range res {
		if strings.HasPrefix(strings.ToLower(k), "x-") {
			if p.Extensions == nil {
				p.Extensions = make(map[string]interface{})
			}
			var d interface{}
			if err := json.Unmarshal(v, &d); err != nil {
				return err
			}
			p.Extensions[k] = d
		}
		if strings.HasPrefix(k, "/") {
			if p.Paths == nil {
				p.Paths = make(map[string]PathItem)
			}
			var pi PathItem
			if err := json.Unmarshal(v, &pi); err != nil {
				return err
			}
			p.Paths[k] = pi
		}
	}
	return nil
}

// some examples of preserving the extensions on marshal. Unmarshal
func (p Paths) MarshalJSON() ([]byte, error) {
	tmp := make(map[string]interface{}, 0)
	for k, v := range p.Extensions {
		tmp[k] = v
	}
	for k, v := range p.Paths {
		tmp[k] = v
	}
	return json.Marshal(tmp)
}
*/

func (doc OpenAPI) MarshalJSON() ([]byte, error) {
	tmp := make(map[string]interface{}, 0)
	for k, v := range doc.Extensions {
		tmp[k] = v
	}
	tmp["openapi"] = doc.OpenAPI
	tmp["info"] = doc.Info
	if doc.Servers != nil {
		tmp["servers"] = doc.Servers
	}
	if doc.Paths != nil {
		tmp["paths"] = doc.Paths
	}
	if doc.Components != nil {
		tmp["components"] = doc.Components
	}
	if doc.Security != nil {
		tmp["security"] = doc.Security
	}
	if doc.ExternalDocs != nil {
		tmp["externalDocs"] = doc.ExternalDocs
	}
	return json.Marshal(tmp)
}
