package sadl

import (
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
)

const indentAmount = "    "

type SadlGenerator struct {
	Generator
	Model *Model
}

func NewGenerator(model *Model, outdir string) *SadlGenerator {
	gen := &SadlGenerator{}
	gen.OutDir = outdir
	gen.Model = model
	return gen
}

func DecompileSadl(model *Model) string {
	g := NewGenerator(model, "")
	sadlSource := g.Generate()
	if g.Err != nil {
		panic(g.Err.Error())
	}
	return sadlSource
}

func (g *SadlGenerator) Generate() string {
	funcMap := template.FuncMap{
		"blockComment": func(s string) string {
			if s == "" {
				return ""
			}
			return g.FormatComment("", s, 100, true)
		},
		"annotations": func(annos map[string]string) string {
			s := ""
			if len(annos) > 0 {
				s = "\n"
				for k, v := range annos {
					if v == "" {
						s += fmt.Sprintf("%s\n", k)
					} else {
						s += fmt.Sprintf("%s %q\n", k, v)
					}
				}
			}
			return s
		},
		"typedef": func(td *TypeDef) string {
			return fmt.Sprintf("type %s %s\n", td.Name, g.sadlTypeSpec(&td.TypeSpec, nil, ""))
		},
		"action": func(act *ActionDef) string {
			out := ""
			exc := ""
			if act.Output != "" {
				out = " " + act.Output
			}
			if len(act.Exceptions) > 0 {
				exc = " " + strings.Join(act.Exceptions, ", ")
			}
			if exc != "" {
				exc = " except" + exc
			}
			return fmt.Sprintf("action %s(%s)%s%s\n", act.Name, act.Input, out, exc)
		},
		"http": func(hact *HttpDef) string {
			return g.sadlHttpSpec(hact)
		},
		"example": func(ed *ExampleDef) string {
			return fmt.Sprintf("example %s %s\n", ed.Target, Pretty(ed.Example))
		},
	}
	g.Begin()
	g.EmitTemplate("sadl", sadlTemplate, g.Model, funcMap)
	return g.End()
}

func (g *SadlGenerator) CreateSadlSource() {
	sadlSource := g.Generate()
	if g.Err != nil {
		panic(g.Err.Error())
	}
	if g.OutDir == "" {
		fmt.Println(sadlSource)
	} else {
		path := filepath.Join(g.OutDir, g.Model.Name+".sadl")
		g.WriteFile(path, sadlSource)
	}
}

func (g *SadlGenerator) sadlTypeSpec(ts *TypeSpec, opts []string, indent string) string {
	switch ts.Type {
	case "Enum":
		//Q: what if this is a required field, defined inline in a struct?!
		s := "Enum {\n"
		for _, el := range ts.Elements {
			com := ""
			if el.Comment != "" {
				com = " // " + el.Comment
			}
			s = s + indent + indentAmount + el.Symbol + com + "\n"
		}
		return s + indent + "}"
	case "String":
		if ts.Pattern != "" {
			opts = append(opts, fmt.Sprintf("pattern=%q", ts.Pattern))
		}
		if ts.MinSize != nil {
			opts = append(opts, fmt.Sprintf("minsize=%d", *ts.MinSize))
		}
		if ts.MaxSize != nil {
			opts = append(opts, fmt.Sprintf("maxsize=%d", *ts.MaxSize))
		}
		if ts.Values != nil {
			opts = append(opts, fmt.Sprintf("values=%s", stringList(ts.Values)))
		}
		sopts := ""
		if len(opts) > 0 {
			sopts = " (" + strings.Join(opts, ", ") + ")"
		}
		return fmt.Sprintf("String%s", sopts)
	case "Int8", "Int16", "Int32", "Int64", "Float32", "Float64", "Decimal":
		if ts.Min != nil {
			opts = append(opts, fmt.Sprintf("min=%v", ts.Min.String()))
		}
		if ts.Max != nil {
			opts = append(opts, fmt.Sprintf("max=%v", ts.Max.String()))
		}
		sopts := ""
		if len(opts) > 0 {
			sopts = " (" + strings.Join(opts, ", ") + ")"
		}
		return fmt.Sprintf("%s%s", ts.Type, sopts)
	case "Array":
		if ts.MinSize != nil {
			opts = append(opts, fmt.Sprintf("minsize=%d", *ts.MinSize))
		}
		if ts.MaxSize != nil {
			opts = append(opts, fmt.Sprintf("maxsize=%d", *ts.MaxSize))
		}
		sopts := ""
		if len(opts) > 0 {
			sopts = " (" + strings.Join(opts, ", ") + ")"
		}
		return fmt.Sprintf("Array<%s>%s", ts.Items, sopts)
	case "Map":
		if ts.MinSize != nil {
			opts = append(opts, fmt.Sprintf("minsize=%d", *ts.MinSize))
		}
		if ts.MaxSize != nil {
			opts = append(opts, fmt.Sprintf("maxsize=%d", *ts.MaxSize))
		}
		sopts := ""
		if len(opts) > 0 {
			sopts = " (" + strings.Join(opts, ", ") + ")"
		}
		return fmt.Sprintf("Map<%s,%s>%s", ts.Keys, ts.Items, sopts)
	case "Struct":
		sopt := ""
		if len(ts.Fields) > 0 {
			s := fmt.Sprintf("Struct%s {\n", sopt)
			for _, fd := range ts.Fields {
				com := ""
				bcom := ""
				if fd.Comment != "" {
					if len(fd.Comment) > 100 {
						bcom = g.FormatComment(indent+indentAmount, fd.Comment, 100, false)
					} else {
						com = " // " + fd.Comment
					}
				}
				fopts := []string{}
				if fd.Required {
					fopts = append(fopts, "required")
				}
				for aname, aval := range fd.Annotations {
					fopts = append(fopts, fmt.Sprintf("%s=%q", aname, aval))
				}
				s += fmt.Sprintf("%s%s%s %s%s\n", bcom, indent+indentAmount, fd.Name, g.sadlTypeSpec(&fd.TypeSpec, fopts, indent+indentAmount), com)
			}
			return s + indent + "}"
		}
		return fmt.Sprintf("Struct {\n}")
	case "Union":
		if true {
			s := fmt.Sprintf("Union {\n")
			for _, fd := range ts.Variants {
				com := ""
				bcom := ""
				if fd.Comment != "" {
					if len(fd.Comment) > 100 {
						bcom = g.FormatComment(indentAmount, fd.Comment, 100, false)
					} else {
						com = " // " + fd.Comment
					}
				}
				fopts := []string{}
				for aname, aval := range fd.Annotations {
					fopts = append(fopts, fmt.Sprintf("%s=%q", aname, aval))
				}
				s += fmt.Sprintf("%s%s%s %s%s\n", bcom, indentAmount, fd.Name, g.sadlTypeSpec(&fd.TypeSpec, fopts, indent+indentAmount), com)
			}
			return s + "}"
		} else {
			s := fmt.Sprintf("Union<")
			for i, v := range ts.Variants {
				if i != 0 {
					s += ","
				}
				s += v.Type
			}
			return s + ">"
		}
	default:
		sopts := ""
		if len(opts) > 0 {
			sopts = " (" + strings.Join(opts, ", ") + ")"
		}
		return fmt.Sprintf("%s%s", ts.Type, sopts)
	}
}

