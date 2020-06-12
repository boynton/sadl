package java

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/util"
)

type Generator struct {
	util.Generator
	Model          *sadl.Model
	Domain         string
	Name           string
	Package        string
	Header         string
	SourceDir      string
	ResourceDir    string
	UseLombok      bool
	UseGetters     bool
	UseInstants    bool
	UseMaven       bool
	Server         bool
	needTimestamps bool
	imports        []string
	serverData     *ServerData
}

func Export(model *sadl.Model, dir string, conf map[string]interface{}) error {
	gen := NewGenerator(model, dir, conf)
	for _, td := range model.Types {
		gen.CreatePojoFromDef(td)
	}
	if gen.needTimestamps {
		gen.CreateTimestamp()
	}
	gen.CreateJsonUtil()
	if gen.Err != nil {
		return gen.Err
	}
	if gen.Server {
		gen.CreateServer()
	}
	if gen.UseMaven {
		gen.CreatePom()
	}
	return gen.Err
}

func defaultDomain() string {
	return os.Getenv("DOMAIN")
}

func reverseStrings(ary []string) []string {
	nary := len(ary)
	rev := make([]string, nary, nary)
	i := nary - 1
	for _, v := range ary {
		rev[i] = v
		i--
	}
	return rev
}

func defaultPackage(domainName, name string) string {
	rev := strings.Join(reverseStrings(strings.Split(domainName, ".")), ".")
	return rev + "." + name
}

func (gen *Generator) AddImport(fullReference string) {
	for _, s := range gen.imports {
		if fullReference == s {
			return
		}
	}
	gen.imports = append(gen.imports, fullReference)
}

func NewGenerator(model *sadl.Model, outdir string, conf map[string]interface{}) *Generator {
	gen := &Generator{}
	if outdir == "" {
		outdir = "."
		panic("yeah")
	}
	domain := gen.GetString(conf, "domain", "-")
	if domain == "-" {
		if model.Namespace != "" {
			domain = strings.Join(reverseStrings(strings.Split(model.Namespace, ".")), ".")
		} else {
			domain = defaultDomain()
		}
	}
	gen.Domain = domain
	name := gen.GetString(conf, "name", model.Name)
	gen.Name = name
	pkg := gen.GetString(conf, "package", "")
	if pkg == "" {
		pkg = defaultPackage(domain, name)
	}
	gen.Package = pkg
	gen.Header = gen.GetString(conf, "header", "//\n// Generated by sadl\n//\n")
	gen.SourceDir = gen.GetString(conf, "source", "src/main/java")
	gen.ResourceDir = gen.GetString(conf, "resource", "src/main/resources")
	gen.Server = gen.GetBool(conf, "server", false)
	gen.UseLombok = gen.GetBool(conf, "lombok", false)
	gen.UseGetters = gen.GetBool(conf, "getters", false)
	gen.UseInstants = gen.GetBool(conf, "instants", false)
	gen.UseMaven = gen.GetBool(conf, "maven", true)
	gen.Model = model
	gen.OutDir = outdir
	srcpath := filepath.Join(outdir, gen.SourceDir)
	pdir := filepath.Join(srcpath, javaPackageToPath(pkg))
	err := os.MkdirAll(pdir, 0755)
	if err != nil {
		gen.Err = err
	}
	return gen
}

func (gen *Generator) WriteJavaFile(name string, content string, pkg string) {
	if gen.Err == nil {
		head := gen.Header
		if pkg != "" {
			head = head + "package " + pkg + ";\n"
		}
		content = head + content
		dir := filepath.Join(gen.OutDir, gen.SourceDir)
		if pkg != "" {
			dir = filepath.Join(dir, javaPackageToPath(pkg))
		}
		path := filepath.Join(dir, name+".java")
		gen.WriteFile(path, content)
	}
}

func (gen *Generator) CreateJavaFileFromTemplate(name string, tmpl string, data interface{}, funcMap template.FuncMap, pkg string) {
	gen.Begin()
	gen.EmitTemplate(name, tmpl, data, funcMap)
	content := gen.End()
	if gen.Err == nil {
		gen.WriteJavaFile(name, content, pkg)
	}
}

func (gen *Generator) CreatePojoFromDef(td *sadl.TypeDef) {
	className := gen.Capitalize(td.Name)
	gen.CreatePojo(&td.TypeSpec, className, td.Comment)
}

