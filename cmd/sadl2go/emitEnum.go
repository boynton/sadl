package main

import(
	"text/template"

	"github.com/boynton/sadl"

)
func (gen *GoGenerator) emitEnumType(td *sadl.TypeDef) {
	if gen.err != nil {
		return
	}
	funcMap := template.FuncMap{
		"openBrace": func() string { return "{" },
	}
	gen.emitTemplate("enumType", enumTemplate, td, funcMap)
}

const enumTemplate = `type {{.Name}} int

const (
    _ {{.Name}} = iota{{range .Elements}}
    {{.Symbol}}{{end}}
)

var names{{.Name}} = []string{{openBrace}}{{range .Elements}}
    {{.Symbol}}: "{{.Symbol}}",{{end}}
}

func (e {{.Name}}) String() string {
    return names{{.Name}}[e]
}

func (e {{.Name}}) MarshalJSON() ([]byte, error) {
    return json.Marshal(e.String())
}

func (e *{{.Name}}) UnmarshalJSON(b []byte) error {
    var s string
    err := json.Unmarshal(b, &s)
    if err == nil {
        for v, s2 := range names{{.Name}} {
            if s == s2 {
                *e = {{.Name}}(v)
                return nil
             }
        }
        err = fmt.Errorf("Bad enum symbol for type {{.Name}}: %s", s)
    }
    return err
}
`