func (g *SadlGenerator) sadlHttpSpec(hact *HttpDef) string {
	var opts []string
//	if hact.Name != "" {
//		if hact.Name != actionName(hact) {
//			opts = append(opts, "action="+hact.Name)
//		}
//	}
	if len(hact.Annotations) > 0 {
		for k, v := range hact.Annotations {
			opts = append(opts, fmt.Sprintf("%s=%q", k, v))
		}
	}
	opt := ""
	if len(opts) > 0 {
		opt = " (" + strings.Join(opts, ", ") + ")"
	}
	var s string
	if hact.Name == "" {
		s = fmt.Sprintf("http %s %q%s {\n", hact.Method, hact.Path, opt)
	} else {
		s = fmt.Sprintf("action %s %s %q%s {\n", hact.Name, hact.Method, hact.Path, opt)
	}
	for _, in := range hact.Inputs {
		s += indentAmount + g.sadlParamSpec(in)
	}
	bcom := ""
	if hact.Expected == nil {
		hact.Expected = &HttpExpectedSpec{
			Status: 200,
		}
	}
	if hact.Expected.Comment != "" {
		bcom = g.FormatComment(indentAmount, hact.Expected.Comment, 100, false)
	}
	s += fmt.Sprintf("\n%s%sexpect %d {\n", bcom, indentAmount, hact.Expected.Status)
	for _, out := range hact.Expected.Outputs {
		s += indentAmount + indentAmount + g.sadlParamSpec(out)
	}
	s += "   }\n"
	if len(hact.Exceptions) > 0 {
		s += "\n"
		for _, exc := range hact.Exceptions {
			bcom := ""
			if exc.Comment != "" {
				bcom = g.FormatComment(indentAmount, exc.Comment, 100, false)
			}
			if exc.Status == 0 {
				s += fmt.Sprintf("%s%sexcept %s\n", bcom, indentAmount, exc.Type)
			} else {
				//todo: header outputs
				s += fmt.Sprintf("%s%sexcept %d %s\n", bcom, indentAmount, exc.Status, exc.Type)
			}
		}
	}
	s += "}\n"
	return s
}

func (g *SadlGenerator) sadlParamSpec(ps *HttpParamSpec) string {
	var opts []string
	if ps.Required {
		opts = append(opts, "required")
	}
	if ps.Default != nil {
		opts = append(opts, "default="+ToString(ps.Default))
	}
	if ps.Header != "" {
		opts = append(opts, fmt.Sprintf("header=%q", ps.Header))
	}
	opt := ""
	if len(opts) > 0 {
		opt = " (" + strings.Join(opts, ", ") + ")"
	}
	com := ""
	bcom := ""
	if ps.Comment != "" {
		if len(ps.Comment) > 100 {
			bcom = g.FormatComment("   ", ps.Comment, 100, false)[3:] + "   "
		} else {
			com = " // " + ps.Comment
		}
	}
	ts := g.sadlTypeSpec(&ps.TypeSpec, nil, indentAmount)
	return bcom + ps.Name + " " + ts + opt + com + "\n"
}

func stringList(lst []string) string {
	result := "["
	for i, s := range lst {
		if i != 0 {
			result += ","
		}
		result += fmt.Sprintf("%q", s)
	}
	return result + "]"
}

const sadlTemplate = `{{if .Comment}}{{blockComment .Comment}}{{end}}{{if .Namespace}}namespace {{.Namespace}}
{{end}}{{if .Name}}name {{.Name}}
{{end}}{{if .Version}}version "{{.Version}}"
{{end}}{{annotations .Annotations}}{{if .Types}}{{range .Types}}
{{blockComment .Comment}}{{typedef .}}{{end}}{{end}}{{if .Actions}}{{range .Actions}}
{{blockComment .Comment}}{{action .}}{{end}}{{end}}{{if .Http}}{{range .Http}}
{{blockComment .Comment}}{{http .}}{{end}}{{end}}{{if .Examples}}{{range .Examples}}
{{blockComment .Comment}}{{example .}}{{end}}{{end}}`
