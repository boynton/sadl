package main

import(
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/boynton/sadl"
)

type serverData struct {
	Model *sadl.Model
	Name string
	Package string
	PackageLine string
	Port int
	MainClass string
	ImplClass string
	InterfaceClass string
	ResourcesClass string
	RootPath string
	Op *sadl.HttpDef
	Inputs []*sadl.HttpParamSpec
	Expected *sadl.HttpExpectedSpec
	Errors []*sadl.HttpExceptionSpec
	Class string
	Imports []string
}

func opInfo(op *sadl.HttpDef) (string, string) {
	switch op.Method {
	case "POST", "PUT":
		for _, in := range op.Inputs {
			if in.Query == "" && in.Header == "" && !in.Path {
				return in.Name, in.Type
			}
		}
	default:
		for _, out := range op.Expected.Outputs {
			if out.Header == "" {
				return out.Name, out.Type
			}
		}
	}
	return "anonymous", "Object"
}

func createServer(model *sadl.Model, pkg, dir, src string) error {
	serviceName := capitalize(model.Name)
	rootPath := model.Base
	if rootPath == "" {
		rootPath = "/"
	}
	data := &serverData{
		RootPath: rootPath,
		Model: model,
		Name: model.Name,
		Port: 8080,
		MainClass: "Main",
		ImplClass: serviceName + "Impl",
		InterfaceClass: serviceName + "Service",
		ResourcesClass: serviceName + "Resources",
	}
	lombok := false //FIXME
	jsonutil := false //FIXME
	instant := false //FIXME
	getters := false //FIXME
	gen := newPojoGenerator(model, dir, src, pkg, lombok, getters, jsonutil, instant)

	opName := func(op *sadl.HttpDef) string {
		return operationName(op)
	}
	entityNameType := func(op *sadl.HttpDef) (string, string) {
//		out := op.Expected
		for _, out := range op.Expected.Outputs {
			if out.Header == "" {
				tn, _, _ := gen.typeName(nil, out.Type, true)
				return out.Name, tn
			}
		}
/*		if out.Type != "" {
			td := gen.model.FindType(out.Type)
			if td == nil {
				//we must generate it.
				td = &sadl.TypeDef{
					Name: out.Type,
					TypeSpec: sadl.TypeSpec{
						Type: "Struct",
					},
				}
			}
			tn, _, _ := gen.typeName(&td.TypeSpec, out.Type, true)
			return "FIXME", tn
		}
*/
		return "", "void"
	}
	reqType := func(name string) string {
		return requestType(name)
	}
	funcMap := template.FuncMap{
		"openBrace": func() string { return "{" },
		"methodPath":  func(op *sadl.HttpDef) string {
			path := op.Path
			i := strings.Index(path, "?")
			if i >= 0 {
				path = path[0:i]
			}
			return path
		},
		"outtype": func (op *sadl.HttpDef) string {
			_, t := opInfo(op)
			return t
		},
		"outname": func (op *sadl.HttpDef) string {
			n, _ := opInfo(op)
			return n
		},
		"reqClass":  func(op *sadl.HttpDef) string { return reqType(opName(op))},
		"resClass":  func(op *sadl.HttpDef) string {
			resType, _ := gen.responsePojoName(op)
			return resType
		},
		"handlerSig": func(op *sadl.HttpDef) string {
			name := opName(op)
			resType := responseType(operationName(op))
//			resType, _ := gen.responsePojoName(op)
			return "public " + resType + " " + name + "(" + reqType(name) + " req)"
		},
		"resourceSig": func(op *sadl.HttpDef) string {
			name := opName(op) //i.e. "getFoo"
			var params []string
			for _, in := range op.Inputs {
				tn, _, _ := gen.typeName(nil, in.Type, false)
				param := tn + " " + in.Name
				if in.Query != "" {
					param = `@QueryParam("` + in.Query + `") ` + param
				} else if in.Header != "" {
					param = `@HeaderParam("` + in.Header + `") ` + param
				} else if in.Path {
					param = `@PathParam("` + in.Name + `") ` + param
				}
				if in.Default != nil {
					switch b := in.Default.(type) {
					case *string:
						param = fmt.Sprintf("@DefaultValue(%q)", *b) + " " + param
					case *sadl.Decimal:
						param = fmt.Sprintf("@DefaultValue(%q)", (*b).String()) + " " + param
					default:
						fmt.Println("Whoops:", sadl.Pretty(in.Default))
						panic("HERE")
					}
				}
				params = append(params, param)
			}
			etype := "Response"
			paramlist := strings.Join(params, ", ")
			return "public " + etype + " " + name + "(" + paramlist + ")"
		},
		"resourceBody": func(op *sadl.HttpDef) string {
			name := opName(op) //i.e. "Hello"
			reqname := reqType(name)
			resname := responseType(operationName(op))
			var params []string
			for _, in := range op.Inputs {
				params = append(params, in.Name)
			}
			ename, etype := entityNameType(op)
			var b bytes.Buffer
			writer := bufio.NewWriter(&b)
			writer.WriteString("        " + reqname + " req = new " + reqname + "()")
			for _, p := range params {
				writer.WriteString("." + p + "(" + p + ")")
			}
			writer.WriteString(";\n")
			writer.WriteString("        try {\n")
			if len(op.Expected.Outputs) > 0 {
				writer.WriteString("            " + resname + " res = impl." + name + "(req);\n")
				wrappedResult := ""
				if ename != "void" && ename != "" {
					wrappedResult = jsonWrapper(etype, "res." + ename)
				}
				ret := fmt.Sprintf("Response.status(%d)", + op.Expected.Status)
				for _, out := range op.Expected.Outputs {
					if out.Header != "" {
						ret = ret + ".header(\"" + out.Header + "\", res." + out.Name + ")"
					}
				}
				if wrappedResult != "" {
					ret = ret + ".entity(" + wrappedResult +")"
				}
				writer.WriteString("            return " + ret + ".build();\n")
			} else {
				writer.WriteString("            impl." + name + "(req);\n")
				writer.WriteString(fmt.Sprintf("            return Response.status(%d).build();\n", op.Expected.Status))
			}
			writer.WriteString("        } catch (ServiceException se) {\n")
			writer.WriteString("            Object entity = se.entity == null? se : se.entity;\n")
			writer.WriteString("            int status = 500;\n")
			first := true
			any := false
			anyNull := false
			for _, resp := range op.Exceptions {
				tn, _, _ := gen.typeName(nil, resp.Type, true)
				if tn != "ServiceException" {
					any = true
					if first {
						writer.WriteString("            if (entity instanceof " + tn + ") {\n")
						first = false
					} else {
						writer.WriteString("            } else if (entity instanceof " + tn + ") {\n")
					}
					writer.WriteString(fmt.Sprintf("                status = %d;\n", resp.Status))
					if resp.Status == 204 || resp.Status == 304 {
						writer.WriteString("                entity = null;\n")
						anyNull = true
					}
				}
			}
			if any {
				writer.WriteString("            }\n");
			}
			if anyNull {
				writer.WriteString("            if (entity == null) {\n")
				writer.WriteString("                throw new WebApplicationException(status);\n")
				writer.WriteString("            }\n")
			}
			writer.WriteString("            throw new WebApplicationException(Response.status(status).entity(entity).build());\n");
			writer.WriteString("        }\n")
			writer.Flush()
			return b.String()
		},
	}
	base := filepath.Join(dir, src)
	packageDir := base
	if pkg != "" {
		data.Package = pkg
		data.PackageLine = "package " + pkg + ";\n"
		packageDir = filepath.Join(base, javaPackageToPath(pkg))
	}
	
	err := createFileFromTemplate(base, data.MainClass + ".java", mainTemplate, data, funcMap)
	if err != nil {
		return err
	}
	err = createFileFromTemplate(packageDir, data.InterfaceClass + ".java", interfaceTemplate, data, funcMap)
	if err != nil {
		return err
	}
	err = createFileFromTemplate(packageDir, data.ResourcesClass + ".java", resourcesTemplate, data, funcMap)
	if err != nil {
		return err
	}
	err = createFileFromTemplate(packageDir, "ServiceException.java", exceptionTemplate, data, funcMap)
	if err != nil {
		return err
	}
	for _, op := range model.Operations {
		gen.createRequestPojo(op)
		gen.createResponsePojo(op)
//		for _, exc := range op.Exceptions {
//			gen.createExceptionPojo(op, exc)
//		}
	}
	return gen.err
}

