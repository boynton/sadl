package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
)

func Capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[0:1]) + s[1:]
}

func Uncapitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[0:1]) + s[1:]
}

func IsSymbolChar(ch rune, first bool) bool {
	if IsLetter(ch) {
		return true
	}
	if first {
		return false
	}
	return IsDigit(ch) || ch == '_'
}

func IsSymbol(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range s {
		if !IsSymbolChar(c, i == 0) {
			return false
		}
	}
	return true
}

func IsWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n'
}

func IsDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func IsLetter(ch rune) bool {
	return IsUppercaseLetter(ch) || IsLowercaseLetter(ch)
}

func IsUppercaseLetter(ch rune) bool {
	return ch >= 'A' && ch <= 'Z'
}

func IsLowercaseLetter(ch rune) bool {
	return ch >= 'a' && ch <= 'z'
}

func Kind(v interface{}) string {
	return fmt.Sprintf("%v", reflect.ValueOf(v).Kind())
}

var Verbose bool

func Debug(args ...interface{}) {
	if Verbose {
		max := len(args) - 1
		for i := 0; i < max; i++ {
			fmt.Print(str(args[i]))
		}
		fmt.Println(str(args[max]))
	}
}

func str(arg interface{}) string {
	return fmt.Sprintf("%v", arg)
}

func Equivalent(obj1 interface{}, obj2 interface{}) bool {
	return Pretty(obj1) == Pretty(obj2)
}

func Pretty(obj interface{}) string {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(&obj); err != nil {
		return fmt.Sprint(obj)
	}
	return string(buf.String())
}

func BaseFileName(path string) string {
	fname := filepath.Base(path)
	//	fname := FileName(path)
	n := strings.LastIndex(fname, ".")
	if n < 1 {
		return fname
	}
	return fname[:n]
}
