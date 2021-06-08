package java

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/boynton/sadl"
)

type ServerData struct {
	Name           string
	Model          *sadl.Model
	ModelPackage   string
	ServerPackage  string
	PackageLine    string
	Port           int
	MainClass      string
	ResourcesClass string
	ImplClass      string
	InterfaceClass string
	RootPath       string
	Http           []*sadl.HttpDef
	Inputs         []*sadl.HttpParamSpec
	Expected       *sadl.HttpExpectedSpec
	Errors         []*sadl.HttpExceptionSpec
	Class          string
	Imports        []string
	Funcs          template.FuncMap
	Interfaces     map[string][]string
	ExtraResources string
}

//type ScopedHttpDef struct {
//	sadl.HttpDef
//	InterfaceName string
//}

func (gen *Generator) CreateServer() {
	src := gen.SourceDir
	rez := gen.ResourceDir
	if gen.Err != nil {
		return
	}
	gen.CreateServerDataAndFuncMap(src, rez)
	gen.CreateJavaFileFromTemplate(gen.ServerData.MainClass, mainTemplate, gen.ServerData, gen.ServerData.Funcs, "")
	gen.CreateJavaFileFromTemplate(gen.ServerData.ResourcesClass, resourcesTemplate, gen.ServerData, gen.ServerData.Funcs, gen.ServerPackage)
	if gen.ServerImpl {
		gen.CreateJavaFileFromTemplate(gen.ServerData.ImplClass, implTemplate, gen.ServerData, gen.ServerData.Funcs, "")
	}
}

func (gen *Generator) ExceptionTypes() map[string]string {
	exceptions := make(map[string]string, 0)
	for _, hact := range gen.Model.Http {
		for _, resp := range hact.Exceptions {
			tn, _, _ := gen.TypeName(nil, resp.Type, true)
			exceptions[tn] = resp.Type
		}
	}
	return exceptions
}