func  jsonWrapper(etype string, val string) string {
	switch etype {
	case "String":
		return "Json.string(" + val + ")"
	default:
		return val //these are already valid JSON objects
	}
}

func addImportIfNeeded(pkgs []string, pkg string) []string {
	switch pkg {
	case "UUID":
		return adjoin(pkgs, "java.util.UUID")
	case "Timestamp":
		return adjoin(pkgs, "java.time.Instant")
	}
	return pkgs
}

const reqTemplate = `//
// Created by sadl2java
//
{{.PackageLine}}{{range .Imports}}
import {{.}};{{end}}

public class {{reqClass}} {{openBrace}}
{{range .Inputs}}{{if .Name}}
    public {{javaType .}} {{.Name}};
    public {{reqClass}} {{.Name}}({{javaType .}} {{.Name}}) { this.{{.Name}} = {{.Name}}; return this; }{{end}}
{{end}}
}
`

const resTemplate = `//
// Created by sadl2java
//
{{.PackageLine}}
public class {{reqClass}} {{openBrace}}
{{if .Output.Outputs}}    public {{outtype .Op}} {{outname .Op}};
    public {{reqClass}} {{outname .Op}}({{outtype .Op}} {{outname .Op}}) { this.{{outname .Op}} = {{outname .Op}}; return this; }{{end}}
}
`