func (gen *Generator) CreatePojo(ts *sadl.TypeSpec, className, comment string) {
	if gen.Err != nil {
		return
	}
	gen.Begin()
	if comment != "" {
		gen.Emit(gen.FormatComment("", comment, 100))
	}
	switch ts.Type {
	case "Struct":
		gen.CreateStructPojo(ts, className, "")
	case "UnitValue":
		gen.CreateUnitValuePojo(ts, className)
	case "Enum":
		gen.CreateEnumPojo(ts, className)
	case "Union":
		gen.CreateUnionPojo(ts, className)
	default:
		//do nothing, i.e. a String subclass
		return
	}
	result := gen.End()
	if result != "" {
		if len(gen.imports) > 0 {
			gen.Begin()
			//			gen.Emit(gen.Header)
			//			if gen.Package != "" {
			//				gen.Emit("package " + gen.Package + ";\n\n")
			//			}
			sort.Strings(gen.imports)
			for _, pack := range gen.imports {
				gen.Emit("import " + pack + ";\n")
			}
			gen.Emit("\n")
			prelude := gen.End()
			result = prelude + result
		}
		gen.WriteJavaFile(className, result, gen.Package)
	}
}

func (gen *Generator) CreateStructPojo(td *sadl.TypeSpec, className string, indent string) {
	optional := false
	for _, fd := range td.Fields {
		if !fd.Required {
			optional = true
		}
	}
	if optional {
		gen.AddImport("com.fasterxml.jackson.annotation.JsonInclude")
		//		gen.emit("@JsonInclude(JsonInclude.Include.NON_EMPTY)\n")
	} else {
		gen.AddImport("javax.validation.constraints.NotNull")
	}
	extends := ""
	if indent == "" {
		if gen.UseLombok {
			gen.Emit(indent + "@Data\n")
			gen.AddImport("lombok.Data")
			if len(td.Fields) > 0 {
				gen.Emit(indent + "@AllArgsConstructor\n")
				gen.AddImport("lombok.AllArgsConstructor")
			}
			gen.Emit(indent + "@Builder\n")
			gen.AddImport("lombok.Builder")
			gen.Emit(indent + "@NoArgsConstructor\n")
			gen.AddImport("lombok.NoArgsConstructor")
		}
		gen.Emit(indent + "public class " + className + extends + " {\n")
	} else {
		gen.Emit(indent + "public static class " + className + extends + " {\n")
	}
	nested := make(map[string]*sadl.TypeSpec, 0)
	for _, fd := range td.Fields {
		if fd.Comment != "" {
			gen.Emit(gen.FormatComment(indent+"    ", fd.Comment, 100))
		}
		if !fd.Required {
			gen.Emit(indent + "    @JsonInclude(JsonInclude.Include.NON_EMPTY) /* Optional field */\n")
		}
		tn, tanno, anonymous := gen.TypeName(&fd.TypeSpec, fd.Type, fd.Required)
		if anonymous != nil {
			tn = gen.Capitalize(fd.Name)
			if tn == className {
				gen.Err = fmt.Errorf("Cannot have identically named inner class with same name as containing class: %q", tn)
				return
			}
			nested[tn] = anonymous
		}
		if tanno != nil {
			for _, anno := range tanno {
				gen.Emit(indent + "    " + anno + "\n")
			}
		}
		gen.Emit(indent + "    public " + tn + " " + fd.Name + ";\n\n")
	}
	if !gen.UseLombok {
		for _, fd := range td.Fields {
			gen.EmitFluidSetter(className, td, fd, indent)
		}
		gen.Emit(`    @Override
    public String toString() {
        return Json.pretty(this);
    }
`)
		if len(nested) > 0 {
			for iname, ispec := range nested {
				gen.CreateStructPojo(ispec, iname, indent+"    ")
			}
		}
	}
	gen.Emit("}\n")
}

