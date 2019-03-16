package javagen

import (
	"bytes"
	"bufio"
	"fmt"
//	"path/filepath"
	"strings"
	"text/template"

	"github.com/boynton/sadl"
)

type ServerData struct {
	Name           string
	Model          *sadl.Model
	Package        string
	PackageLine    string
	Port           int
	MainClass      string
	ImplClass      string
	InterfaceClass string
	ResourcesClass string
	RootPath       string
	Op             *sadl.HttpDef
	Inputs         []*sadl.HttpParamSpec
	Expected       *sadl.HttpExpectedSpec
	Errors         []*sadl.HttpExceptionSpec
	Class          string
	Imports        []string
	Funcs template.FuncMap
	ExtraResources string
}

func (gen *Generator) CreateServerDataAndFuncMap(src, rez string) {
	if gen.Err != nil {
		return
	}
	if gen.ServerData != nil {
		return
	}
	serviceName := gen.Capitalize(gen.Model.Name)
	rootPath := gen.Model.Base
	if rootPath == "" {
		rootPath = "/"
	}
	gen.ServerData = &ServerData{
		RootPath:       rootPath,
		Model: gen.Model,
		Name:           serviceName,
		Port:           8080,
		MainClass:      "Main",
		ImplClass:      serviceName + "Impl",
		InterfaceClass: serviceName + "Service",
		ResourcesClass: serviceName + "Resources",
	}
	opName := func(op *sadl.HttpDef) string {
		return gen.OperationName(op)
	}
	entityNameType := func(op *sadl.HttpDef) (string, string) {
		for _, out := range op.Expected.Outputs {
			if out.Header == "" {
				tn, _, _ := gen.TypeName(nil, out.Type, true)
				return out.Name, tn
			}
		}
		return "", "void"
	}
	reqType := func(name string) string {
		return gen.RequestType(name)
	}
	funcMap := template.FuncMap{
		"openBrace": func() string { return "{" },
		"methodPath": func(op *sadl.HttpDef) string {
			path := op.Path
			i := strings.Index(path, "?")
			if i >= 0 {
				path = path[0:i]
			}
			return path
		},
		"outtype": func(op *sadl.HttpDef) string {
			_, t := gen.OperationInfo(op)
			return t
		},
		"outname": func(op *sadl.HttpDef) string {
			n, _ := gen.OperationInfo(op)
			return n
		},
		"reqClass": func(op *sadl.HttpDef) string { return reqType(opName(op)) },
		"resClass": func(op *sadl.HttpDef) string {
			return gen.ResponseType(gen.OperationName(op))
		},
		"handlerSig": func(op *sadl.HttpDef) string {
			name := opName(op)
			resType := gen.ResponseType(gen.OperationName(op))
			return "public " + resType + " " + name + "(" + reqType(name) + " req)"
		},
		"resourceSig": func(op *sadl.HttpDef) string {
			name := opName(op) //i.e. "getFoo"
			var params []string
			for _, in := range op.Inputs {
				tn, _, _ := gen.TypeName(nil, in.Type, false)
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
			resname := gen.ResponseType(gen.OperationName(op))
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
					wrappedResult = jsonWrapper(etype, "res."+ename)
				}
				ret := fmt.Sprintf("Response.status(%d)", +op.Expected.Status)
				for _, out := range op.Expected.Outputs {
					if out.Header != "" {
						ret = ret + ".header(\"" + out.Header + "\", res." + out.Name + ")"
					}
				}
				if wrappedResult != "" {
					ret = ret + ".entity(" + wrappedResult + ")"
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
				tn, _, _ := gen.TypeName(nil, resp.Type, true)
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
				writer.WriteString("            }\n")
			}
			if anyNull {
				writer.WriteString("            if (entity == null) {\n")
				writer.WriteString("                throw new WebApplicationException(status);\n")
				writer.WriteString("            }\n")
			}
			writer.WriteString("            throw new WebApplicationException(Response.status(status).entity(entity).build());\n")
			writer.WriteString("        }\n")
			writer.Flush()
			return b.String()
		},
		"extraResources": func() string { return gen.ServerData.ExtraResources },
	}
	gen.ServerData.Package = gen.Package
	if gen.Package != "" {
		gen.ServerData.PackageLine = "package " + gen.Package + ";\n"
	}
	gen.ServerData.Funcs = funcMap
}

