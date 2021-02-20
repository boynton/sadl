package golang

import (
	"fmt"
	"github.com/boynton/sadl"
	"strings"
	"text/template"
)

var _ = sadl.Pretty

func (gen *Generator) CreateServer() {
	if gen.Err != nil {
		return
	}
	gen.Begin()
	gen.EmitServerAdaptor()
	content := gen.End()
	fname := sadl.Uncapitalize(gen.Name) + "_server.go"
	gen.WriteGoFile(fname, content, gen.Pkg)

}

func (gen *Generator) EmitServerAdaptor() {
	gen.imports = nil
	gen.addImport("bytes")
	gen.addImport("encoding/json")
	gen.addImport("fmt")
	gen.addImport("github.com/gorilla/mux")
	gen.addImport("github.com/gorilla/handlers")
	gen.addImport("io")
	gen.addImport("log")
	gen.addImport("net/http")
	gen.addImport("net/url")
	gen.addImport("os")
	gen.addImport("strconv")
	gen.addImport("strings")
	gen.addImport("time")
	funcMap := template.FuncMap{
		"openBrace":   func() string { return "{" },
		"backquote":   func() string { return "`" },
		"serviceName": func() string { return sadl.Capitalize(gen.Name) },
		"adaptorName": func() string { return sadl.Uncapitalize(gen.Name) + "Adaptor" },
		"reqTypeName": func(hd *sadl.HttpDef) string { return gen.RequestTypeName(hd) },
		"resTypeName": func(hd *sadl.HttpDef) string { return gen.ResponseTypeName(hd) },
		"methodName":  func(hd *sadl.HttpDef) string { return sadl.Capitalize(hd.Name) },
		"routePath": func(hd *sadl.HttpDef) string {
			path := hd.Path
			i := strings.Index(path, "?")
			if i >= 0 {
				path = path[0:i]
			}
			return path
		},
		"defaultLiteral": func(any interface{}) string {
			if any == nil {
				return "nil"
			}
			switch val := any.(type) {
			case string:
				return fmt.Sprintf("%q", val)
			case *string:
				return fmt.Sprintf("%q", *val)
			default:
				return fmt.Sprint(val)
			}
		},
		"inputs": func(hd *sadl.HttpDef) string {
			//1. the struct is already declared, as is the err variable.
			form := false
			for _, in := range hd.Inputs {
				if in.Query != "" {
					form = true
					break
				}
			}
			s := ""
			if form {
				s = "\terr := r.ParseForm()\n\tif err != nil {\n\t\terrorResponse(w, 400, fmt.Sprint(err))\n\t\treturn\n\t}\n"
			}
			for _, in := range hd.Inputs {
				name := sadl.Capitalize(in.Name)
				if in.Query != "" {
					s = s + "\treq." + name + " = " + gen.paramAccessor(in, fmt.Sprintf("r.Form.Get(%q)", in.Query)) + "\n"
				} else if in.Header != "" {
					s = s + "\treq." + name + " = " + gen.paramAccessor(in, fmt.Sprintf("r.Header.Get(%q)", in.Header)) + "\n"
				} else if in.Path {
					s = s + "\treq." + name + " = " + gen.paramAccessor(in, fmt.Sprintf("mux.Vars(r)[%q]", in.Name)) + "\n"
				} else {
					s = s + "\terr := json.NewDecoder(r.Body).Decode(&req." + name + ")\n"
					s = s + "\tif err != nil {\n\t\terrorResponse(w, 400, fmt.Sprint(err))\n\t\treturn\n\t}\n"
				}
			}
			return s
		},
		"outputs": func(hd *sadl.HttpDef) string {
			s := ""
			for _, out := range hd.Expected.Outputs {
				name := sadl.Capitalize(out.Name)
				if out.Header != "" {
					s = s + "\t\tw.Header().Add(\"" + out.Header + "\", normalizeHeaderValue(\"" + out.Header + "\", res." + name + "))\n"
				}
			}
			return s
		},
		"exceptions": func(hd *sadl.HttpDef) string {
			s := ""
			for _, e := range hd.Exceptions {
				s = s + "\t\tcase " + gen.nativeType(e.Type) + ":\n"
				s = s + fmt.Sprintf("\t\t\tjsonResponse(w, %d, err)\n", e.Status)
			}
			return s
		},
		"expectedResult": func(hd *sadl.HttpDef) string {
			switch hd.Expected.Status {
			case 204, 304:
				return "_"
			default:
				return "res"
			}
		},
		"expectedEntity": func(hd *sadl.HttpDef) string {
			for _, out := range hd.Expected.Outputs {
				name := sadl.Capitalize(out.Name)
				if out.Header == "" {
					return "res." + name
				}
			}
			return "nil"
		},
		"signature": func(hd *sadl.HttpDef) string {
			name := gen.Capitalize(hd.Name)
			reqType := gen.RequestTypeName(hd)
			resType := gen.ResponseTypeName(hd)
			return "func " + name + "(req *" + reqType + ") (*" + resType + ", error)"
		},
		"capitalize": func(s string) string { return gen.Capitalize(s) },
	}
	gen.EmitTemplate("server", serverTemplate, gen, funcMap)
}

