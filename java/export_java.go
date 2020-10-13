package java

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/boynton/sadl"
)

type Generator struct {
	sadl.Generator
	Model         *sadl.Model
	Domain        string //the default DNS domain. Used when generating a POM, defaults to getenv("DOMAIN")
	Name          string //the name of the service, if not in the model
	Package       string //the package of the service. Defaults to the reverse domain name
	Header        string //the banner to prepend to every generated file. Defaults to something obvious and simple
	SourceDir     string //the source directory, relative to the project directory. Defaults to "src/main/java"
	ResourceDir   string //the resource directory, relative to the project directory. Defaults to "src/main/resource"
	UseLombok     bool   //use the Lombok library for generated POJOs. The default is to not.
	UseImmutable  bool   //generate immutable POJOs with a builder inner class
	UseGetters    bool   //generate getters and setters for POJOs. By default, a fluid-style setter and public members are used
	UseInstants   bool   //use java.time.Instant for Timestamp implementation. By default, a Timestamp class is generated
	UseJsonPretty bool   //generate a toString() method that pretty prints JSON.
	UseMaven      bool   //use Maven defaults, and generate a pom.xml file for the project to immedaitely build it.
	Server        bool   //generate server code, including a default (but empty) implementation of the service interface.
	needTimestamp bool
	needJson      bool
	imports       []string
	serverData    *ServerData
}

