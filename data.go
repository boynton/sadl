package sadl

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/boynton/sadl/util"
	"github.com/ghodss/yaml"
)

type Data struct {
	value interface{}
}

func (data *Data) String() string {
	return util.Pretty(data.value)
}

func DataToFile(data *Data, path string) error {
	raw := []byte(data.String())
	return ioutil.WriteFile(path, raw, 0660)
}

func DataFromFile(path string) (*Data, error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ext := filepath.Ext(path)
	var value map[string]interface{}
	if ext == ".yaml" {
		err = yaml.Unmarshal(raw, &value)
	} else {
		err = json.Unmarshal(raw, &value)
	}
	if err != nil {
		return nil, err
	}
	return &Data{value: value}, nil
}

func AsDecimal(v interface{}) *Decimal {
	switch n := v.(type) {
	case Decimal:
		return &n
	case *Decimal:
		return n
	default:
		return nil
	}
}

func (data *Data) Put(key string, value interface{}) {
	m := data.AsMap()
	if m != nil {
		m[key] = value
	}
}

func (data *Data) AsMap() map[string]interface{} {
	if data != nil {
		if m, ok := data.value.(map[string]interface{}); ok {
			return m
		}
	}
	return nil
}

func (data *Data) Get(keys ...string) interface{} {
	return data.get(keys)
}

func (data *Data) get(keys []string) interface{} {
	m := data.AsMap()
	if m != nil {
		if len(keys) == 1 {
			key := keys[0]
			if v, ok := m[key]; ok {
				return v
			}
		} else {
			for i, key := range keys {
				if v, ok := m[key]; ok {
					if i < len(keys)-1 {
						if mm, ok := v.(map[string]interface{}); ok {
							m = mm
						} else {
							return nil
						}
					} else {
						return v
					}
				}
			}
		}
	}
	return nil
}

func (data *Data) Has(keys ...string) bool {
	return data.get(keys) != nil
}

func (data *Data) GetString(keys ...string) string {
	return util.AsString(data.get(keys))
}

func (data *Data) GetBool(keys ...string) bool {
	return util.AsBool(data.get(keys))
}

func (data *Data) GetArray(key string) []interface{} {
	return util.AsArray(data.Get(key))
}

func (data *Data) GetStruct(key string) map[string]interface{} {
	return util.AsStruct(data.Get(key))
}

func (data *Data) GetData(key string) *Data {
	v := data.Get(key)
	if v == nil {
		return nil
	}
	return &Data{value: v}
}

func GetDecimal(m map[string]interface{}, key string) *Decimal {
	return AsDecimal(m[key])
}
