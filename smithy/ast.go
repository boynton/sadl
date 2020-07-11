package smithy

import (
	"strings"
)

const SmithyVersion = "1.0"
const UnspecifiedNamespace = "example"
const UnspecifiedVersion = "0.0"

type AST struct {
	Version  string                 `json:"smithy"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Shapes   map[string]*Shape      `json:"shapes,omitempty"`
}

func (ast *AST) NamespaceAndServiceVersion() (string, string, string) {
	var namespace, name, version string
	for k, v := range ast.Shapes {
		if strings.HasPrefix(k, "smithy.") || strings.HasPrefix(k, "aws.") {
			continue
		}
		i := strings.Index(k, "#")
		if i >= 0 {
			namespace = k[:i]
		}
		if v.Type == "service" {
			version = v.Version
			name = k[i+1:]
			break
		}
	}
	return namespace, name, version
}

type Shape struct {
	Type   string                 `json:"type"`
	Traits map[string]interface{} `json:"traits,omitempty"` //service, resource, operation, apply

	//List and Set
	Member *Member `json:"member,omitempty"`

	//Map
	Key   *Member `json:"key,omitempty"`
	Value *Member `json:"value,omitempty"`

	//Structure and Union
	Members map[string]*Member `json:"members,omitempty"` //keys must be case-insensitively unique. For union, len(Members) > 0,

	//Resource
	Identifiers          map[string]*ShapeRef `json:"identifiers,omitempty"`
	Create               *ShapeRef            `json:"create,omitempty"`
	Put                  *ShapeRef            `json:"put,omitempty"`
	Read                 *ShapeRef            `json:"read,omitempty"`
	Update               *ShapeRef            `json:"update,omitempty"`
	Delete               *ShapeRef            `json:"delete,omitempty"`
	List                 *ShapeRef            `json:"list,omitempty"`
	CollectionOperations []*ShapeRef          `json:"collectionOperations,omitempty"`

	//Resource and Service
	Operations []*ShapeRef `json:"operations,omitempty"`
	Resources  []*ShapeRef `json:"resources,omitempty"`

	//Operation
	Input  *ShapeRef `json:"input,omitempty"`
	Output *ShapeRef `json:"output,omitempty"`
	//	Errors []*Member `json:"errors,omitempty"`
	Errors []*ShapeRef `json:"errors,omitempty"`

	//Service
	Version string `json:"version,omitempty"`
}

type ShapeRef struct {
	Target string `json:"target"`
}

type Member struct {
	Target string                 `json:"target"`
	Traits map[string]interface{} `json:"traits,omitempty"`
}

// Constraint traits - https://awslabs.github.io/smithy/1.0/spec/core/constraint-traits.html
//enum
/*type EnumTraitItem struct {
	Value         string   `json:"value"`
	Name          string   `json:"name,omitempty"` //must be an Identifier, preferably upper case
	Documentation string   `json:"documentation,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Deprecated    bool     `json:"deprecated,omitempty"`
}
type EnumTrait []*EnumTraitItem
*/

//idRef
//length
//pattern
//private
//range

//required

//uniqueItems

//Documentation traits - https://awslabs.github.io/smithy/1.0/spec/core/documentation-traits.html
//deprecated
//type DeprecatedTrait struct {
//	Message string `json:"message,omitempty"`
//	Since   string `json:"since,omitempty"`
//}

//documentation
//examples
//externalDocumentation
//sensitive
//since
//tags
//title
//unstable

// Type Refinement traits - https://awslabs.github.io/smithy/1.0/spec/core/type-refinement-traits.html
//box
//error
type ErrorTrait int

// Protocol traits - https://awslabs.github.io/smithy/1.0/spec/core/protocol-traits.html
//protocolDefinition
//jsonName
//mediaType
//timestampFormat

// Authentication traits - https://awslabs.github.io/smithy/1.0/spec/core/auth-traits.html
//authDefinition
//httpBasicAuth
//httpDigestAuth
//httpBearerAuth
//httpApiKeyAuth
//optionalAuth
//auth

//Behavior traits - https://awslabs.github.io/smithy/1.0/spec/core/behavior-traits.html
//idempotencyToken
//idempotent
//readonly
//retryable
//paginated

//Resource traits - https://awslabs.github.io/smithy/1.0/spec/core/resource-traits.html
//noReplace
//references
//resourceIdentifier

//Streaming traits - https://awslabs.github.io/smithy/1.0/spec/core/stream-traits.html
//streaming
//requiresLength
//eventHeader
//eventPayload

//-------------- HTTP protocol binding traits - https://awslabs.github.io/smithy/1.0/spec/core/http-traits.html
type HttpTrait struct {
	Method string `json:"method"`
	Uri    string `json:"uri"`
	Code   int    `json:"code,omitempty"` //default to 200
}

// httpError
type HttpErrorTrait int

// httpHeader
// httpLabel
// httpPayload
// httpPrefixHeaders
// httpQuery
// cors

//-------------- XML binding traits - https://awslabs.github.io/smithy/1.0/spec/core/xml-traits.html
// xmlAttribute
// xmlFlattened
// xmlName
// xmlNamespace

//-------------- Endpoint traits - https://awslabs.github.io/smithy/1.0/spec/core/endpoint-traits.html
// endpoint
// hostLabel

//--------------
