package java

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/boynton/sadl"
)

type ClientData struct {
	Name           string
	Model          *sadl.Model
	ModelPackage   string
	ClientPackage  string
	PackageLine    string
	Port           int
	RootPath       string
	InterfaceClass string
	Funcs          template.FuncMap
}

func (gen *Generator) CreateClient() {
	src := gen.SourceDir
	rez := gen.ResourceDir
	if gen.Err != nil {
		return
	}
	gen.CreateClientDataAndFuncMap(src, rez)
	gen.CreateClientConfig()
	gen.CreateJavaFileFromTemplate(gen.clientData.Name, clientTemplate, gen.clientData, gen.clientData.Funcs, gen.ClientPackage)
}

func (gen *Generator) CreateClientDataAndFuncMap(src, rez string) {
	if gen.Err != nil {
		return
	}
	serviceName := gen.Capitalize(gen.Model.Name)
	rootPath := gen.Model.Base
	if rootPath == "" {
		rootPath = "/"
	}
	gen.clientData = &ClientData{
		RootPath:       rootPath,
		Model:          gen.Model,
		ModelPackage:   gen.ModelPackage,
		ClientPackage:  gen.ClientPackage,
		Name:           serviceName + "Client",
		Port:           8080,
		InterfaceClass: serviceName,
	}
	reqType := func(name string) string {
		return gen.RequestType(name)
	}
	entityNameType := func(hact *sadl.HttpDef) (string, string) {
		for _, out := range hact.Expected.Outputs {
			if out.Header == "" {
				tn, _, _ := gen.TypeName(nil, out.Type, true)
				return out.Name, tn
			}
		}
		return "", "void"
	}
	fromString := func(typename, val string) string {
		switch typename {
		case "Instant":
			return "Instant.parse(" + val + ")"
		default:
			return val
		}
	}

	funcMap := template.FuncMap{
		"openBrace": func() string { return "{" },
		"handlerSig": func(hact *sadl.HttpDef) string {
			name := gen.ActionName(hact)
			resType := gen.ResponseType(gen.ActionName(hact))
			return "public " + resType + " " + name + "(" + reqType(name) + " req)"
		},
		"resEntity": func(hact *sadl.HttpDef) string {
			resClass := gen.ResponseType(gen.ActionName(hact))
			if gen.UseImmutable {
				return resClass + ".builder().build()"
			} else {
				return "new " + resClass + "()"
			}
		},
		"interfaceHttp": func(interfaceName string) []*sadl.HttpDef {
			return gen.Model.Http
		},
		"handlerBody": func(hact *sadl.HttpDef) string {
			var b bytes.Buffer
			writer := bufio.NewWriter(&b)
			pq := strings.Split(hact.Path, "?")
			path := pq[0]
			writer.WriteString("        WebTarget target = client.target(config.getTarget()).path(\"" + path + "\")")
			for _, in := range hact.Inputs {
				src := "req.get" + gen.Capitalize(in.Name) + "()"
				if in.Path {
					writer.WriteString("\n            .resolveTemplate(\"" + in.Name + "\", " + src + ", true)")
				} else if in.Query != "" {
					writer.WriteString("\n            .queryParam(\"" + in.Query + "\", " + src + ")")
				}
			}
			writer.WriteString(";\n        Invocation.Builder inv = target.request(MediaType.APPLICATION_JSON)")
			for _, in := range hact.Inputs {
				if in.Header != "" {
					src := "req.get" + gen.Capitalize(in.Name) + "()"
					writer.WriteString("\n            .header(\"" + in.Header + "\", " + src + ")")
				}
			}
			switch hact.Method {
			case "PUT", "POST":
				ename, _ := gen.ActionInfo(hact)
				src := "Entity.entity(req.get" + gen.Capitalize(ename) + "(), MediaType.APPLICATION_JSON)"
				writer.WriteString(";\n        Response response = inv." + strings.ToLower(hact.Method) + "(" + src + ");\n")
			case "GET", "DELETE":
				writer.WriteString(";\n        Response response = inv." + strings.ToLower(hact.Method) + "();\n")
			default:
				panic("fix me: method = " + hact.Method)
			}
			writer.WriteString("        switch (response.getStatus()) {\n")
			writer.WriteString("        case " + fmt.Sprint(hact.Expected.Status) + ":\n")
			if len(hact.Expected.Outputs) == 0 {
				writer.WriteString("            return null; //shoudn't this be void?!\n") //?
			} else {
				ename, etype := entityNameType(hact)
				if etype != "void" {
					writer.WriteString("            " + etype + " " + ename + " = response.readEntity(" + etype + ".class);\n")
				}
				for _, out := range hact.Expected.Outputs {
					if out.Header != "" {
						tn, _, _ := gen.TypeName(nil, out.Type, true)
						osrc := fromString(tn, "response.getHeaderString(\""+out.Header+"\")")
						writer.WriteString("            " + tn + " " + out.Name + " = " + osrc + ";\n")
					}
				}
				responseType := gen.ResponseType(hact.Name)
				writer.WriteString("            return " + responseType + ".builder()\n")
				if etype != "void" {
					writer.WriteString("                ." + ename + "(" + ename + ")\n")
				}
				for _, out := range hact.Expected.Outputs {
					if out.Header != "" {
						writer.WriteString("                ." + out.Name + "(" + out.Name + ")\n")
					}
				}
				writer.WriteString("                .build();\n")
			}
			for _, exc := range hact.Exceptions {
				writer.WriteString("        case " + fmt.Sprint(exc.Status) + ":\n")
				writer.WriteString("            throw response.readEntity(" + exc.Type + ".class);\n")
			}
			writer.WriteString("        default:\n")
			writer.WriteString("            throw new RuntimeException(\"Unexpected service response status: \" + response.getStatus());\n")
			writer.WriteString("        }\n")
			writer.Flush()
			return b.String()
		},
	}
	gen.clientData.Funcs = funcMap
}

func (gen *Generator) CreateClientConfig() {
	if gen.Err != nil {
		return
	}
	gen.Begin()
	gen.Emit(clientConfig)
	result := gen.End()
	if gen.Err == nil {
		gen.WriteJavaFile("ClientConfig", result, gen.clientData.ClientPackage)
	}
}

var clientConfig = `
public interface ClientConfig {
    public String getTarget();
}
`

const clientTemplate = `
{{if .ModelPackage}}import {{.ModelPackage}}.*;{{end}}
import java.time.Instant;
import javax.ws.rs.client.Client;
import javax.ws.rs.client.ClientBuilder;
import javax.ws.rs.client.Entity;
import javax.ws.rs.client.WebTarget;
import javax.ws.rs.client.Invocation;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;
import javax.ws.rs.WebApplicationException;

public class {{.Name}} implements {{.InterfaceClass}} {
    private static Client client = ClientBuilder.newClient();
    private static final String base = "/crudl";
    private ClientConfig config;

    public CrudlClient(ClientConfig conf) {
        config = conf;
    }
{{range .Model.Http}}
    {{handlerSig .}} {{openBrace}}
{{handlerBody .}}    }
{{end}}
}
`