func createFileFromTemplate(dir, file string, tmplSource string, data interface{}, funcMap template.FuncMap) error {
	path := filepath.Join(dir, file)
	if fileExists(path) {
		fmt.Printf("[%s already exists, not overwriting]\n", path)
		return nil
	}
   f, err := os.Create(path)
   if err != nil {
		return err
   }
	defer f.Close()
   writer := bufio.NewWriter(f)
	tmpl, err := template.New(file).Funcs(funcMap).Parse(tmplSource)
	if err != nil {
		return err
	}
	err = tmpl.Execute(writer, data)
	if err != nil {
		return err
	}
	writer.Flush()
	return nil
}

const mainTemplate = `//
// Sample server main program generated by sadl2java
//
import org.eclipse.jetty.server.Server;
import org.glassfish.jersey.jetty.JettyHttpContainerFactory;
import org.glassfish.jersey.server.ResourceConfig;
import org.glassfish.hk2.utilities.binding.AbstractBinder;
import org.glassfish.jersey.jackson.JacksonFeature;
import javax.ws.rs.core.UriBuilder;
import java.io.IOException;
import java.net.URI;
{{if .Package}}import {{.Package}}.*;{{end}}

public class {{.MainClass}} {

    public static String BASE_URI = "http://localhost:{{.Port}}/";

    public static void main(String[] args) {
        try {
            Server server = startServer(new {{.ImplClass}}());
            server.join();
        } catch (Exception e) {
            System.err.println("*** " + e);
        }
    }

    public static Server startServer({{.ImplClass}} impl) throws Exception {
        URI baseUri = UriBuilder.fromUri(BASE_URI).build();
        ResourceConfig config = new ResourceConfig({{.ResourcesClass}}.class);
        config.registerInstances(new AbstractBinder() {
                @Override
                protected void configure() {
                    bind(impl).to({{.InterfaceClass}}.class);
                }
            });
        Server server = JettyHttpContainerFactory.createServer(baseUri, config);
        server.start();
        System.out.println(String.format("Service started at %s", BASE_URI));
        return server;
    }


    // Stubs for an implementation of the service follow

    static class {{.ImplClass}} implements {{.InterfaceClass}} {
{{range .Model.Operations}}
        {{handlerSig .}} {{openBrace}}
            return new {{resClass .}}(); //implement me!
        }
{{end}}
    }
}
`


const resourcesTemplate = `//
// Example resources generated by sadl2java
//
{{.PackageLine}}

import java.util.*;
import javax.inject.Inject;
import javax.ws.rs.*;
import javax.ws.rs.core.*;
import static javax.ws.rs.core.Response.Status;

@Path("{{.RootPath}}")
public class {{.ResourcesClass}} {
    @Inject
    private {{.InterfaceClass}} impl;
{{range .Model.Operations}}
    
    @{{.Method}}
    @Path("{{methodPath .}}")
    @Produces(MediaType.APPLICATION_JSON)
    {{resourceSig .}} {{openBrace}}
{{resourceBody .}}    }
{{end}}
}
`

