package smithy

import(
	"encoding/json"
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

type Traitable struct {
	Documentation string `json:"documentation,omitempty"`
	Sensitive bool `json:"sensitive,omitempty"`
	Deprecated *Deprecated `json:"deprecated,omitempty"`
	Length *Length `json:"length,omitempty"`
	Pattern string `json:"pattern,omitempty"`
	Enum map[string]*Item `json:"enum,omitempty"`
}

type Member struct {
	Traitable
	Target string `json:"target"`
	Required bool `json:"required,omitempty"`
}

type Identifiers struct {
	Id string `json:"id,omitempty"`
}

type Protocol struct {
	Name string `json:"name"`
	Auth []string `json:"auth,omitempty"`
	Tags []string `json:"tags,omitempty"`
}

type Shape struct {
	Traitable
	Type string `json:"type"`
	Members map[string]*Member `json:"members,omitempty"`
	Member *Member `json:"member,omitempty"`
	Key *Member `json:"key,omitempty"`
	Value *Member `json:"value,omitempty"`
	Trait bool `json:"trait,omitempty"`
	Identifiers map[string]string `json:"identifiers,omitempty"`
	Create string `json:"create,omitempty"`
	Put string `json:"put,omitempty"`
	Read string `json:"read,omitempty"`
	Update string `json:"update,omitempty"`
	Delete string `json:"delete,omitempty"`
	List string `json:"list,omitempty"`
	Operations []string `json:"operations,omitempty"`
	CollectionOperations []string `json:"collectionOperations,omitempty"`
	Version string `json:"version,omitempty"`
	Resources []string `json:"resources,omitempty"`
	Protocols []Protocol `json:"protocols,omitempty"`
	Input string `json:"input,omitempty"`
	Output string `json:"output,omitempty"`
	Errors []string `json:"errors,omitempty"`
	Idempotent bool `json:"idempotent,omitempty"`
	ReadOnly bool `json:"readonly,omitempty"`
}

type Namespace struct {
	Shapes map[string]*Shape `json:"shapes,omitempty"`
	Traits map[string]Node `json:"traits,omitempty"`
}

type Model struct {
	Version      string
	Namespaces   map[string]*Namespace
	Metadata     map[string]Node
//	Metadata    map[string]Node `json:"metadata,omitempty"`
}

func (model *Model) MarshalJSON() ([]byte, error) {
	tmp := make(map[string]interface{}, 0)
	tmp["smithy"] = model.Version
	if len(model.Metadata) > 0 {
		tmp["metadata"] = model.Metadata
	}
	for k, v := range model.Namespaces {
		tmp[k] = v
	}
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
	var namespaces map[string]*Namespace
	err = json.Unmarshal(b2, &namespaces)
	if err != nil {
		return err
	}
	model.Namespaces = namespaces
	return nil
}