func (gen *Generator) CreateEnumPojo(ts *sadl.TypeSpec, className string) {
	gen.AddImport("com.fasterxml.jackson.annotation.JsonValue")

	gen.Emit("public enum " + className + "{\n")
	max := len(ts.Elements)
	delim := ","
	for i := 0; i < max; i++ {
		el := ts.Elements[i]
		if i == max-1 {
			delim = ";"
		}
		comment := "\n"
		if el.Comment != "" {
			comment = gen.FormatComment(" ", el.Comment, 0)
		}
		gen.Emit("    " + strings.ToUpper(el.Symbol) + "(\"" + el.Symbol + "\")" + delim + comment)
	}
	gen.Emit("\n")
	gen.Emit("    private String repr;\n\n")
	gen.Emit("    private " + className + "(String repr) {\n        this.repr = repr;\n    }\n\n")

	gen.Emit("    @JsonValue\n    @Override\n")
	gen.Emit("    public String toString() {\n        return repr;\n    }\n\n")

	gen.Emit("    public static " + className + " fromString(String repr) {\n")
	gen.Emit("        for (" + className + " e : values()) {\n")
	gen.Emit("            if (e.repr.equals(repr)) {\n")
	gen.Emit("                return e;\n")
	gen.Emit("            }\n")
	gen.Emit("        }\n")
	gen.Emit("        throw new IllegalArgumentException(\"Invalid string representation for " + className + ": \" + repr);\n")
	gen.Emit("    }\n\n")
	gen.Emit("}\n")
}

func (gen *Generator) CreateUnionPojo(td *sadl.TypeSpec, className string) {
	indent := ""
	//firstFieldName := td.Name + "Variant"
	//other fields names: td.Variants
	//constructors: one for each type. Is the empty constructor needed for Jackson? Probably.
	optional := false
	for _, fd := range td.Fields {
		if !fd.Required {
			optional = true
		}
	}
	if optional {
		gen.AddImport("com.fasterxml.jackson.annotation.JsonInclude")
		//		gen.emit("@JsonInclude(JsonInclude.Include.NON_EMPTY)\n")
	} else {
		gen.AddImport("javax.validation.constraints.NotNull")
	}
	extends := ""
	if indent == "" {
		if gen.UseLombok {
			gen.Emit(indent + "@Data\n")
			gen.AddImport("lombok.Data")
			gen.Emit(indent + "@AllArgsConstructor\n")
			gen.AddImport("lombok.AllArgsConstructor")
			gen.Emit(indent + "@Builder\n")
			gen.AddImport("lombok.Builder")
			gen.Emit(indent + "@NoArgsConstructor\n")
			gen.AddImport("lombok.NoArgsConstructor")
		}
		gen.Emit(indent + "public class " + className + extends + " {\n")
	} else {
		gen.Emit(indent + "public static class " + className + extends + " {\n")
	}
	variantType := className + "Variant"
	nindent := indent + "    "
	gen.Emit(nindent + variantType + " {\n")

	max := len(td.Variants)
	delim := ","
	for i := 0; i < max; i++ {
		v := td.Variants[i]
		if i == max-1 {
			delim = ""
		}
		gen.Emit(nindent + "    " + v + delim + "\n")
	}
	gen.Emit(nindent + "}\n\n")
	gen.Emit(nindent + "@com.fasterxml.jackson.annotation.JsonIgnore\n")
	gen.Emit(nindent + "public " + variantType + " variant;\n\n")
	for _, v := range td.Variants {
		gen.Emit(nindent + "@JsonInclude(JsonInclude.Include.NON_EMPTY) /* Optional field */\n")
		gen.Emit(nindent + "public " + v + " " + v + ";\n")
	}
	//create each constructor. Do I need the empty constructor?
	for _, v := range td.Variants {
		gen.Emit(nindent + "\n" + nindent + "public " + className + "(" + v + " v) {\n")
		gen.Emit(nindent + "    this.variant = " + v + ";\n")
		gen.Emit(nindent + "    this." + v + " = v;\n" + nindent + "}\n")
	}
	gen.Emit("\n")
	gen.Emit(`    @Override
    public String toString() {
        return Json.pretty(this);
    }
`)
	gen.Emit("}\n")
}