const interfaceTemplate = `//
// Example resources generated by sadl2java
//
{{.PackageLine}}

public interface {{.InterfaceClass}} {
{{range .Model.Operations}}
    {{handlerSig .}};
{{end}}
}
`

const exceptionTemplate = `//
// Generated by sadl2java
//
{{.PackageLine}}

import com.fasterxml.jackson.databind.annotation.JsonSerialize;
import com.fasterxml.jackson.databind.annotation.JsonDeserialize;
import com.fasterxml.jackson.core.JsonParser;
import com.fasterxml.jackson.databind.DeserializationContext;
import com.fasterxml.jackson.databind.JsonSerializer;
import com.fasterxml.jackson.databind.JsonDeserializer;
import com.fasterxml.jackson.core.JsonGenerator;
import com.fasterxml.jackson.databind.SerializerProvider;
import com.fasterxml.jackson.core.JsonProcessingException;
import java.io.IOException;

@JsonSerialize(using = ServiceException.JSONSerializer.class)
@JsonDeserialize(using = ServiceException.JSONDeserializer.class)
public class ServiceException extends RuntimeException {
    public Object entity;

    public ServiceException() { super(); }
    public ServiceException(String message) { super(message); }
    public ServiceException(String message, Throwable cause) { super(message, cause); }
    public ServiceException(Throwable cause) { super(cause); }

    public ServiceException entity(Object entity) { this.entity = entity; return this; }

    public static class ErrorObject {
        public String error;
        public ErrorObject(String error) { this.error = error; }
    }

    public static class JSONSerializer extends JsonSerializer<ServiceException> {
        @Override
        public void serialize(ServiceException value, JsonGenerator jgen, SerializerProvider provider) throws IOException, JsonProcessingException {
            Object tmp = value.entity;
            if (tmp == null) {
                tmp = new ErrorObject(value.getMessage());
            }
            jgen.writeObject(tmp);
        }
    }

    public static class JSONDeserializer extends JsonDeserializer<ServiceException> {
        @Override
        public ServiceException deserialize(JsonParser jp, DeserializationContext ctxt) throws IOException, JsonProcessingException {
            ErrorObject tmp = jp.readValueAs(ErrorObject.class);
            return new ServiceException(tmp.error);
        }
    }

}
`

func (gen *PojoGenerator) finishPojo(b bytes.Buffer, className string) {
	if gen.err == nil {
		gen.createJavaFile(className) //then create file and write the header with imports
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

func requestType(name string) string {
	return capitalize(name) + "Request"
}

func responseType(name string) string {
	return capitalize(name) + "Response"
}

func operationName(op *sadl.HttpDef) string {
	name := op.Name
	if name == "" {
		method := lowercase(op.Method)
		_, etype := opInfo(op)
		name = method + etype
	}
	return name
}

func (gen *PojoGenerator) createRequestPojo(op *sadl.HttpDef) {
	if gen.err != nil {
		return
	}
	ts := &sadl.TypeSpec{
		Type: "Struct",
	}
	for _, in := range op.Inputs {
		ts.Fields = append(ts.Fields, &in.StructFieldDef)
	}
	className := requestType(operationName(op))
   var b bytes.Buffer
	gen.writer = bufio.NewWriter(&b) //first write to a string
	gen.createStructPojo(ts, className, "")
	gen.writer.Flush()
	gen.finishPojo(b, className)
}

func (gen *PojoGenerator) responsePojoName(op *sadl.HttpDef) (string, bool) {
	for _, out := range op.Expected.Outputs {
		if out.Header == "" {
			tn, _, _ := gen.typeName(nil, out.Type, true)
			return tn, false
		}
	}
	return "void", false
}

func (gen *PojoGenerator) createResponsePojo(op *sadl.HttpDef) {
	if gen.err != nil {
		return
	}
	className := responseType(operationName(op))
	ts := &sadl.TypeSpec{
		Type: "Struct",
	}
	for _, spec := range op.Expected.Outputs {
		ts.Fields = append(ts.Fields, &spec.StructFieldDef)
	}
   var b bytes.Buffer
	gen.writer = bufio.NewWriter(&b) //first write to a string
	gen.createStructPojo(ts, className, "")
	gen.writer.Flush()
	gen.finishPojo(b, className)
}
