package smithy

//type Model struct {
//	ast *AST
//}

/*
import(
//	"encoding/json"
//	"fmt"
//	"github.com/boynton/sadl"
)

//the model can be output directly as JSON.

type Node interface{}

//traits:
// sensitive
// documentation
// length(min, max) - for strings, collections
// pattern (for strings)
// private
// required (struct field)
// deprecated
// enum (the "name" attribute of each item is the variable name to use in codegen, the key is the value
// uniqueItems (for a list)
// range (for numbers)
// and of course: user-defined traits, made by applying the @trait trait to another type

type Length struct {
	Min *int64 `json:"min,omitempty"`
	Max *int64 `json:"max,omitempty"`
}

type Deprecated struct {
	Message string `json:"message,omitempty"`
	Since string `json:"since,omitempty"`
}

type Item struct {
	Name string `json:"name"`
	Documentation string `json:"documentation,omitempty"`
	Tags []string `json:"tags,omitempty"`
}

type Traits struct {
	Documentation string `json:"smithy.api#documentation,omitempty"`
	Sensitive bool `json:"smithy.api#sensitive,omitempty"`
	Deprecated *Deprecated `json:"smithy.api#deprecated,omitempty"`
	Length *Length `json:"smithy.api#length,omitempty"`
	Pattern string `json:"smithy.api#pattern,omitempty"`
	Enum map[string]*Item `json:"smithy.api#enum,omitempty"`
	Idempotent bool `json:"smithy.api#idempotent,omitempty"`
	ReadOnly bool `json:"smithy.api#readonly,omitempty"`
	Required bool `json:"smithy.api#required,omitempty"`
	Paginated *Paginated `json:"smithy.api#paginated,omitempty"`
	References []*Ref `json:"smithy.api#references,omitempty"`
	Error string `json:"smithy.api#error,omitempty"`
	Http *Http `json:"smithy.api#http,omitempty"`
	HttpLabel bool `json:"smithy.api#httpLabel,omitempty"`
	HttpQuery string `json:"smithy.api#httpQuery,omitempty"`
	HttpPayload bool `json:"smithy.api#httpPayload,omitempty"`
	HttpHeader string `json:"smithy.api#httpHeader,omitempty"`
	HttpError int32 `json:"smithy.api#httpError,omitempty"`
	Protocols []Protocol `json:"smithy.api#protocols,omitempty"`
}

type Paginated struct {
	Items string `json:"items,omitempty"`
	InputToken string `json:"inputToken,omitempty"`
	OutputToken string `json:"outputToken,omitempty"`
	PageSize string `json:"pageSizeomitempty"`	
}

type Ref struct {
	Resource string `json:"resource,omitempty"`
}

type Member struct {
	Target string `json:"target"`
	Traits *Traits `json:"traits,omitempty"`
}

type Protocol struct {
	Name string `json:"name"`
	Auth []string `json:"auth,omitempty"`
	Tags []string `json:"tags,omitempty"`
}

type Http struct {
	Uri string `json:"uri,omitempty"`
	Method string `json:"method,omitempty"`
	Code int `json:"code,omitempty"`
}

type Shape struct {
	Type string `json:"type"`
	Members map[string]*Member `json:"members,omitempty"`
	Member *Member `json:"member,omitempty"`
	Key *Member `json:"key,omitempty"`
	Value *Member `json:"value,omitempty"`
	Identifiers map[string]*Member `json:"identifiers,omitempty"`
	Create *Member `json:"create,omitempty"`
	Put *Member `json:"put,omitempty"`
	Read *Member `json:"read,omitempty"`
	Update *Member `json:"update,omitempty"`
	Delete *Member `json:"delete,omitempty"`
	List *Member `json:"list,omitempty"`
	Operations []*Member `json:"operations,omitempty"`
	CollectionOperations []*Member `json:"collectionOperations,omitempty"`
	Version string `json:"version,omitempty"`
	Resources []*Member `json:"resources,omitempty"`
	Input *Member `json:"input,omitempty"`
	Output *Member `json:"output,omitempty"`
	Errors []*Member `json:"errors,omitempty"`
	Traits *Traits `json:"traits,omitempty"`
}

type Model struct {
	Version      string `json:"smithy"`
	Metadata     map[string]Node `json:"metadata,omitempty"`
	Shapes       map[string]*Shape `json:"shapes,omitempty"`
}

*/

/*type Namespace struct {
	Shapes map[string]*Shape `json:"shapes,omitempty"`
	Traits map[string]Node `json:"traits,omitempty"`
}
*/


/*
func (model *Model) MarshalJSON() ([]byte, error) {
	tmp := make(map[string]interface{}, 0)
	tmp["smithy"] = model.Version
	if len(model.Metadata) > 0 {
		tmp["metadata"] = model.Metadata
	}
	tmp["shapes"] = model.Shapes
	return json.Marshal(tmp)
}

func (model *Model) UnmarshalJSON(b []byte) error {
	tmp := make(map[string]interface{}, 0)
	err := json.Unmarshal(b, &tmp)
	if err != nil {
		return err
	}
	model.Namespaces = make(map[string]*Namespace, 0)
	if v, ok := tmp["smithy"].(string); ok {
		model.Version = v
	}
	if m, ok := tmp["metadata"]; ok {
		b1, err := json.Marshal(m)
		if err != nil {
			return err
		}
		var metadata map[string]Node
		err = json.Unmarshal(b1, &metadata)
		if err != nil {
			return err
		}
		model.Metadata = metadata
	}
	delete(tmp, "smithy")
	delete(tmp, "metadata")
	b2, err := json.Marshal(tmp)
	if err != nil {
		return err
	}
	fmt.Println(sadl.Pretty(tmp))
	panic("here")
	var namespaces map[string]*Namespace
	err = json.Unmarshal(b2, &namespaces)
	if err != nil {
		return err
	}
	model.Namespaces = namespaces
	return nil
}
*/