func (gen *Generator) CreateServer(src, rez string) {
	if gen.Err != nil {
		return
	}
	gen.CreateServerDataAndFuncMap(src, rez)
	gen.CreateJavaFileFromTemplate(gen.ServerData.MainClass, mainTemplate, gen.ServerData, gen.ServerData.Funcs, "")
	gen.CreateJavaFileFromTemplate(gen.ServerData.InterfaceClass, interfaceTemplate, gen.ServerData, gen.ServerData.Funcs, gen.Package)
	gen.CreateJavaFileFromTemplate(gen.ServerData.ResourcesClass, resourcesTemplate, gen.ServerData, gen.ServerData.Funcs, gen.Package)
	gen.CreateJavaFileFromTemplate("ServiceException", exceptionTemplate, gen.ServerData, gen.ServerData.Funcs, gen.Package)
	for _, op := range gen.Model.Operations {
		gen.CreateRequestPojo(op)
		gen.CreateResponsePojo(op)
	}
}

const mainTemplate = `
import org.eclipse.jetty.server.Server;
import org.glassfish.jersey.jetty.JettyHttpContainerFactory;
import org.glassfish.jersey.server.ResourceConfig;
import org.glassfish.hk2.utilities.binding.AbstractBinder;
import org.glassfish.jersey.jackson.JacksonFeature;
import javax.ws.rs.core.UriBuilder;
import java.io.IOException;
import java.net.URI;
{{if .Package}}import {{.Package}}.*;{{end}}

// A placeholder implementation and launcher for the service
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

const resourcesTemplate = `
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
{{extraResources}}
}
`

const interfaceTemplate = `
public interface {{.InterfaceClass}} {
{{range .Model.Operations}}
    {{handlerSig .}};
{{end}}
}
`

const exceptionTemplate = `
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

func (gen *Generator) CreateRequestPojo(op *sadl.HttpDef) {
	if gen.Err != nil {
		return
	}
	ts := &sadl.TypeSpec{
		Type: "Struct",
	}
	for _, in := range op.Inputs {
		ts.Fields = append(ts.Fields, &in.StructFieldDef)
	}
	className := gen.RequestType(gen.OperationName(op))
	gen.CreatePojo(ts, className, "")
}

func (gen *Generator) responsePojoName(op *sadl.HttpDef) (string, bool) {
	for _, out := range op.Expected.Outputs {
		if out.Header == "" {
			tn, _, _ := gen.TypeName(nil, out.Type, true)
			return tn, false
		}
	}
	return "void", false
}

func (gen *Generator) CreateResponsePojo(op *sadl.HttpDef) {
	if gen.Err != nil {
		return
	}
	className := gen.ResponseType(gen.OperationName(op))
	ts := &sadl.TypeSpec{
		Type: "Struct",
	}
	for _, spec := range op.Expected.Outputs {
		ts.Fields = append(ts.Fields, &spec.StructFieldDef)
	}
	gen.CreatePojo(ts, className, "")
}

func (gen *Generator) OperationInfo(op *sadl.HttpDef) (string, string) {
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

func (gen *Generator) OperationName(op *sadl.HttpDef) string {
	name := op.Name
	if name == "" {
		method := strings.ToLower(op.Method)
		_, etype := gen.OperationInfo(op)
		name = method + etype
	} else {
		name = gen.Uncapitalize(name)
	}
	return name
}

func (gen *Generator) RequestType(name string) string {
	return gen.Capitalize(name) + "Request"
}

func (gen *Generator) ResponseType(name string) string {
	return gen.Capitalize(name) + "Response"
}

func jsonWrapper(etype string, val string) string {
	switch etype {
	case "String":
		return "Json.string(" + val + ")"
	default:
		return val //these are already valid JSON objects
	}
}
