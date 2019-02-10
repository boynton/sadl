package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/parse"
)

func main() {
	pOutdir := flag.String("dir", "", "output directory for generated source")
	pPackage := flag.String("package", "", "Go package for generated source")
	pRuntime := flag.Bool("runtime", false, "Use SADL runtime library for base types. If false, they are generated in the target package")
	flag.Parse()
	argv := flag.Args()
	argc := len(argv)
	if argc == 0 {
		fmt.Fprintf(os.Stderr, "usage: sadl2go -dir outdir -package go_package_name -runtime some_model.sadl\n")
		os.Exit(1)
	}
	model, err := parse.File(argv[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "*** %v\n", err)
		os.Exit(1)
	}
	name := filepath.Base(argv[0])
	n := strings.LastIndex(name, ".")
	if n > 0 {
		name = name[:n]
	}
	gen := newGoGenerator(model, name, *pOutdir, *pPackage, *pRuntime)
	for _, td := range model.Types {
		gen.emitType(td)
	}
	if gen.needsDecimalType() {
		gen.emitDecimalType()
	}
	gen.emitJsonUtil()
	gen.finish()
}

type GoGenerator struct {
	model    *sadl.Model
	name string
	outdir   string
	pkgname  string
	runtime bool
	pkgpath  string
	imports  []string
	header   string
	buf bytes.Buffer
	file     *os.File
	writer   *bufio.Writer
	err      error
}

func newGoGenerator(model *sadl.Model, name, outdir, pkg string, runtime bool) *GoGenerator {
	gen := &GoGenerator{
		model:    model,
		name: name,
		outdir:   outdir,
		pkgname:  pkg,
		runtime: runtime,
		header:   "//\n// Generated by sadl2go\n//\n",
	}
	gen.pkgpath = filepath.Join(gen.outdir, gen.pkgname)
	if gen.pkgpath != "" {
		err := os.MkdirAll(gen.pkgpath, 0755)
		if err != nil {
			gen.err = err
		}
	}
	gen.writer = bufio.NewWriter(&gen.buf)
	return gen
}

func (gen *GoGenerator) emit(s string) {
	if gen.err == nil {
		_, err := gen.writer.WriteString(s)
		if err != nil {
			gen.err = err
		}
	}
}

func (gen *GoGenerator) needsDecimalType() bool {
	if !gen.runtime {
		for _, pack := range gen.imports {
			if pack == "math/big" {
				return true
			}
		}
	}
	return false
}

func (gen *GoGenerator) finish() {
	if gen.err == nil {
		gen.writer.Flush()
		path := filepath.Join(gen.pkgpath, gen.name + "_model.go")
		f, err := os.Create(path)
		if err != nil {
			gen.err = err
			return
		}
		gen.file = f
		gen.writer = bufio.NewWriter(f)
		gen.emit(gen.header)
		if gen.pkgname == "" {
			gen.pkgname = "main"
		}
		gen.emit("package " + gen.pkgname + ";\n\n")
		fmt.Println("imports", gen.imports)
		if len(gen.imports) > 0 {
			gen.emit("import(\n")
			for _, pack := range gen.imports {
				gen.emit("    \"" + pack + "\"\n")
			}
			gen.emit(")\n\n")
		}
		gen.buf.WriteTo(gen.writer)
		gen.writer.Flush()
		gen.file.Close()
	}
}

func adjoin(lst []string, val string) []string {
	for _, s := range lst {
		if val == s {
			return lst
		}
	}
	return append(lst, val)
}

func (gen *GoGenerator) addImport(fullReference string) {
	gen.imports = adjoin(gen.imports, fullReference)
}


func (gen *GoGenerator) emitJsonUtil() {
	if gen.err != nil {
		return
	}
	gen.addImport("encoding/json")
	gen.addImport("fmt")
	gen.emit(goJsonUtil)
}

var goJsonUtil = `
func Pretty(obj interface{}) string {
	j, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		return fmt.Sprint(obj)
	}
	return string(j)
}
`

func (gen *GoGenerator) createGoFile(name string) {
	if gen.err != nil {
		return
	}
	path := filepath.Join(gen.pkgpath, name + ".go")
   f, err := os.Create(path)
   if err != nil {
		gen.err = err
		return
   }
	gen.file = f
   gen.writer = bufio.NewWriter(f)
	gen.emit(gen.header)
	if gen.pkgname != "" {
		gen.emit("package " + gen.pkgname + ";\n\n")
	}
}

func (gen *GoGenerator) requiredPrimitiveTypeName(nameOptional, nameRequired string, required bool) string {
	if !required {
		return nameOptional
	}
	return nameRequired
}

func (gen *GoGenerator) nativeTypeName(ts *sadl.TypeSpec, name string) string {
	//not happy about optional values in Go. For now, just let zero values be omitted for optional fields.
	switch name {
	case "Int8", "Int16", "Int32", "Int64", "Float32", "Float64", "Bool", "String":
		return uncapitalize(name)
	case "Decimal":
		if gen.runtime {
			gen.addImport("github.com/boynton/sadl")
			return "*sadl." + name
		} else {
			gen.addImport("math/big")
			return "*" + name
		}
	case "Array":
		its := gen.model.FindType(ts.Items)
		return "[]" + gen.nativeTypeName(&its.TypeSpec, ts.Items)
	case "Map":
		its := gen.model.FindType(ts.Items)
		kts := gen.model.FindType(ts.Keys)
		return "map[" + gen.nativeTypeName(&kts.TypeSpec, ts.Keys) +"]" + gen.nativeTypeName(&its.TypeSpec, ts.Items)
	default:
		//must be a app-defined class. Parser should have already verified its existence
		td := gen.model.FindType(name)
		if td == nil {
			panic("Unresolved type, parser should have caught this: " + name)
		}
		if td.Type == "Struct" {
			name = "*" + name
		}
		return name
	}
}

func (gen *GoGenerator) emitType(td *sadl.TypeDef) {
	gen.emit("\n//\n// " + td.Name + "\n//\n")
	switch td.Type {
	case "Struct":
		gen.emitStructType(td)
	case "Quantity":
		gen.emitQuantityType(td)
	case "Enum":
		gen.emitEnumType(td)
	default:
		panic("Check this")
		//do nothing, i.e. a String subclass
	}
	
}

func (gen *GoGenerator) emitStructType(td *sadl.TypeDef) {
	gen.emit("type " + td.Name + " struct {\n")
	for _, fd := range td.Fields {
		fname := capitalize(fd.Name)
		ftype := gen.nativeTypeName(&fd.TypeSpec, fd.Type)
		anno := " `json:\"" + fd.Name
		if !fd.Required {
			anno = anno + ",omitempty"
		}
		anno = anno + "\"`"
		gen.emit("    " + fname + " " + ftype + anno + "\n")
	}
	gen.emit("}\n\n")
}

func (gen *GoGenerator) emitQuantityType(td *sadl.TypeDef) {
	panic("NYI")
}

func (gen *GoGenerator) emitEnumType(td *sadl.TypeDef) {
	panic("NYI")
}

func capitalize(s string) string {
	return strings.ToUpper(s[0:1]) + s[1:]
}

func uncapitalize(s string) string {
	return strings.ToLower(s[0:1]) + s[1:]
}