func (gen *Generator) CreateUnitValuePojo(ts *sadl.TypeSpec, className string) {
	gen.AddImport("javax.validation.constraints.NotNull")
	gen.AddImport("com.fasterxml.jackson.annotation.JsonValue")
	gen.Emit("public class " + className + " {\n\n")

	valueType, _, _ := gen.TypeName(ts, ts.Value, true) //this type must be  primitive numeric type
	unitType, _, _ := gen.TypeName(ts, ts.Unit, true)
	if gen.Err != nil {
		return
	}
	gen.Emit("    public final " + valueType + " value;\n")
	gen.Emit("    public final " + unitType + " unit;\n\n")

	gen.Emit("    public " + className + "(" + valueType + " value, @NotNull " + unitType + " unit) {\n")
	gen.Emit("        this.value = value;\n")
	gen.Emit("        this.unit = unit;\n")
	gen.Emit("    }\n\n")

	v := "<bad value type, must be numeric>"
	switch valueType {
	case "double":
		v = "Double.parseDouble(tmp[0])"
	case "float":
		v = "Float.parseFloat(tmp[0])"
	case "long":
		v = "Long.parseLong(tmp[0])"
	case "int":
		v = "Integer.parseInt(tmp[0])"
	case "short":
		v = "Short.parseShort(tmp[0])"
	case "byte":
		v = "Byte.parseByte(tmp[0])"
	case "BigDecimal":
		v = "new BigDecimal(tmp[0])"
	}
	u := "<bad unit type, must be enum or string>"
	ut := gen.Model.FindType(unitType)
	switch ut.Type {
	case "String":
		u = "tmp[1]"
	case "Enum":
		u = unitType + ".fromString(tmp[1])"
	}
	gen.Emit("    public " + className + "(@NotNull String repr) {\n")
	gen.Emit("        String[] tmp = repr.split(\" \");\n")
	gen.Emit("        this.value = " + v + ";\n")
	gen.Emit("        this.unit = " + u + ";\n")
	gen.Emit("    }\n\n")

	gen.Emit(`    @JsonValue
    @Override
    public String toString() {
        return value + " " + unit;
    }
}
`)
}

func (gen *Generator) EmitFluidSetter(className string, ts *sadl.TypeSpec, fd *sadl.StructFieldDef, indent string) {
	if gen.Err != nil {
		return
	}
	tn, _, anonymous := gen.TypeName(&fd.TypeSpec, fd.Type, fd.Required)
	//fixme: the annotations are getting ignored. Figure out if this is preferred or not
	if anonymous != nil {
		tn = gen.Capitalize(fd.Name)
	}
	gen.Emit(indent + "    public " + className + " " + fd.Name + "(" + tn + " val) {\n")
	gen.Emit(indent + "        this." + fd.Name + " = val;\n")
	gen.Emit(indent + "        return this;\n")
	gen.Emit(indent + "    }\n\n")
}

