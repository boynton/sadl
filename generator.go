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
	Config *Data
	OutDir string
	Err    error
	buf    bytes.Buffer
	file   *os.File
	writer *bufio.Writer
}

func (gen *Generator) GetConfigString(k string, defaultValue string) string {
	if !gen.Config.Has(k) {
		return defaultValue
	}
	return gen.Config.GetString(k)
}

func (gen *Generator) GetConfigBool(k string, defaultValue bool) bool {
	if !gen.Config.Has(k) {
		return defaultValue
	}
	return gen.Config.GetBool(k)
}

func (gen *Generator) GetConfigInt(k string, defaultValue int) int {
	if !gen.Config.Has(k) {
		return defaultValue
	}
	return gen.Config.GetInt(k)
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
	if !gen.Config.GetBool("force-overwrite") && gen.FileExists(path) {
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
	return FormatComment(indent, "// ", comment, maxcol, extraPad)
}
