package oas

import(
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghodss/yaml"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/importers/oas/oas3"
	"github.com/boynton/sadl/importers/oas/oas2"
)

var _ = sadl.Pretty

type Oas struct {
	V3 *oas3.OpenAPI
}

func (oas *Oas) MarshalJSON() ([]byte, error) {
	return json.Marshal(oas.V3)
}


func DetermineVersion(data []byte, format string) (string, error) {
	var raw map[string]interface{}
	var err error
	switch format {
	case "json":
      err = json.Unmarshal(data, &raw)
	case "yaml":
      err = yaml.Unmarshal(data, &raw)
	default:
		err = fmt.Errorf("Unsupported file format: %q. Only \"json\" and \"yaml\" are supported.", format)
   }
	if err != nil {
		return "", err
	}
	if v, ok := raw["openapi"]; ok {
		if s, ok := v.(string); ok {
			return s, nil
		}
	}
	if v, ok := raw["swagger"]; ok {
		if s, ok := v.(string); ok {
			return s, nil
		}
	}
	return "", fmt.Errorf("Cannot find an 'openapi' in the specified %s file to determine the version", format)
}

func Parse(data []byte, format string) (*Oas, error) {
	version, err := DetermineVersion(data, format)
	if err != nil {
		return nil, err
	}
	oas := &Oas{}
	if strings.HasPrefix(version, "3.") {
		oas.V3, err = oas3.Parse(data, format)
		return oas, nil
	} else if strings.HasPrefix(version, "2.") {
		v2, err := oas2.Parse(data, format)
		if err == nil {
			oas.V3, err = oas2.ConvertToV3(v2)
		}
		return oas, err
	}
	return nil, fmt.Errorf("Unsupported version of OpenAPI Spec: %s", version)
}

