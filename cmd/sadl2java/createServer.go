package main

import(
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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
	Output *sadl.HttpResponseSpec
	Errors []*sadl.HttpResponseSpec
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
		for _, out := range op.Output.Outputs {
			if out.Header == "" {
				return out.Name, out.Type
			}
		}
	}
	return "anonymous", "Object"
}

func createServer(model *sadl.Model, pkg, dir, src string) error {
	serviceName := capitalize(model.Name)
	data := &serverData{
		RootPath: "/" + lowercase(model.Name),
		Model: model,
		Name: model.Name,
		Port: 8080,
		MainClass: "Main",
		ImplClass: serviceName + "Impl",
		InterfaceClass: serviceName + "Service",
		ResourcesClass: serviceName + "Resources",
	}
	opName := func(op *sadl.HttpDef) string {
		name := op.Name
		if name == "" {
			method := lowercase(op.Method)
			_, etype := opInfo(op)
			name = method + etype
		}
		return name
	}
	entityNameType := func(op *sadl.HttpDef) (string, string) {
		for _, out := range op.Output.Outputs {
			if out.Header == "" { //must be the body
				return out.Name, out.Type
			}
		}
		return "", ""
	}
	reqType := func(name string) string {
		return capitalize(name) + "Request"
	}
	resType := func(name string) string {
		return capitalize(name) + "Response"
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
		"resClass":  func(op *sadl.HttpDef) string { return resType(opName(op))},
		"handlerSig": func(op *sadl.HttpDef) string {
			name := opName(op)
			return "public " + resType(name) + " " + name + "(" + reqType(name) + " req)"
		},
		"resourceSig": func(op *sadl.HttpDef) string {
			name := opName(op) //i.e. "getFoo"
			var params []string
			for _, in := range op.Inputs {
				param := in.Type + " " + in.Name
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
						param = fmt.Sprintf("@DefaultValue(%q)", *b) + " " + param //strings only
					default:
						fmt.Println("Whoops:", sadl.Pretty(in.Default))
					}
				}
				params = append(params, param)
			}
			_, etype := entityNameType(op)
			paramlist := strings.Join(params, ", ")
			return "public " + etype + " " + name + "(" + paramlist + ")"
		},
		"resourceBody": func(op *sadl.HttpDef) string {
			name := opName(op) //i.e. "Hello"
			reqname := reqType(name)
			resname := resType(name)
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
			writer.WriteString("            " + resname + " res = impl." + name + "(req);\n")
			if ename != "" {
				wrappedResult := jsonWrapper(etype, "res." + ename)
				writer.WriteString("            return " + wrappedResult + ";\n")
			} else {
				writer.WriteString("            return null;\n")
			}
			writer.WriteString("        } catch (ServiceException se) {\n")
			for _, errRes := range op.Errors {
				writer.WriteString(fmt.Sprintf("            // %d %s\n", errRes.Status))
			   /* if (se.entity instanceof NotFoundError) {
				throw new WebApplicationException(Response.status(404).entity(se.entity).build());
			}*/
			}
			writer.WriteString("			     se.printStackTrace();\n")
			writer.WriteString("			     throw new WebApplicationException(Response.status(500).build());\n")
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
		cname := reqType(opName(op))
		funcMap["reqClass"] = func() string { return cname }
		data.Inputs = op.Inputs
		for _, in := range op.Inputs {
			data.Imports = addImportIfNeeded(data.Imports, in.Type)
		}
		err := createFileFromTemplate(packageDir, cname + ".java", reqTemplate, data, funcMap)
		if err != nil {
			return err
		}
		//fixme: headers
		_, otype := opInfo(op)
		data.Imports = addImportIfNeeded(nil, otype)
		data.Output = op.Output
		data.Errors = op.Errors
		data.Op = op
		cname = resType(opName(op))
		funcMap["reqClass"] = func() string { return cname }
		err = createFileFromTemplate(packageDir, cname + ".java", resTemplate, data, funcMap)
		if err != nil {
			return err
		}
	}
	return err
}

func  jsonWrapper(etype string, val string) string {
	switch etype {
	case "Struct", "Array":
		return val //these are already valid JSON objects
	default:
		return "Json.string(" + val + ")"
	}
}

func addImportIfNeeded(pkgs []string, pkg string) []string {
	switch pkg {
	case "UUID":
		return adjoin(pkgs, "java.util.UUID")
	case "Timestamp":
		return adjoin(pkgs, "java.time.Instance")
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
    public {{.Type}} {{.Name}};
    public {{reqClass}} {{.Name}}({{.Type}} {{.Name}}) { this.{{.Name}} = {{.Name}}; return this; }{{end}}
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

public class ServiceException extends RuntimeException {
    public Object entity;
    public ServiceException(Object entity) {
        this.entity = entity;
    }
}
`