func firstTag(annos map[string]string) string {
	if csv, ok := annos["x_tags"]; ok {
		return strings.Split(csv, ",")[0]
	}
	return ""
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
		Model:          gen.Model,
		Name:           serviceName,
		ModelPackage:   gen.ModelPackage,
		ServerPackage:  gen.ServerPackage,
		Port:           8000,
		MainClass:      "Main",
		ResourcesClass: serviceName + "Resources",
	}

	gen.ServerData.InterfaceClass = serviceName
	gen.ServerData.ImplClass = serviceName + "Controller"
	entityNameType := func(hact *sadl.HttpDef) (string, string) {
		for _, out := range hact.Expected.Outputs {
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
	implName := func(base string) string {
		return sadl.Uncapitalize(base) + "Controller"
	}
	implTypeName := func(base string) string {
		return base + "Controller"
	}

	funcMap := template.FuncMap{
		"openBrace": func() string { return "{" },
		"methodPath": func(hact *sadl.HttpDef) string {
			path := hact.Path
			i := strings.Index(path, "?")
			if i >= 0 {
				path = path[0:i]
			}
			return path
		},
		"implTypeName": implTypeName,
		"implName":     implName,
		"implConstructors": func() string {
			var lst []string
			for iname, _ := range gen.ServerData.Interfaces {
				lst = append(lst, "new "+implTypeName(iname)+"()")
			}
			return strings.Join(lst, ", ")
		},
		"implDecls": func() string {
			var lst []string
			for iname, _ := range gen.ServerData.Interfaces {
				lst = append(lst, implTypeName(iname)+" "+implName(iname))
			}
			return strings.Join(lst, ", ")
		},
		"interfaceName": func(base string) string { return sadl.Uncapitalize(base) },
		/*		"interfaceHttp": func() []*ScopedxxxHttpDef {
				var tmp []*ScopedHttpDef
				for _, hname := range gen.ServerData.Interfaces[interfaceName] {
					h := gen.Model.FindHttp(hname)
					h2 := &ScopedHttpDef{
						HttpDef:       *h,
						InterfaceName: interfaceName,
					}
					tmp = append(tmp, h2)
				}
				return tmp
			},*/
		"outtype": func(hact *sadl.HttpDef) string {
			_, t := gen.ActionInfo(hact)
			return t
		},
		"instantProvider": func() string {
			if !gen.NeedInstant {
				return ""
			}
			return "config.register(Util.InstantConverterProvider.class);\n        "
		},
		"outname": func(hact *sadl.HttpDef) string {
			n, _ := gen.ActionInfo(hact)
			return n
		},
		"reqClass": func(hact *sadl.HttpDef) string { return reqType(gen.ActionName(hact)) },
		"resClass": func(hact *sadl.HttpDef) string {
			return gen.ResponseType(gen.ActionName(hact))
		},
		"resEntity": func(hact *sadl.HttpDef) string {
			resClass := gen.ResponseType(gen.ActionName(hact))
			if gen.UseImmutable {
				return resClass + ".builder().build()"
			} else {
				return "new " + resClass + "()"
			}
		},
		"handlerSig": func(hact *sadl.HttpDef) string {
			name := gen.ActionName(hact)
			resType := gen.ResponseType(gen.ActionName(hact))
			return "public " + resType + " " + name + "(" + reqType(name) + " req)"
		},
		"resourceSig": func(hact *sadl.HttpDef) string {
			name := gen.ActionName(hact) //i.e. "getFoo"
			var params []string
			for _, in := range hact.Inputs {
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
					case bool:
						param = fmt.Sprintf("@DefaultValue(\"%v\")", b) + " " + param
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
		"resourceBody": func(hact *sadl.HttpDef) string {
			iname := gen.Name
			name := gen.ActionName(hact)
			reqname := reqType(name)
			resname := gen.ResponseType(gen.ActionName(hact))
			var params []string
			for _, in := range hact.Inputs {
				params = append(params, in.Name)
			}
			ename, etype := entityNameType(hact)
			var b bytes.Buffer
			writer := bufio.NewWriter(&b)
			if gen.UseImmutable {
				writer.WriteString("        " + reqname + " req = " + reqname + ".builder()")
			} else {
				writer.WriteString("        " + reqname + " req = new " + reqname + "()")
			}
			for _, p := range params {
				writer.WriteString("." + p + "(" + p + ")")
			}
			if gen.UseImmutable {
				writer.WriteString(".build()")
			}
			writer.WriteString(";\n")
			writer.WriteString("        try {\n")
			if len(hact.Expected.Outputs) > 0 {
				writer.WriteString("            " + resname + " res = " + implName(iname) + "." + name + "(req);\n")
				wrappedResult := ""
				if ename != "void" && ename != "" {
					if gen.UseImmutable {
						wrappedResult = gen.jsonWrapper(etype, "res.get"+gen.Capitalize(ename)+"()")
					} else {
						wrappedResult = gen.jsonWrapper(etype, "res."+ename)
					}
				}
				ret := fmt.Sprintf("Response.status(%d)", +hact.Expected.Status)
				for _, out := range hact.Expected.Outputs {
					if out.Header != "" {
						ret = ret + ".header(\"" + out.Header + "\", res.get" + gen.Capitalize(out.Name) + "())"
					}
				}
				if wrappedResult != "" {
					ret = ret + ".entity(" + wrappedResult + ")"
				}
				writer.WriteString("            return " + ret + ".build();\n")
			} else {
				writer.WriteString("            " + implName(iname) + "." + name + "(req);\n")
				writer.WriteString(fmt.Sprintf("            return Response.status(%d).build();\n", hact.Expected.Status))
			}

			if gen.Config.GetBool("service-exception") {
				writer.WriteString("        } catch (ServiceException se) {\n")
				writer.WriteString("            Object entity = se.entity == null? se : se.entity;\n")
				writer.WriteString("            int status = 500;\n")
				first := true
				any := false
				anyNull := false
				for _, resp := range hact.Exceptions {
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
			} else {
				for _, resp := range hact.Exceptions {
					status := fmt.Sprint(resp.Status)
					tn, _, _ := gen.TypeName(nil, resp.Type, true)
					writer.WriteString("        } catch (" + tn + " e) {\n")
					writer.WriteString("                throw new WebApplicationException(Response.status(" + status + ").entity(e).build());\n")
				}
			}
			writer.WriteString("        } catch (Throwable th) {\n")
			writer.WriteString("            return Response.status(500).build();\n")
			writer.WriteString("        }\n")
			writer.Flush()
			return b.String()
		},
		"extraResources": func() string { return gen.ServerData.ExtraResources },
	}
	if gen.ServerPackage != "" {
		gen.ServerData.PackageLine = "package " + gen.ServerPackage + ";\n"
	}
	gen.ServerData.Funcs = funcMap
}

const mainTemplate = `
import org.eclipse.jetty.server.Server;
import org.glassfish.jersey.jetty.JettyHttpContainerFactory;
import org.glassfish.jersey.server.ResourceConfig;
import org.glassfish.jersey.logging.LoggingFeature;
import org.glassfish.hk2.utilities.binding.AbstractBinder;
import org.glassfish.jersey.jackson.JacksonFeature;
import javax.ws.rs.core.UriBuilder;
import java.io.IOException;
import java.net.URI;
import java.util.logging.Logger;
import java.util.logging.Level;
{{if .ModelPackage}}import {{.ModelPackage}}.*;{{end}}
{{if .ServerPackage}}import {{.ServerPackage}}.*;{{end}}

// A placeholder implementation and launcher for the service
public class {{.MainClass}} {
    public static String BASE_URI = "http://localhost:{{.Port}}/";
    public static void main(String[] args) {
        try {
            Server server = startServer({{implConstructors}});
            server.join();
        } catch (Exception e) {
            System.err.println("*** " + e);
        }
    }
    public static Server startServer({{implDecls}}) throws Exception {
        URI baseUri = UriBuilder.fromUri(BASE_URI).build();
        ResourceConfig config = new ResourceConfig({{.ResourcesClass}}.class);
        config.register(new LoggingFeature(Logger.getLogger(LoggingFeature.DEFAULT_LOGGER_NAME),
                                           Level.INFO, LoggingFeature.Verbosity.PAYLOAD_ANY, 10000));
        {{instantProvider}}config.registerInstances(new AbstractBinder() {
                @Override
                protected void configure() {
                    bind({{.ImplClass}}.class).to({{.InterfaceClass}}.class);
                }
            });
        Server server = JettyHttpContainerFactory.createServer(baseUri, config);
        server.start();
        System.out.println(String.format("Service started at %s", BASE_URI));
        return server;
    }

}
`

const resourcesTemplate = `
{{if .ModelPackage}}import {{.ModelPackage}}.*;{{end}}
import java.util.*;
import java.time.Instant;
import javax.inject.Inject;
import javax.ws.rs.*;
import javax.ws.rs.core.*;
import static javax.ws.rs.core.Response.Status;
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonInclude.Include;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.fasterxml.jackson.databind.DeserializationFeature;

@Path("{{.RootPath}}")
public class {{.ResourcesClass}} {
    @Inject
    private {{.InterfaceClass}} {{implName .Name}};
{{range .Model.Http}}
    
    @{{.Method}}
    @Path("{{methodPath .}}")
    @Produces(MediaType.APPLICATION_JSON)
    {{resourceSig .}} {{openBrace}}
{{resourceBody .}}    }
{{end}}
{{extraResources}}
}
`

const implTemplate = `
// Stubs for an implementation of the service follow
{{if .ModelPackage}}import {{.ModelPackage}}.*;{{end}}

public class {{.ImplClass}} implements {{.InterfaceClass}} {
{{range .Model.Http}}
    {{handlerSig .}} {{openBrace}}
        return {{resEntity .}}; //implement me!
    }
{{end}}
}
`

func (gen *Generator) CreateRequestPojo(hact *sadl.HttpDef) {
	if gen.Err != nil {
		return
	}
	ts := &sadl.TypeSpec{
		Type: "Struct",
	}
	for _, in := range hact.Inputs {
		ts.Fields = append(ts.Fields, &in.StructFieldDef)
	}
	className := gen.RequestType(gen.ActionName(hact))
	gen.CreatePojo(ts, className, "", nil)
}

func (gen *Generator) responsePojoName(hact *sadl.HttpDef) (string, bool) {
	for _, out := range hact.Expected.Outputs {
		if out.Header == "" {
			tn, _, _ := gen.TypeName(nil, out.Type, true)
			return tn, false
		}
	}
	return "void", false
}

func (gen *Generator) CreateResponsePojo(hact *sadl.HttpDef) {
	if gen.Err != nil {
		return
	}
	className := gen.ResponseType(gen.ActionName(hact))
	ts := &sadl.TypeSpec{
		Type: "Struct",
	}
	for _, spec := range hact.Expected.Outputs {
		ts.Fields = append(ts.Fields, &spec.StructFieldDef)
	}
	gen.CreatePojo(ts, className, "", nil)
}

func (gen *Generator) ActionInfo(hact *sadl.HttpDef) (string, string) {
	switch hact.Method {
	case "POST", "PUT":
		for _, in := range hact.Inputs {
			if in.Query == "" && in.Header == "" && !in.Path {
				return in.Name, in.Type
			}
		}
	default:
		for _, out := range hact.Expected.Outputs {
			if out.Header == "" {
				return out.Name, out.Type
			}
		}
	}
	return "anonymous", "Object"
}

func (gen *Generator) ActionName(hact *sadl.HttpDef) string {
	name := hact.Name
	if name == "" {
		method := strings.ToLower(hact.Method)
		_, etype := gen.ActionInfo(hact)
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

func (gen *Generator) jsonWrapper(etype string, val string) string {
	switch etype {
	case "String":
		gen.NeedUtil = true
		return "Util.toJson(" + val + ")"
	default:
		return val //these are already valid JSON objects
	}
}