func (gen *Generator) paramAccessor(in *sadl.HttpParamSpec, v string) string {
	coerceTo := gen.nativeType(in.Type)
	if strings.HasPrefix(coerceTo, "*") {
		coerceTo = "&" + coerceTo[1:]
	}
	bt := gen.baseType(in.Type)
	switch bt {
	case "String":
		vv := fmt.Sprintf("stringParam(%s, %q)", v, stringDefault(in.Default))
		if coerceTo != bt {
			vv = fmt.Sprintf("%s(%s)", coerceTo, vv)
		}
		return vv
	case "Timestamp":
		return fmt.Sprintf("timestampParam(%s, %s)", v, timestampDefault(in.Default))
	case "Int32", "Int8", "Int16", "Int64":
		vv := fmt.Sprintf("intParam(%s, %d)", v, intDefault(in.Default))
		if coerceTo != bt {
			vv = fmt.Sprintf("%s(%s)", coerceTo, vv)
		}
		return vv
		/*
			case "Float32", "Float64":
			case "Decimal":
			case "UUID":
		*/
	default:
		panic("Fix this: " + in.Type)
	}
}

func (gen *Generator) nativeType(name string) string {
	td := gen.Model.FindType(name)
	if td == nil {
		return name
	}
	return gen.nativeTypeName(&td.TypeSpec, name)
}

func (gen *Generator) baseType(tname string) string {
	td := gen.Model.FindType(tname)
	if td == nil {
		return ""
	}
	return td.Type
}

func stringDefault(any interface{}) string {
	if any == nil {
		return ""
	}
	switch v := any.(type) {
	case string:
		return v
	default:
		return fmt.Sprint(any)
	}
}

func timestampDefault(any interface{}) string {
	switch v := any.(type) {
	case *sadl.Timestamp:
		return fmt.Sprintf("%q", v.String())
	case string:
		ts, err := sadl.ParseTimestamp(v)
		if err == nil {
			return fmt.Sprintf("%q", ts.String())
		}
	}
	return "nil"
}

func intDefault(v interface{}) int64 {
	if v != nil {
		switch n := v.(type) {
		case *sadl.Decimal:
			return n.AsInt64()
		}
	}
	return 0
}

