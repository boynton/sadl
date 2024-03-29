package sadl

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/ghodss/yaml"
)

type Data struct {
	value interface{}
}

func AsData(v interface{}) *Data {
	return &Data{value: v}
}

func NewData() *Data {
	return &Data{}
}

func (d Data) MarshalJSON() ([]byte, error) {
	return []byte(d.String()), nil
}

func (d *Data) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &d.value)
}

func (data *Data) String() string {
	return Pretty(data.value)
}

func DataToFile(data *Data, path string) error {
	raw := []byte(data.String())
	return ioutil.WriteFile(path, raw, 0660)
}

func DataFromFile(path string) (*Data, error) {
	var data *Data
	raw, err := ioutil.ReadFile(path)
	if err == nil {
		ext := filepath.Ext(path)
		var value map[string]interface{}
		if ext == ".yaml" {
			err = yaml.Unmarshal(raw, &value)
		} else {
			err = json.Unmarshal(raw, &value)
		}
		if err == nil {
			data = &Data{value: value}
		}
	}
	return data, err
}

func DataFromJsonString(raw string) (*Data, error) {
	var data *Data
	var value map[string]interface{}
	err := json.Unmarshal([]byte(raw), &value)
	if err == nil {
		data = &Data{value: value}
	}
	return data, err
}

func (data *Data) Put(key string, value interface{}) {
	if data.value == nil {
		data.value = make(map[string]interface{}, 0)
	}
	m := data.AsMap()
	if m != nil {
		m[key] = value
	}
}

func (data *Data) IsNil() bool {
	return data.value == nil
}

func (data *Data) AsMap() map[string]interface{} {
	if data != nil && data.value != nil {
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
	return AsString(data.get(keys))
}

func (data *Data) GetBool(keys ...string) bool {
	return AsBool(data.get(keys))
}

func (data *Data) GetInt(keys ...string) int {
	return AsInt(data.get(keys))
}

func (data *Data) GetInt64(keys ...string) int64 {
	return AsInt64(data.get(keys))
}

func (data *Data) GetFloat64(keys ...string) float64 {
	return AsFloat64(data.get(keys))
}

func (data *Data) GetArray(keys ...string) []interface{} {
	return AsArray(data.get(keys))
}

func (data *Data) GetStringArray(keys ...string) []string {
	var sa []string
	d := data.get(keys)
	for _, v := range AsArray(d) {
		sa = append(sa, AsString(v))
	}
	return sa
}

func (data *Data) GetMap(keys ...string) map[string]interface{} {
	return AsMap(data.get(keys))
}

func (data *Data) GetData(keys ...string) *Data {
	return &Data{value: data.get(keys)}
}

func (data *Data) GetDecimal(keys ...string) *Decimal {
	return AsDecimal(data.get(keys))
}
