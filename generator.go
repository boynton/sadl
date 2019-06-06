package sadl

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"
)

type Generator struct {
	Model  *Model
	OutDir string
	Err    error
	buf    bytes.Buffer
	file   *os.File
	writer *bufio.Writer
}

func (gen *Generator) Emit(s string) {
	if gen.Err == nil && gen.writer != nil {
		_, gen.Err = gen.writer.WriteString(s)
	}
}

func (gen *Generator) Begin() {
	if gen.Err != nil {
		return
	}
	gen.buf.Reset()
	gen.writer = bufio.NewWriter(&gen.buf)
}

func (gen *Generator) End() string {
	if gen.Err != nil || gen.writer == nil {
		return ""
	}
	gen.writer.Flush()
	return gen.buf.String()
}

func (gen *Generator) WriteFile(path string, content string) {
	if gen.FileExists(path) {
		//if debug, echo it anyway?
		fmt.Printf("[%s already exists, not overwriting]\n", path)
		return
	}
	f, err := os.Create(path)
	if err != nil {
		gen.Err = err
		return
	}
	defer f.Close()
	writer := bufio.NewWriter(f)
	_, gen.Err = writer.WriteString(content)
	writer.Flush()
}

func (gen *Generator) EmitTemplate(name string, tmplSource string, data interface{}, funcMap template.FuncMap) {
	if gen.Err != nil {
		fmt.Println("emitTemplate("+name+"): already have an error, do not continue:", gen.Err)
		return
	}
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)
	tmpl, err := template.New(name).Funcs(funcMap).Parse(tmplSource)
	if err != nil {
		fmt.Println("emitTemplate -> cannot create template:", gen.Err)
		gen.Err = err
		return
	}
	err = tmpl.Execute(writer, data)
	if err != nil {
		gen.Err = err
		return
	}
	writer.Flush()
	gen.Emit(b.String())
}

func (gen *Generator) Capitalize(s string) string {
	return strings.ToUpper(s[0:1]) + s[1:]
}

func (gen *Generator) Uncapitalize(s string) string {
	return strings.ToLower(s[0:1]) + s[1:]
}

func (gen *Generator) FileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func (gen *Generator) FormatComment(indent, comment string, maxcol int, extraPad bool) string {
	prefix := "// "
	left := len(indent)
	if maxcol <= left {
		return indent + prefix + comment + "\n"
	}
	tabbytes := make([]byte, 0, left)
	for i := 0; i < left; i++ {
		tabbytes = append(tabbytes, ' ')
	}
	tab := string(tabbytes)
	prefixlen := len(prefix)
	var buf bytes.Buffer
	col := 0
	lines := 1
	tokens := strings.Split(comment, " ")
	for _, tok := range tokens {
		toklen := len(tok)
		if col+toklen >= maxcol {
			buf.WriteString("\n")
			lines++
			col = 0
		}
		if col == 0 {
			buf.WriteString(tab)
			buf.WriteString(prefix)
			buf.WriteString(tok)
			col = left + prefixlen + toklen
		} else {
			buf.WriteString(" ")
			buf.WriteString(tok)
			col += toklen + 1
		}
	}
	buf.WriteString("\n")
	emptyPrefix := strings.Trim(prefix, " ")
	pad := ""
	if extraPad {
		pad = tab + emptyPrefix + "\n"
	}
	return pad + buf.String() + pad
}
