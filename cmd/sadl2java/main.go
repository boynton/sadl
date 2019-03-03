package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/extensions/graphql"
	"github.com/boynton/sadl/parse"
)

func main() {
	pDir := flag.String("dir", ".", "output directory for generated artifacts")
	pSrc := flag.String("src", "src/main/java", "output directory for generated source tree")
	pRez := flag.String("rez", "src/main/resources", "output directory for generated resources")
	pPackage := flag.String("package", "", "Java package for generated source")
	pServer := flag.Bool("server", false, "generate server code")
	pGraphql := flag.Bool("graphql", false, "generate graphql endpoint that resolves to http operations")
	pLombok := flag.Bool("lombok", false, "generate Lombok annotations")
	pGetters := flag.Bool("getters", false, "generate setters/getters instead of the default fluent style")
	pInstant := flag.Bool("instant", false, "Use java.time.Instant. By default, use generated Timestamp class")
	pJsonutil := flag.Bool("jsonutil", false, "Create Json.java utility class")
	pPom := flag.Bool("pom", false, "Create Maven pom.xml file to build the project")
	flag.Parse()
	argv := flag.Args()
	argc := len(argv)
	if argc == 0 {
		fmt.Fprintf(os.Stderr, "usage: sadl2java -dir projdir -src relative_source_dir -rez relative_resource_dir -package java.package.name -pom -server -jsonutil -getters -lombok some_model.sadl\n")
		os.Exit(1)
	}
	var model *sadl.Model
	var err error
	if *pGraphql {
		model, err = parse.File(argv[0], graphql.NewExtension())
	} else {
		model, err = parse.File(argv[0])
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "*** %v\n", err)
		os.Exit(1)
	}
	err = generatePojos(model, *pDir, *pSrc, *pPackage, *pLombok, *pGetters, *pJsonutil, *pInstant)
	if err != nil {
		fmt.Fprintf(os.Stderr, "*** %v\n", err)
		os.Exit(1)
	}
	if *pServer {
		err = createServer(model, *pPackage, *pDir, *pSrc, *pRez, *pGraphql)
		if err != nil {
			fmt.Fprintf(os.Stderr, "*** %v\n", err)
		}
	}
	if *pPom {
		domain := os.Getenv("DOMAIN")
		if domain == "" {
			domain = "my.domain"
		}
		err = createPom(domain, model.Name, *pDir, *pLombok, *pGraphql)
		if err != nil {
			fmt.Fprintf(os.Stderr, "*** %v\n", err)
		}
	}
}

func fileExists(filepath string) bool {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return false
	}
	return true
}

func generatePojos(model *sadl.Model, dir, src, pkg string, lombok, getters, jsonutil, instant bool) error {
	gen := newPojoGenerator(model, dir, src, pkg, lombok, getters, jsonutil, instant)
	if gen.jsonutil {
		gen.createJsonUtil()
	}
	for _, td := range model.Types {
		gen.createPojo(td)
	}
	if gen.timestamps {
		gen.createTimestamp()
	}
	return gen.err
}

func newPojoGenerator(model *sadl.Model, dir, src, pkg string, lombok, getters, jsonutil, instant bool) *PojoGenerator {
	gen := &PojoGenerator{
		model:    model,
		dir:      dir,
		src:      src,
		pkgname:  pkg,
		lombok:   lombok,
		getters:  getters,
		jsonutil: jsonutil,
		instant:  instant,
		header:   "//\n// Generated by sadl2java\n//\n",
	}
	srcpath := filepath.Join(gen.dir, gen.src)
	gen.pkgpath = filepath.Join(srcpath, javaPackageToPath(gen.pkgname))
	if gen.pkgpath != "" {
		err := os.MkdirAll(gen.pkgpath, 0755)
		if err != nil {
			gen.err = err
		}
	}
	return gen
}

type PojoGenerator struct {
	model      *sadl.Model
	dir        string
	src        string
	pkgname    string
	pkgpath    string
	imports    []string
	pom        bool
	lombok     bool
	getters    bool
	jsonutil   bool
	instant    bool
	timestamps bool //something in the model references the Timestamp type
	header     string
	file       *os.File
	writer     *bufio.Writer
	err        error
}