var serverTemplate = `
type {{adaptorName}} struct {
	impl {{serviceName}}
}

{{range .Model.Http}}
func (handler *{{adaptorName}}) {{methodName .}}Handler(w http.ResponseWriter, r *http.Request) {
	req := new({{reqTypeName .}})
{{inputs .}}
	{{expectedResult .}}, err := handler.impl.{{methodName .}}(req)
	if err != nil {
		switch err.(type) {
{{exceptions .}}		default:
			jsonResponse(w, 500, &serverError{Message: fmt.Sprint(err)})
		}
	} else {
{{outputs .}}		jsonResponse(w, {{.Expected.Status}}, {{expectedEntity .}})
	}
}
{{end}}

func InitServer(impl {{serviceName}}, baseURL string) http.Handler {
	adaptor := &{{adaptorName}}{
		impl: impl,
	}
   u, err := url.Parse(strings.TrimSuffix(baseURL, "/"))
   if err != nil {
      log.Fatal(err)
   }	
   b := u.Path
	r := mux.NewRouter()
{{range .Model.Http}}
	r.HandleFunc(b+"{{routePath .}}", func (w http.ResponseWriter, r *http.Request) {
      adaptor.{{methodName .}}Handler(w, r)
	}).Methods("{{.Method}}"){{end}}
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 404, &serverError{Message: fmt.Sprintf("Not Found: %s", r.URL.Path)})
	})
	return r
}

func stringParam(val string, def string) string {
	if val == "" {
		return def
	}
	return val
}

func timestampParam(val string, def *Timestamp) *Timestamp {
	if val != "" {
		ts, err := ParseTimestamp(val)
		if err == nil {
			return &ts
		}
	}
	return def
}

func intParam(val string, def int64) int64 {
	if val != "" {
		i, err := strconv.ParseInt(val, 10, 64)
		if err == nil {
			return i
		}
	}
	return def
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

func Json(obj interface{}) string {
   buf := new(bytes.Buffer)
   enc := json.NewEncoder(buf)
   enc.SetEscapeHTML(false)
   if err := enc.Encode(&obj); err != nil {
      return fmt.Sprint(obj)
   }
   return string(buf.String())
}

func jsonResponse(w http.ResponseWriter, status int, entity interface{}) {
   w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	io.WriteString(w, Pretty(entity))
}

func errorResponse(w http.ResponseWriter, status int, message string) {
	jsonResponse(w, status, &serverError{Error: http.StatusText(status), Message: message})
}

func normalizeHeaderValue(key string, value interface{}) string {
   switch strings.ToLower(key) {
   case "last-modified", "date", "expires":
      switch ts := value.(type) {
      case time.Time:
         return timestampRfc2616(ts)
      case *Timestamp:
         return timestampRfc2616(ts.Time)
      case string:
			return ts //just trust the application
      default:
         fmt.Sprintf("Noncompliant value for %s header: %v", key, ts)
			return fmt.Sprint(ts) //rather than fail
      }
   }
   return fmt.Sprint(value)
}

func intFromString(s string) int64 {
	var n int64 = 0
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n
}

func floatFromString(s string) float64 {
	var n float64 = 0
	_, _ = fmt.Sscanf(s, "%g", &n)
	return n
}

func timestampRfc3339(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.UTC().Format(time.RFC3339Nano)
}

func timestampRfc2616(ts time.Time) string {	
	//i.e. 
	if ts.IsZero() {
		return ""
	}
	return ts.UTC().Format(http.TimeFormat)
}

// FoldHttpHeaderName adapts to the Go misfeature: all headers are
// canonicalized as Capslike-This (for a header "CapsLike-this").
func FoldHttpHeaderName(name string) string {
	return http.CanonicalHeaderKey(name)
}

type serverError struct {
   Error string  {{backquote}}json:"error"{{backquote}}
	Message string {{backquote}}json:"message"{{backquote}}
}

func WebLog(h http.Handler) http.Handler {
	return handlers.CombinedLoggingHandler(os.Stdout, h)
}

func AllowCors(next http.Handler, host string) http.Handler {
   return handlers.CORS(handlers.AllowedOrigins([]string{"*"}), handlers.AllowedHeaders([]string{"Content-Type", "api_key", "Authorization"}), handlers.AllowedMethods([]string{"GET","PUT","DELETE","POST","OPTIONS"}))(next)
}
`