func (gen *Generator) TypeName(ts *sadl.TypeSpec, name string, required bool) (string, []string, *sadl.TypeSpec) {
	primitiveName, isPrimitive := primitiveType(name)
	var annotations []string
	if required {
		if isPrimitive {
			return primitiveName, nil, nil
		}
		gen.AddImport("javax.validation.constraints.NotNull")
		annotations = append(annotations, "@NotNull")
	} else {
		if isPrimitive {
			if primitiveName == "int" {
				name = "Integer"
			} else {
				name = gen.Capitalize(primitiveName)
			}
		}
	}
	switch name {
	case "String":
		if ts != nil {
			if ts.Pattern != "" {
				gen.AddImport("javax.validation.constraints.Pattern")
				annotations = append(annotations, fmt.Sprintf("@Pattern(regexp=%q)", ts.Pattern))
			} else if ts.Values != nil {
				//?
			}
			if ts.MinSize != nil || ts.MaxSize != nil {
				gen.AddImport("javax.validation.constraints.Size")
				smin := ""
				if ts.MinSize != nil {
					smin = fmt.Sprintf("min=%d", *ts.MinSize)
				}
				smax := ""
				if ts.MaxSize != nil {
					smax = fmt.Sprintf("max=%d", *ts.MaxSize)
				}
				if smax != "" {
					smax = ", " + smax
				}
				annotations = append(annotations, fmt.Sprintf("@Size(%s%s)", smin, smax))
			}
		}
		return "String", annotations, nil
	case "Byte", "Short", "Integer", "Long", "Float", "Double":
		if ts.Min != nil {
			annotations = append(annotations, fmt.Sprintf("@Min(%s)", ts.Min.String()))
		}
		if ts.Max != nil {
			annotations = append(annotations, fmt.Sprintf("@Max(%s)", ts.Max.String()))
		}
		return name, annotations, nil
	case "Decimal":
		gen.AddImport("java.math.BigDecimal")
		if ts.Min != nil {
			annotations = append(annotations, fmt.Sprintf("@DecimalMin(%q)", ts.Min.String()))
		}
		if ts.Max != nil {
			annotations = append(annotations, fmt.Sprintf("@DecimalMax(%q)", ts.Max.String()))
		}
		return "BigDecimal", annotations, nil
	case "Timestamp":
		gen.needTimestamps = true
		if gen.UseInstants {
			gen.AddImport("java.time.Instant")
			gen.AddImport("com.fasterxml.jackson.databind.annotation.JsonSerialize")
			gen.AddImport("com.fasterxml.jackson.databind.annotation.JsonDeserialize")
			annotations = append(annotations, fmt.Sprintf("@JsonDeserialize(using = Timestamp.InstantDeserializer.class)"))
			annotations = append(annotations, fmt.Sprintf("@JsonSerialize(using = Timestamp.InstantSerializer.class)"))
			return "Instant", annotations, nil
		}
		return "Timestamp", annotations, nil
	case "Array":
		gen.AddImport("java.util.List")
		if ts == nil {
			return "List", annotations, nil
		}
		td := gen.Model.FindType(ts.Items)
		items, _, _ := gen.TypeName(&td.TypeSpec, ts.Items, true)
		return "List<" + items + ">", annotations, nil
	case "Map":
		gen.AddImport("java.util.Map")
		if ts == nil {
			return "Map", annotations, nil
		}
		ktd := gen.Model.FindType(ts.Keys)
		keys, _, _ := gen.TypeName(&ktd.TypeSpec, ts.Keys, true)
		itd := gen.Model.FindType(ts.Items)
		items, _, _ := gen.TypeName(&itd.TypeSpec, ts.Items, true)
		return "Map<" + keys + "," + items + ">", annotations, nil
	case "UUID":
		gen.AddImport("java.util.UUID")
		return name, annotations, nil
	case "Any":
		return "Object", annotations, nil
	default:
		//app-defined type. Parser will have already verified its existence
		td := gen.Model.FindType(name)
		if td != nil {
			switch td.Type {
			case "Int8", "Int16", "Int32", "Int64", "Float32", "Float64":
				return gen.TypeName(&td.TypeSpec, td.Type, required)
			case "String":
				return gen.TypeName(&td.TypeSpec, "String", required)
			case "Array":
				return gen.TypeName(&td.TypeSpec, "Array", false) //FIXME: the "required/optional" state of the field is lost
			case "Map":
				return gen.TypeName(&td.TypeSpec, "Map", false) //FIXME: the "required/optional" state of the field is lost
			case "Struct":
				if name == "Struct" {
					return name, annotations, ts
				}
			case "UUID":
				gen.AddImport("java.util.UUID")
				return "UUID", annotations, nil
			case "Decimal":
				gen.AddImport("java.math.BigDecimal")
				fmt.Println(util.Pretty(td))
				if td.Min != nil {
					annotations = append(annotations, fmt.Sprintf("@DecimalMin(%q)", td.Min.String()))
				}
				if td.Max != nil {
					annotations = append(annotations, fmt.Sprintf("@DecimalMax(%q)", td.Max.String()))
				}
				return "BigDecimal", annotations, nil
			case "Timestamp":
				gen.needTimestamps = true
				if gen.UseInstants {
					gen.AddImport("java.time.Instant")
					gen.AddImport("com.fasterxml.jackson.databind.annotation.JsonSerialize")
					gen.AddImport("com.fasterxml.jackson.databind.annotation.JsonDeserialize")
					annotations = append(annotations, fmt.Sprintf("@JsonDeserialize(using = Timestamp.InstantDeserializer.class)"))
					annotations = append(annotations, fmt.Sprintf("@JsonSerialize(using = Timestamp.InstantSerializer.class)"))
					return "Instant", annotations, nil
				}
				return "Timestamp", annotations, nil
			}
		}
		return name, annotations, nil
	}
}

func javaPackageToPath(pkg string) string {
	return strings.Join(strings.Split(pkg, "."), "/")
}

func primitiveType(name string) (string, bool) {
	switch name {
	case "Bool":
		return "boolean", true
	case "Int8":
		return "byte", true
	case "Int16":
		return "short", true
	case "Int32":
		return "int", true
	case "Int64":
		return "long", true
	case "Float32":
		return "float", true
	case "Float64":
		return "double", true
	default:
		return "", false
	}
}

func (gen *Generator) FormatComment(indent, comment string, maxcol int) string {
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
	pad := tab + emptyPrefix + "\n"
	return pad + buf.String() + pad
}