func (gen *PojoGenerator) createJavaFile(name string) {
	if gen.err != nil {
		return
	}
	path := filepath.Join(gen.pkgpath, name+".java")
	if fileExists(path) {
		fmt.Printf("[%s already exists, not overwriting]\n", path)
		return
	}
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

func (gen *PojoGenerator) createPojo(td *sadl.TypeDef) {
	if gen.err != nil {
		return
	}
	var b bytes.Buffer
	gen.writer = bufio.NewWriter(&b) //first write to a string
	if td.Comment != "" {
		gen.emit(gen.formatComment("", td.Comment, 100))
	}
	className := capitalize(td.Name)
	switch td.Type {
	case "Struct":
		gen.createStructPojo(&td.TypeSpec, className, "")
	case "Quantity":
		gen.createQuantityPojo(td, className)
	case "Enum":
		gen.createEnumPojo(td, className)
	default:
		//do nothing, i.e. a String subclass
		return
	}
	if gen.err == nil {
		gen.writer.Flush()
		gen.createJavaFile(td.Name) //then create file and write the header with imports
		if len(gen.imports) > 0 {
			sort.Strings(gen.imports)
			for _, pack := range gen.imports {
				gen.emit("import " + pack + ";\n")
			}
			gen.emit("\n")
		}
		b.WriteTo(gen.writer) //and append the originally written output after that
		gen.writer.Flush()
		gen.file.Close()
	}
}

func (gen *PojoGenerator) createQuantityPojo(td *sadl.TypeDef, className string) {
	gen.addImport("javax.validation.constraints.NotNull")
	gen.addImport("com.fasterxml.jackson.annotation.JsonValue")
	gen.emit("public class " + className + " {\n\n")

	valueType, _, _ := gen.typeName(&td.TypeSpec, td.Value, true) //this type must be  primitive numeric type
	unitType, _, _ := gen.typeName(&td.TypeSpec, td.Unit, true)
	if gen.err != nil {
		return
	}
	gen.emit("    public final " + valueType + " value;\n")
	gen.emit("    public final " + unitType + " unit;\n\n")

	gen.emit("    public " + className + "(" + valueType + " value, @NotNull " + unitType + " unit) {\n")
	gen.emit("        this.value = value;\n")
	gen.emit("        this.unit = unit;\n")
	gen.emit("    }\n\n")

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
	ut := gen.model.FindType(unitType)
	switch ut.Type {
	case "String":
		u = "tmp[1]"
	case "Enum":
		u = unitType + ".fromString(tmp[1])"
	}
	gen.emit("    public " + className + "(@NotNull String repr) {\n")
	gen.emit("        String[] tmp = repr.split(\" \");\n")
	gen.emit("        this.value = " + v + ";\n")
	gen.emit("        this.unit = " + u + ";\n")
	gen.emit("    }\n\n")

	gen.emit(`    @JsonValue
    @Override
    public String toString() {
        return value + " " + unit;
    }
}
`)
}

func (gen *PojoGenerator) createEnumPojo(td *sadl.TypeDef, className string) {
	gen.addImport("com.fasterxml.jackson.annotation.JsonValue")

	gen.emit("public enum " + className + "{\n")
	max := len(td.Elements)
	delim := ","
	for i := 0; i < max; i++ {
		el := td.Elements[i]
		if i == max-1 {
			delim = ";"
		}
		comment := "\n"
		if el.Comment != "" {
			comment = gen.formatComment(" ", el.Comment, 0)
		}
		gen.emit("    " + strings.ToUpper(el.Symbol) + "(\"" + el.Symbol + "\")" + delim + comment)
	}
	gen.emit("\n")
	gen.emit("    private String repr;\n\n")
	gen.emit("    private " + className + "(String repr) {\n        this.repr = repr;\n    }\n\n")

	gen.emit("    @JsonValue\n    @Override\n")
	gen.emit("    public String toString() {\n        return repr;\n    }\n\n")

	gen.emit("    public static " + className + " fromString(String repr) {\n")
	gen.emit("        for (" + className + " e : values()) {\n")
	gen.emit("            if (e.repr.equals(repr)) {\n")
	gen.emit("                return e;\n")
	gen.emit("            }\n")
	gen.emit("        }\n")
	gen.emit("        throw new IllegalArgumentException(\"Invalid string representation for " + className + ": \" + repr);\n")
	gen.emit("    }\n\n")
	gen.emit("}\n")
}

func (gen *PojoGenerator) createStructPojo(td *sadl.TypeSpec, className string, indent string) {
	optional := false
	for _, fd := range td.Fields {
		if !fd.Required {
			optional = true
		}
	}

	if optional {
		gen.addImport("com.fasterxml.jackson.annotation.JsonInclude")
		//		gen.emit("@JsonInclude(JsonInclude.Include.NON_EMPTY)\n")
	} else {
		gen.addImport("javax.validation.constraints.NotNull")
	}
	extends := ""
	if indent == "" {
		if gen.lombok {
			gen.emit(indent + "@Data\n")
			gen.addImport("lombok.Data")
			gen.emit(indent + "@AllArgsConstructor\n")
			gen.addImport("lombok.AllArgsConstructor")
			gen.emit(indent + "@Builder\n")
			gen.addImport("lombok.Builder")
			gen.emit(indent + "@NoArgsConstructor\n")
			gen.addImport("lombok.NoArgsConstructor")
		}
		gen.emit(indent + "public class " + className + extends + " {\n")
	} else {
		gen.emit(indent + "public static class " + className + extends + " {\n")
	}
	nested := make(map[string]*sadl.TypeSpec, 0)
	for _, fd := range td.Fields {
		if fd.Comment != "" {
			gen.emit(gen.formatComment(indent+"    ", fd.Comment, 100))
		}
		if !fd.Required {
			gen.emit(indent + "    @JsonInclude(JsonInclude.Include.NON_EMPTY) /* Optional field */\n")
		}
		tn, tanno, anonymous := gen.typeName(&fd.TypeSpec, fd.Type, fd.Required)
		if anonymous != nil {
			tn = capitalize(fd.Name)
			if tn == className {
				gen.err = fmt.Errorf("Cannot have identically named inner class with same name as containing class: %q", tn)
				return
			}
			nested[tn] = anonymous
		}
		if tanno != nil {
			for _, anno := range tanno {
				gen.emit(indent + "    " + anno + "\n")
			}
		}
		gen.emit(indent + "    public " + tn + " " + fd.Name + ";\n\n")
	}
	if !gen.lombok {
		for _, fd := range td.Fields {
			gen.emitFluidSetter(className, td, fd, indent)
		}
		if gen.jsonutil {
			gen.emit(`    @Override
    public String toString() {
        return Json.pretty(this);
    }
`)
		}
		if len(nested) > 0 {
			for iname, ispec := range nested {
				gen.createStructPojo(ispec, iname, indent+"    ")
			}
		}
	}
	gen.emit("}\n")
}

func (gen *PojoGenerator) emitFluidSetter(className string, ts *sadl.TypeSpec, fd *sadl.StructFieldDef, indent string) {
	if gen.err != nil {
		return
	}
	tn, _, anonymous := gen.typeName(&fd.TypeSpec, fd.Type, fd.Required)
	//fixme: the annotations are getting ignored. Figure out if this is preferred or not
	if anonymous != nil {
		tn = capitalize(fd.Name)
	}
	gen.emit(indent + "    public " + className + " " + fd.Name + "(" + tn + " val) {\n")
	gen.emit(indent + "        this." + fd.Name + " = val;\n")
	gen.emit(indent + "        return this;\n")
	gen.emit(indent + "    }\n\n")
}

func adjoin(lst []string, val string) []string {
	for _, s := range lst {
		if val == s {
			return lst
		}
	}
	return append(lst, val)
}

func (gen *PojoGenerator) addImport(fullReference string) {
	gen.imports = adjoin(gen.imports, fullReference)
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

func (gen *PojoGenerator) typeName(ts *sadl.TypeSpec, name string, required bool) (string, []string, *sadl.TypeSpec) {
	primitiveName, isPrimitive := primitiveType(name)
	var annotations []string
	if required {
		if isPrimitive {
			return primitiveName, nil, nil
		}
		gen.addImport("javax.validation.constraints.NotNull")
		annotations = append(annotations, "@NotNull")
	} else {
		if isPrimitive {
			if primitiveName == "int" {
				name = "Integer"
			} else {
				name = capitalize(primitiveName)
			}
		}
	}
	switch name {
	case "String":
		if ts != nil {
			if ts.Pattern != "" {
				gen.addImport("javax.validation.constraints.Pattern")
				annotations = append(annotations, fmt.Sprintf("@Pattern(regexp=%q)", ts.Pattern))
			} else if ts.Values != nil {
				//?
			}
			if ts.MinSize != nil || ts.MaxSize != nil {
				gen.addImport("javax.validation.constraints.Size")
				smin := ""
				if ts.MinSize != nil {
					smin = fmt.Sprintf("min=%d", *ts.MinSize)
				}
				smax := ""
				if ts.MaxSize != nil {
					smax = fmt.Sprintf("max=%d", *ts.MaxSize)
				}
				if smin != "" {
					smax = ", " + smax
				}
				annotations = append(annotations, fmt.Sprintf("@Size(%s%s)", smin, smax))
			}
		}
		return "String", annotations, nil
	case "Decimal":
		gen.addImport("java.math.BigDecimal")
		return "BigDecimal", annotations, nil
	case "Timestamp":
		gen.timestamps = true
		if gen.instant {
			gen.addImport("java.time.Instant")
			gen.addImport("com.fasterxml.jackson.databind.annotation.JsonSerialize")
			gen.addImport("com.fasterxml.jackson.databind.annotation.JsonDeserialize")
			annotations = append(annotations, fmt.Sprintf("@JsonDeserialize(using = Timestamp.InstantDeserializer.class)"))
			annotations = append(annotations, fmt.Sprintf("@JsonSerialize(using = Timestamp.InstantSerializer.class)"))
			return "Instant", annotations, nil
		}
		return "Timestamp", annotations, nil
	case "Array":
		gen.addImport("java.util.List")
		if ts == nil {
			return "List", annotations, nil
		}
		td := gen.model.FindType(ts.Items)
		items, _, _ := gen.typeName(&td.TypeSpec, ts.Items, true)
		return "List<" + items + ">", annotations, nil
	case "Map":
		gen.addImport("java.util.Map")
		if ts == nil {
			return "Map", annotations, nil
		}
		ktd := gen.model.FindType(ts.Keys)
		keys, _, _ := gen.typeName(&ktd.TypeSpec, ts.Keys, true)
		itd := gen.model.FindType(ts.Items)
		items, _, _ := gen.typeName(&itd.TypeSpec, ts.Items, true)
		return "Map<" + keys + "," + items + ">", annotations, nil
	case "UUID":
		gen.addImport("java.util.UUID")
		return name, annotations, nil
	case "Any":
		return "Object", annotations, nil
	default:
		//app-defined type. Parser will have already verified its existence
		td := gen.model.FindType(name)
		if td != nil {
			switch td.Type {
			case "String":
				return gen.typeName(&td.TypeSpec, "String", required)
			case "Array":
				return gen.typeName(&td.TypeSpec, "Array", false) //FIXME: the "required/optional" state of the field is lost
			case "Map":
				return gen.typeName(&td.TypeSpec, "Map", false) //FIXME: the "required/optional" state of the field is lost
			case "Struct":
				if name == "Struct" {
					return name, annotations, ts
				}
			case "UUID":
				gen.addImport("java.util.UUID")
				return "UUID", annotations, nil
			case "Timestamp":
				gen.timestamps = true
				if gen.instant {
					gen.addImport("java.time.Instant")
					gen.addImport("com.fasterxml.jackson.databind.annotation.JsonSerialize")
					gen.addImport("com.fasterxml.jackson.databind.annotation.JsonDeserialize")
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

func (gen *PojoGenerator) emit(s string) {
	if gen.err == nil {
		_, err := gen.writer.WriteString(s)
		if err != nil {
			gen.err = err
		}
	}
}

func javaPackageToPath(pkg string) string {
	return strings.Join(strings.Split(pkg, "."), "/")
}

func capitalize(s string) string {
	return strings.ToUpper(s[0:1]) + s[1:]
}

func lowercase(s string) string {
	return strings.ToLower(s)
}

func uncapitalize(s string) string {
	return strings.ToLower(s[0:1]) + s[1:]
}

func (gen *PojoGenerator) createJsonUtil() {
	gen.createJavaFile("Json")
	if gen.err != nil {
		return
	}
	gen.emit(javaJsonUtil)
	gen.writer.Flush()
	gen.file.Close()
}

var javaJsonUtil = `
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.fasterxml.jackson.databind.DeserializationFeature;

public class Json {

    static final ObjectMapper om = initMapper();
    static ObjectMapper initMapper() {
        ObjectMapper om = new ObjectMapper();
        om.disable(SerializationFeature.WRITE_NULL_MAP_VALUES);
        om.configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
        return om;
    }

    public static <T> T parse(String jsonData, Class<T> dataType) {
        try {
            return om.readerFor(dataType).readValue(jsonData);
        } catch (Exception e) {
            e.printStackTrace();
            return null;
        }
    }

    public static String string(Object o) {
        try {
            Class<?> cls = (o == null)? Object.class : o.getClass();
            return om.writerWithView(cls).writeValueAsString(o);
        } catch (Exception e) {
            e.printStackTrace();
            return "?";
        }
    }

    public static String pretty(Object o) {
        try {
            Class<?> cls = (o == null)? Object.class : o.getClass();
            return om.writerWithView(cls).with(SerializationFeature.INDENT_OUTPUT).writeValueAsString(o);
        } catch (Exception e) {
            e.printStackTrace();
            return "?";
        }
    }

    public static <T> String[] validate(T t) {
        return new String[0]; //replace with a real validator
    }
}
`

func (gen *PojoGenerator) formatComment(indent, comment string, maxcol int) string {
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