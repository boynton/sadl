package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func ToString(obj interface{}) string {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(&obj); err != nil {
		return fmt.Sprint(obj)
	}
	s := buf.String()
	s = strings.Trim(s, " \n")
	return string(s)
}

func AsStruct(v interface{}) map[string]interface{} {
	if v != nil {
		if m, ok := v.(map[string]interface{}); ok {
			return m
		}
	}
	return nil
}

func AsArray(v interface{}) []interface{} {
	if v != nil {
		if a, ok := v.([]interface{}); ok {
			return a
		}
	}
	return nil
}

func AsStringArray(v interface{}) []string {
	var sa []string
	for _, i := range AsArray(v) {
		if s, ok := i.(*string); ok {
			sa = append(sa, *s)
		} else {
			return nil
		}
	}
	return sa
}

func AsString(v interface{}) string {
	if v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func AsBool(v interface{}) bool {
	if v != nil {
		if b, isBool := v.(bool); isBool {
			return b
		}
		return true
	}
	return false
}

func AsInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int32:
		return int(n)
	case int:
		return n
	}
	return 0
}

func AsInt64(v interface{}) int64 {
	if n, ok := v.(float64); ok {
		return int64(n)
	}
	return 0
}

func AsFloat64(v interface{}) float64 {
	if n, ok := v.(float64); ok {
		return n
	}
	return 0
}

func Get(m map[string]interface{}, key string) interface{} {
	if m != nil {
		if v, ok := m[key]; ok {
			return v
		}
	}
	return nil
}

func GetString(m map[string]interface{}, key string) string {
	return AsString(m[key])
}
func GetBool(m map[string]interface{}, key string) bool {
	return AsBool(m[key])
}
func GetInt(m map[string]interface{}, key string) int {
	return AsInt(m[key])
}
func GetInt64(m map[string]interface{}, key string) int64 {
	return AsInt64(m[key])
}
func GetArray(m map[string]interface{}, key string) []interface{} {
	return AsArray(m[key])
}
func GetStruct(m map[string]interface{}, key string) map[string]interface{} {
	return AsStruct(m[key])
}
