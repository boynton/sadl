package main

import(
	"bufio"
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
	Inputs []*sadl.HttpParamSpec
	Output *sadl.HttpResponseSpec
	Errors []*sadl.HttpResponseSpec
	Class string
	Imports []string
}

func createServer(model *sadl.Model, pkg, dir, src string) error {
	fmt.Println("OK:", sadl.Pretty(model))
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
			entityType := op.Output.Type
			name = lowercase(op.Method) + entityType
		}
		return name
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
		"reqClass":  func(op *sadl.HttpDef) string { return reqType(opName(op))},
		"resClass":  func(op *sadl.HttpDef) string { return resType(opName(op))},
		"reqInnerClass": func(op *sadl.HttpDef) string {
			return `public static class GetFooRequest {
        public UUID id;
        public GetFooRequest id(UUID id) { this.id = id; return this; }
    }
`},
		"resInnerClass": func(op *sadl.HttpDef) string {
			return "public static class GetFooResponse { public Foo body; public GetFooResponse body(Foo body) { this.body = body; return this; } }\n"
		},
		"handlerSig": func(op *sadl.HttpDef) string {
			name := opName(op)
			return "public " + resType(name) + " " + name + "(" + reqType(name) + " req)"
		},
		"resourceSig": func(op *sadl.HttpDef) string {
			name := opName(op) //i.e. "getFoo"
			rtype := op.Output.Type
			var params []string
			params = append(params, `@PathParam("id") UUID id`) //FIXME
			paramlist := strings.Join(params, ", ")
			return "public " + rtype + " " + name + "(" + paramlist + ")"
		},
		"resourceBody": func(r *sadl.HttpDef) string {
			//FIXME
			return `//fixme: set up headers, params, etc.
        GetFooRequest req = new GetFooRequest().id(id);
        try {
            GetFooResponse res = impl.getFoo(req);
            //set output headers, return body
            return res.body;
        } catch (ServiceException se) {
            if (se.entity instanceof NotFoundError) {
                throw new WebApplicationException(Response.status(404).entity(se.entity).build());
            }
            se.printStackTrace();
            throw new WebApplicationException(Response.status(500).build());
        }
`		},
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
		data.Imports = addImportIfNeeded(nil, op.Output.Type)
		data.Output = op.Output
		data.Errors = op.Errors
		cname = resType(opName(op))
		funcMap["reqClass"] = func() string { return cname }
		err = createFileFromTemplate(packageDir, cname + ".java", resTemplate, data, funcMap)
		if err != nil {
			return err
		}
	}
	return err
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
    public int status;
    public {{.Output.Type}} {{.Output.Name}};
    public {{reqClass}} {{.Output.Name}}({{.Output.Type}} {{.Output.Name}}) { this.{{.Output.Name}} = {{.Output.Name}}; return this; }
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
import {{.Package}}.*;

public class {{.MainClass}} {

    public static String BASE_URI = "http://localhost:{{.Port}}/";

    public static void main(String[] args) {
        try {
            Server server = startServer();
            server.join();
        } catch (Exception e) {
            System.err.println("*** " + e);
        }
    }

    public static Server startServer() throws Exception {
        URI baseUri = UriBuilder.fromUri(BASE_URI).build();
        ResourceConfig config = new ResourceConfig({{.ResourcesClass}}.class);
        //inject the implementation of {{.InterfaceClass}} into {{.ResourcesClass}} here.
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
    {{.InterfaceClass}} impl = null;
{{range .Model.Operations}}
    
    @{{.Method}}
    @Path("{{methodPath .}}")
    @Produces(MediaType.APPLICATION_JSON)
    {{resourceSig .}} {{openBrace}}
{{resourceBody .}}    }
{{end}}

    
    //-----------------------------

    static public class Hello {
        public String message;
    }

    @GET
    @Path("/hello")
    @Produces(MediaType.APPLICATION_JSON)
    public Hello hello() {
        Hello h = new Hello();
        h.message = "Hello there";
        return h;
    }

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
package model;

public class ServiceException extends RuntimeException {
    public Object entity;
    public ServiceException(Object entity) {
        this.entity = entity;
    }
}
`
