package sadl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

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

const BLACK = "\033[0;0m"
const RED = "\033[0;31m"
const YELLOW = "\033[0;33m"
const BLUE = "\033[94m"
const GREEN = "\033[92m"

func formattedAnnotation(filename string, source string, prefix string, msg string, tok *Token, color string, contextSize int) string {
	highlight := color + "\033[1m"
	restore := BLACK + "\033[0m"
	if source != "" && contextSize >= 0 && tok != nil {
		lines := strings.Split(source, "\n")
		line := tok.Line - 1
		begin := max(0, line-contextSize)
		end := min(len(lines), line+contextSize+1)
		context := lines[begin:end]
		tmp := ""
		for i, l := range context {
			if i+begin == line {
				toklen := len(tok.Text)
				if toklen > 0 {
					if tok.Type == STRING {
						toklen = len(fmt.Sprintf("%q", tok.Text))
					} else if tok.Type == LINE_COMMENT {
						toklen = toklen + 2
					}
					left := ""
					mid := l
					right := ""
					if tok.Start > 0 && toklen > 1 {
						left = l[:tok.Start-1]
						mid = l[tok.Start-1 : tok.Start-1+toklen]
						right = l[tok.Start-1+toklen:]
					}
					tmp += fmt.Sprintf("%3d\t%v", i+begin+1, left)
					tmp += fmt.Sprintf("%s%v%s", highlight, mid, restore)
					tmp += fmt.Sprintf("%v\n", right)
				} else {
					tmp += fmt.Sprintf("%3d\t%v\n", i+begin+1, l)
				}
			} else {
				tmp += fmt.Sprintf("%3d\t%v\n", i+begin+1, l)
			}
		}
		if tok != nil {
			if filename != "" {
				return fmt.Sprintf("%s%s:%d:%d: %s%s%s\n%s", prefix, path.Base(filename), tok.Line, tok.Start, highlight, msg, restore, tmp)
			}
			return fmt.Sprintf("%s%s%s%s\n%s", prefix, highlight, msg, restore, tmp)
		} else {
			return fmt.Sprintf("%s%d:%d: %s", prefix, tok.Line, tok.Start, msg)
		}
	}
	return fmt.Sprintf("%s: %s", prefix, msg)
}

func max(n1 int, n2 int) int {
	if n1 > n2 {
		return n1
	}
	return n2
}

func min(n1 int, n2 int) int {
	if n1 < n2 {
		return n1
	}
	return n2
}

func capitalize(s string) string {
	return strings.ToUpper(s[0:1]) + s[1:]
}

func uncapitalize(s string) string {
	return strings.ToLower(s[0:1]) + s[1:]
}

func (model *Model) FindExampleType(ex *ExampleDef) (*TypeSpec, error) {
	lst := strings.Split(ex.Target, ".")
	theType := lst[0]
	lst = lst[1:]
	var ts *TypeSpec
	t := model.FindType(theType)
	if t != nil {
		ts = &t.TypeSpec
	} else {
		//http requests and responses are not quite like structs, although inputs and expected outputs are of type StructFieldDef
		if strings.HasSuffix(theType, "Request") {
			httpName := uncapitalize(theType[:len(theType)-len("Request")])
			h := model.FindHttp(httpName)
			if h != nil {
				if len(lst) > 0 {
					var tmp *TypeSpec
					fname := lst[0]
					for _, in := range h.Inputs {
						if in.Name == fname {
							lst = lst[1:]
							tmp = &in.TypeSpec
							break
						}
					}
					if tmp == nil {
						return nil, fmt.Errorf("Unknown http input '%s' when dereferencing example target: %s", fname, ex.Target)
					}
					ts = tmp
				} else {
					//return nil, fmt.Errorf("NYI: example target of the top level response type, which is synthesized: %s", theType)
					//let it just not validate for now
					return nil, nil
				}
			}
		} else if strings.HasSuffix(theType, "Response") {
			httpName := uncapitalize(theType[:len(theType)-len("Response")])
			h := model.FindHttp(httpName)
			if h != nil {
				if len(lst) > 0 {
					var tmp *TypeSpec
					fname := lst[0]
					for _, out := range h.Expected.Outputs {
						if out.Name == fname {
							lst = lst[1:]
							tmp = &out.TypeSpec
							break
						}
					}
					if tmp == nil {
						return nil, fmt.Errorf("Unknown http input '%s' when dereferencing example target: %s", fname, ex.Target)
					}
					ts = tmp
				} else {
					//return nil, fmt.Errorf("NYI: example target of the top level response type, which is synthesized: %s", theType)
					//let it just not validate for now
					return nil, nil
				}
			}
		}
	}
	if ts == nil {
		return nil, fmt.Errorf("Undefined type '%s' in example: %s", theType, Pretty(ex))
	}
	for len(lst) > 0 {
		if ts.Type != "Struct" {
			return nil, fmt.Errorf("Cannot dereference a non-struct in example target: %v", ex)
		}
		fname := lst[0]
		lst = lst[1:]
		var field *StructFieldDef
		for _, fd := range ts.Fields {
			if fd.Name == fname {
				field = fd
				break
			}
		}
		if field == nil {
			return nil, fmt.Errorf("Unknown field '%s' when dereferencing example target: %s", fname, ex.Target)
		}
		ts = &field.TypeSpec
	}
	return ts, nil
}