func Export(model *sadl.Model, dir string, conf *sadl.Data) error {
	gen := NewGenerator(model, dir, conf)
	for _, td := range model.Types {
		gen.CreatePojoFromDef(td)
	}
	if gen.needTimestamp {
		gen.CreateTimestamp()
	} else {
		gen.CreateInstantJson()
	}
	if gen.needJson {
		gen.CreateJsonUtil()
	}
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

func NewGenerator(model *sadl.Model, outdir string, config *sadl.Data) *Generator {
	gen := &Generator{}
	gen.Config = config
	domain := gen.Config.GetString("domain", "-")
	if domain == "" {
		if model.Namespace != "" {
			domain = strings.Join(reverseStrings(strings.Split(model.Namespace, ".")), ".")
		} else {
			domain = defaultDomain()
		}
	}
	gen.Domain = domain
	name := gen.GetConfigString("name", model.Name)
	gen.Name = name
	pkg := gen.GetConfigString("package", "")
	if pkg == "" {
		pkg = defaultPackage(domain, name)
	}
	gen.Package = pkg
	gen.Header = gen.GetConfigString("header", "//\n// Generated by sadl\n//\n")
	gen.SourceDir = gen.GetConfigString("source", "src/main/java")
	gen.ResourceDir = gen.GetConfigString("resource", "src/main/resources")
	gen.Server = gen.GetConfigBool("server", false)
	gen.UseLombok = gen.GetConfigBool("lombok", false)
	gen.UseGetters = gen.GetConfigBool("getters", false)
	gen.UseImmutable = gen.GetConfigBool("immutable", true)
	gen.UseInstants = gen.GetConfigBool("instants", true)
	gen.UseMaven = gen.GetConfigBool("maven", true)
	gen.UseJsonPretty = gen.GetConfigBool("json", true)
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
		gen.Emit(gen.FormatComment("", comment, 100, false))
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

func (gen *Generator) CreateStructPojo(ts *sadl.TypeSpec, className string, indent string) {
	optional := false
	for _, fd := range ts.Fields {
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
	if gen.UseImmutable {
		gen.AddImport("com.fasterxml.jackson.databind.annotation.JsonDeserialize")
		gen.AddImport("com.fasterxml.jackson.databind.annotation.JsonPOJOBuilder")
		gen.Emit(indent + "@JsonDeserialize(builder = " + className + "." + className + "Builder.class)\n")
	}
	if indent == "" {
		if gen.UseLombok {
			gen.Emit(indent + "@Data\n")
			gen.AddImport("lombok.Data")
			if len(ts.Fields) > 0 {
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
	for _, fd := range ts.Fields {
		if fd.Comment != "" {
			gen.Emit(gen.FormatComment(indent+"    ", fd.Comment, 100, false))
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
		if gen.UseImmutable {
			gen.Emit(indent + "    private final " + tn + " " + fd.Name + ";\n\n")
		} else {
			gen.Emit(indent + "    public " + tn + " " + fd.Name + ";\n\n")
		}
	}
	if !gen.UseLombok {
		if gen.UseImmutable {
			gen.EmitAllFieldsConstructor(className, ts, indent)
			for _, fd := range ts.Fields {
				gen.EmitGetter(className, ts, fd, indent)
			}
			gen.EmitBuilder(className, ts, indent+"    ")
		} else {
			for _, fd := range ts.Fields {
				gen.EmitFluidSetter(className, ts, fd, indent)
			}
		}
		if gen.UseJsonPretty {
			gen.needJson = true
			gen.Emit(`    @Override
    public String toString() {
        return Json.pretty(this);
    }
`)
		}
		if len(nested) > 0 {
			for iname, ispec := range nested {
				gen.CreateStructPojo(ispec, iname, indent+"    ")
			}
		}
	}
	gen.Emit("}\n")
}

func (gen *Generator) EmitAllFieldsConstructor(className string, ts *sadl.TypeSpec, indent string) {
	var args []string
	for _, fd := range ts.Fields {
		tn, _, _ := gen.TypeName(&fd.TypeSpec, fd.Type, fd.Required)
		args = append(args, tn+" "+fd.Name)
	}
	gen.Emit(indent + "    public " + className + "(" + strings.Join(args, ", ") + ") {\n")
	for _, fd := range ts.Fields {
		gen.Emit(indent + "        this." + fd.Name + " = " + fd.Name + ";\n")
	}
	gen.Emit(indent + "    }\n\n")
}

func (gen *Generator) EmitGetter(className string, ts *sadl.TypeSpec, fd *sadl.StructFieldDef, indent string) {
	tn, _, _ := gen.TypeName(&fd.TypeSpec, fd.Type, fd.Required)
	gen.Emit(indent + "    public " + tn + " get" + gen.Capitalize(fd.Name) + "() {\n")
	gen.Emit(indent + "        return " + fd.Name + ";\n")
	gen.Emit(indent + "    }\n\n")
}

func (gen *Generator) EmitBuilder(className string, ts *sadl.TypeSpec, indent string) {
	builderClass := className + "Builder"
	gen.Emit(indent + "public static " + builderClass + " builder() {\n")
	gen.Emit(indent + "    return new " + builderClass + "();\n")
	gen.Emit(indent + "}\n\n")
	gen.Emit(indent + "@JsonPOJOBuilder(withPrefix=\"\")\n")
	gen.Emit(indent + "public static class " + builderClass + " {\n")
	for _, fd := range ts.Fields {
		tn, _, _ := gen.TypeName(&fd.TypeSpec, fd.Type, fd.Required)
		gen.Emit(indent + "    private " + tn + " " + fd.Name + ";\n")
	}
	gen.Emit("\n")
	var args []string
	for _, fd := range ts.Fields {
		tn, _, _ := gen.TypeName(&fd.TypeSpec, fd.Type, fd.Required)
		gen.Emit(indent + "    public " + builderClass + " " + fd.Name + "(" + tn + " " + fd.Name + ") {\n")
		gen.Emit(indent + "        this." + fd.Name + " = " + fd.Name + ";\n")
		gen.Emit(indent + "        return this;\n")
		gen.Emit(indent + "    }\n\n")
		args = append(args, fd.Name)
	}
	gen.Emit(indent + "    public " + className + " build() {\n")
	gen.Emit(indent + "        return new " + className + "(" + strings.Join(args, ", ") + ");\n")
	gen.Emit(indent + "    }\n")
	gen.Emit(indent + "}\n")
}

func (gen *Generator) CreateEnumPojo(ts *sadl.TypeSpec, className string) {
	gen.AddImport("com.fasterxml.jackson.annotation.JsonValue")
	gen.AddImport("com.fasterxml.jackson.annotation.JsonCreator")

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
			comment = gen.FormatComment(" ", el.Comment, 0, false)
		}
		gen.Emit("    " + strings.ToUpper(el.Symbol) + "(\"" + el.Symbol + "\")" + delim + comment)
	}
	gen.Emit("\n")
	gen.Emit("    private String repr;\n\n")
	gen.Emit("    private " + className + "(String repr) {\n        this.repr = repr;\n    }\n\n")

	gen.Emit("    @JsonValue\n    @Override\n")
	gen.Emit("    public String toString() {\n        return repr;\n    }\n\n")

	gen.Emit("    @JsonCreator\n") //not strictly necessary for enums
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
	indent0 := ""
	indent1 := indent0 + "    "
	indent2 := indent1 + "    "
	indent3 := indent2 + "    "
	gen.AddImport("com.fasterxml.jackson.annotation.JsonInclude")
	gen.AddImport("com.fasterxml.jackson.annotation.JsonIgnore")
	gen.AddImport("com.fasterxml.jackson.annotation.JsonCreator")
	gen.AddImport("com.fasterxml.jackson.annotation.JsonProperty")
	extends := ""
	if indent0 == "" {
		if gen.UseLombok {
			gen.Emit("@Data\n")
			gen.AddImport("lombok.Data")
			gen.Emit("@AllArgsConstructor\n")
			gen.AddImport("lombok.AllArgsConstructor")
			gen.Emit("@Builder\n")
			gen.AddImport("lombok.Builder")
			gen.Emit("@NoArgsConstructor\n")
			gen.AddImport("lombok.NoArgsConstructor")
		}
		gen.Emit("public class " + className + extends + " {\n")
	} else {
		gen.Emit(indent0 + "public static class " + className + extends + " {\n")
	}

	variantType := "Variant"
	gen.Emit(indent1 + "public enum " + variantType + " {\n")

	max := len(td.Variants)
	delim := ","
	for i := 0; i < max; i++ {
		vd := td.Variants[i]
		if i == max-1 {
			delim = ""
		}
		gen.Emit(indent2 + vd.Name + delim + "\n")
	}
	gen.Emit(indent1 + "}\n\n")
	gen.Emit(indent1 + "@JsonIgnore\n")
	gen.Emit(indent1 + "public " + variantType + " variant;\n\n")
	nested := make(map[string]*sadl.TypeSpec, 0)
	for _, vd := range td.Variants {
		gen.Emit(indent1 + "@JsonInclude(JsonInclude.Include.NON_EMPTY)\n")

		if vd.Comment != "" {
			gen.Emit(gen.FormatComment(indent1, vd.Comment, 100, false))
		}
		tn, tanno, anonymous := gen.TypeName(&vd.TypeSpec, vd.Type, false)
		if anonymous != nil {
			tn = gen.Capitalize(vd.Name)
			if tn == className {
				gen.Err = fmt.Errorf("Cannot have identically named inner class with same name as containing class: %q", tn)
				return
			}
			nested[tn] = anonymous
		}
		if tanno != nil {
			for _, anno := range tanno {
				gen.Emit(indent1 + anno + "\n")
			}
		}
		//todo: if useGetters, make this private and generate the getter
		gen.Emit(indent1 + "public final " + tn + " " + vd.Name + ";\n")
	}
	gen.Emit("\n" + indent1 + "@JsonCreator\n")
	gen.Emit(indent1 + "private " + className + "(")
	delim = ", "
	for i := 0; i < max; i++ {
		vd := td.Variants[i]
		if i == max-1 {
			delim = ""
		}
		tn, _, _ := gen.TypeName(&vd.TypeSpec, vd.Type, false)
		gen.Emit("@JsonProperty(\"" + vd.Name + "\") " + tn + " " + vd.Name + delim)
	}
	gen.Emit(") {\n")
	for _, vd := range td.Variants {
		gen.Emit(indent2 + "this." + vd.Name + " = " + vd.Name + ";\n")
		gen.Emit(indent2 + "if (" + vd.Name + " != null) {\n")
		gen.Emit(indent3 + "this.variant = " + variantType + "." + vd.Name + ";\n")
		gen.Emit(indent2 + "}\n")
	}
	gen.Emit(indent1 + "}\n\n")

	for _, vd := range td.Variants {
		tn, _, _ := gen.TypeName(&vd.TypeSpec, vd.Type, false)
		gen.Emit("\n" + indent1 + "public static " + className + " of" + gen.Capitalize(vd.Name) + "(" + tn + " v) {\n")
		gen.Emit(indent2 + "return new " + className + "(")
		delim = ", "
		for i, vd2 := range td.Variants {
			if i == max-1 {
				delim = ""
			}
			if vd.Name == vd2.Name {
				gen.Emit("v" + delim)
			} else {
				gen.Emit("null" + delim)
			}
		}
		gen.Emit(");\n")
		gen.Emit(indent1 + "}\n")
	}
	gen.Emit("\n")
	if gen.UseJsonPretty {
		gen.needJson = true
		gen.Emit(`    @Override
    public String toString() {
        return Json.pretty(this);
    }
`)
	}
	if len(nested) > 0 {
		for iname, ispec := range nested {
			gen.CreateStructPojo(ispec, iname, indent1)
		}
	}
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
	gen.Emit("    @JsonCreator\n")
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
		if ts != nil {
			if ts.Min != nil {
				annotations = append(annotations, fmt.Sprintf("@Min(%s)", ts.Min.String()))
			}
			if ts.Max != nil {
				annotations = append(annotations, fmt.Sprintf("@Max(%s)", ts.Max.String()))
			}
		}
		return name, annotations, nil
	case "Decimal":
		gen.AddImport("java.math.BigDecimal")
		if ts != nil {
			if ts.Min != nil {
				annotations = append(annotations, fmt.Sprintf("@DecimalMin(%q)", ts.Min.String()))
			}
			if ts.Max != nil {
				annotations = append(annotations, fmt.Sprintf("@DecimalMax(%q)", ts.Max.String()))
			}
		}
		return "BigDecimal", annotations, nil
	case "Timestamp":
		if gen.UseInstants {
			gen.AddImport("java.time.Instant")
			gen.AddImport("com.fasterxml.jackson.databind.annotation.JsonSerialize")
			gen.AddImport("com.fasterxml.jackson.databind.annotation.JsonDeserialize")
			annotations = append(annotations, fmt.Sprintf("@JsonDeserialize(using = InstantJson.Deserializer.class)"))
			annotations = append(annotations, fmt.Sprintf("@JsonSerialize(using = InstantJson.Serializer.class)"))
			return "Instant", annotations, nil
		}
		gen.needTimestamp = true
		return "Timestamp", annotations, nil
	case "Array":
		gen.AddImport("java.util.List")
		if ts == nil {
			return "List", annotations, nil
		}
		td := gen.Model.FindType(ts.Items)
		items, _, _ := gen.TypeName(&td.TypeSpec, ts.Items, false)
		return "List<" + items + ">", annotations, nil
	case "Map":
		gen.AddImport("java.util.Map")
		if ts == nil {
			return "Map", annotations, nil
		}
		ktd := gen.Model.FindType(ts.Keys)
		keys, _, _ := gen.TypeName(&ktd.TypeSpec, ts.Keys, false)
		itd := gen.Model.FindType(ts.Items)
		items, _, _ := gen.TypeName(&itd.TypeSpec, ts.Items, false)
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
			case "Int8", "Int16", "Int32", "Int64", "Float32", "Float64", "Boolean":
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
				if td.Min != nil {
					annotations = append(annotations, fmt.Sprintf("@DecimalMin(%q)", td.Min.String()))
				}
				if td.Max != nil {
					annotations = append(annotations, fmt.Sprintf("@DecimalMax(%q)", td.Max.String()))
				}
				return "BigDecimal", annotations, nil
			case "Timestamp":
				if gen.UseInstants {
					gen.AddImport("java.time.Instant")
					gen.AddImport("com.fasterxml.jackson.databind.annotation.JsonSerialize")
					gen.AddImport("com.fasterxml.jackson.databind.annotation.JsonDeserialize")
					annotations = append(annotations, fmt.Sprintf("@JsonDeserialize(using = InstantJson.InstantDeserializer.class)"))
					annotations = append(annotations, fmt.Sprintf("@JsonSerialize(using = InstantJson.InstantSerializer.class)"))
					return "Instant", annotations, nil
				}
				gen.needTimestamp = true
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
