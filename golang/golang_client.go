package golang

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/boynton/sadl"
)

var _ = sadl.Pretty

func (gen *Generator) CreateClient() {
	if gen.Err != nil {
		return
	}
	gen.Begin()
	gen.EmitClient()
	content := gen.End()
	fname := sadl.Uncapitalize(gen.Name) + "_client.go"
	gen.WriteGoFile(fname, content, gen.Pkg)
}

func (gen *Generator) EmitClient() {
	gen.imports = nil
	gen.addImport("encoding/json")
	gen.addImport("fmt")
	gen.addImport("net/http")
	//	gen.addImport("net/url")
	gen.addImport("strings")
	//	gen.addImport("time")
	funcMap := template.FuncMap{
		"openBrace":   func() string { return "{" },
		"clientName":  func() string { return sadl.Capitalize(gen.Name) + "Client" },
		"methodName":  func(hd *sadl.HttpDef) string { return sadl.Capitalize(hd.Name) },
		"reqTypeName": func(hd *sadl.HttpDef) string { return gen.RequestTypeName(hd) },
		"resTypeName": func(hd *sadl.HttpDef) string { return gen.ResponseTypeName(hd) },
		"inputs":      func(hd *sadl.HttpDef) string { return "// fix me: 'inputs'" },
		"outputs":     func(hd *sadl.HttpDef) string { return "// fix me: 'outputs'" },
		"exceptions":  func(hd *sadl.HttpDef) string { return "// fix me: 'exceptions'" },
		"requestEntityContentType": func(hd *sadl.HttpDef) string {
			switch hd.Method {
			case "PUT", "POST", "PATCH":
				return "\threq.Header.Add(\"Content-Type\", \"application/json\")"
			}
			return ""
		},
		"reqBodyReader": func(hd *sadl.HttpDef) string {
			switch hd.Method {
			case "PUT", "POST", "PATCH":
				for _, in := range hd.Inputs {
					if !in.Path && in.Query == "" && in.Header == "" {
						gen.addImport("bytes")
						return "bytes.NewReader([]byte(Json(req." + sadl.Capitalize(in.Name) + ")))"
					}
				}
			}
			return "nil"
		},
		"queryParams": func(hd *sadl.HttpDef) string {
			s := ""
			for _, in := range hd.Inputs {
				if in.Query != "" {
					s = s + "\tif req." + sadl.Capitalize(in.Name) + " != " + gen.notSet(in.Type) + " {\n"
					s = s + "\t\targs = append(args, fmt.Sprintf(\"" + in.Name + "=" + gen.typeFormat(in.Type) + "\", req." + sadl.Capitalize(in.Name) + "))\n"
					s = s + "\t}\n"
				}
			}
			return s
		},
		"methodSignature": func(hd *sadl.HttpDef) string {
			name := sadl.Capitalize(hd.Name)
			return name + "(req *" + name + "Request) (*" + name + "Response, error)"
		},
		"methodPath": func(hd *sadl.HttpDef) string {
			path := hd.Path
			i := strings.Index(path, "?")
			if i >= 0 {
				path = path[0:i]
			}
			return path
		},
		"expectedResults": func(hd *sadl.HttpDef) string {
			s := fmt.Sprintf("\tcase %d:\n", hd.Expected.Status)
			//			if hd.Expected.Status == 204 || hd.Expected.Status == 304 {
			//				if len(hd.Expected.Outputs) == 0 {
			//					return s + "\t\treturn nil, nil\n"
			//				}
			//			}
			s = s + "\t\tresponse := &" + sadl.Capitalize(hd.Name) + "Response{}\n"
			for _, out := range hd.Expected.Outputs {
				natType := gen.nativeType(out.Type)
				name := sadl.Capitalize(out.Name)
				if out.Header == "" {
					s = s + "\t\tvar entity " + natType + "\n"
					s = s + "\t\terr = json.NewDecoder(res.Body).Decode(&entity)\n"
					s = s + "\t\tif err != nil {\n"
					s = s + "\t\t\treturn nil, err\n"
					s = s + "\t\t}\n"
					s = s + "\t\tresponse." + name + " = entity\n"
				} else {
					i := "res.Header.Get(" + fmt.Sprintf("%q", out.Header) + ")" //fold header name consistently?
					switch natType {
					case "*Timestamp":
						i = "TimestampFromString(" + i + ")"
					default:
					}
					s = s + "\t\tresponse." + name + " = " + i + "\n"
				}
			}
			s = s + "\t\treturn response, nil\n"
			return s
		},
		"exceptionResults": func(hd *sadl.HttpDef) string {
			s := ""
			for _, es := range hd.Exceptions {
				natType := gen.nativeType(es.Type)
				s = s + fmt.Sprintf("\tcase %d:\n", es.Status)
				s = s + "\t\tvar errEntity " + natType + "\n"
				s = s + "\t\terr = json.NewDecoder(res.Body).Decode(&errEntity)\n"
				s = s + "\t\tif err != nil {\n"
				s = s + "\t\t\treturn nil, err\n"
				s = s + "\t\t}\n"
				s = s + "\t\treturn nil, errEntity\n"
			}
			return s
		},
	}
	gen.EmitTemplate("client", clientTemplate, gen, funcMap)
}

func (gen *Generator) notSet(sadlType string) string {
	bt := gen.baseType(sadlType)
	switch bt {
	case "String":
		return `""`
	case "Int8", "Int16", "Int32", "Int64":
		return "0"
	case "Float32", "Float64", "Decimal":
		return "0.0"
	case "Bool":
		return "false"
	default:
		return "nil"
	}
}

func (gen *Generator) typeFormat(sadlType string) string {
	bt := gen.baseType(sadlType)
	switch bt {
	case "String":
		return "%s"
	case "Int8", "Int16", "Int32", "Int64":
		return "%d"
	case "Float32", "Float64", "Decimal":
		return "&g"
	default:
		return "%v"
	}
}

var clientTemplate = `
type {{clientName}} struct {
	Target string
}

func NewClient(target string) (*CrudlClient, error) {
	return &CrudlClient{
		Target: target,
	}, nil
}
{{range .Model.Http}}
func (client *{{clientName}}) {{methodSignature .}} {
	url := client.Target + "{{methodPath .}}"
	var args []string
{{queryParams .}}	if len(args) > 0 {
		url = url + "?" + strings.Join(args, "&")
	}
	hreq, err := http.NewRequest("{{.Method}}", url, {{reqBodyReader .}})
	if err != nil {
		return nil, err
	}
{{requestEntityContentType .}}
	res, err := http.DefaultClient.Do(hreq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	switch res.StatusCode {
{{expectedResults .}}
{{exceptionResults .}}
	}
   return nil,fmt.Errorf("whoops")
}
{{end}}

`
