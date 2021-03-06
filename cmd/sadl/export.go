package main

import (
	"fmt"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/golang"
	"github.com/boynton/sadl/graphql"
	"github.com/boynton/sadl/httptrace"
	"github.com/boynton/sadl/java"
	"github.com/boynton/sadl/openapi"
	"github.com/boynton/sadl/smithy"
)

var ExportFormats = []string{
	"sadl",
	"smithy",
	"openapi",
	"graphql",
	"java",
}

func SetupCommandLineArgs(generator string) {
}

func ExportFiles(model *sadl.Model, generator, dir string, conf *sadl.Data) error {
	switch generator {
	case "json", "sadl-ast":
		fmt.Println(sadl.Pretty(model))
		return nil
	case "sadl":
		fmt.Println(sadl.DecompileSadl(model))
		return nil
	case "smithy":
		return smithy.Export(model, dir, conf, false)
	case "smithy-ast":
		return smithy.Export(model, dir, conf, true)
	case "openapi":
		return openapi.Export(model, conf)
	case "swagger-ui":
		return serveSwaggerUi(model, conf)
	case "graphql":
		return graphql.Export(model, conf)
	case "java", "java-model":
		return java.Export(model, dir, conf)
	case "java-server":
		conf.Put("server", true)
		return java.Export(model, dir, conf)
	case "java-client":
		conf.Put("client", true)
		return java.Export(model, dir, conf)
	case "go", "go-model":
		return golang.Export(model, dir, conf)
	case "go-server":
		conf.Put("server", true)
		return golang.Export(model, dir, conf)
	case "go-client":
		conf.Put("client", true)
		return golang.Export(model, dir, conf)
	case "http-trace":
		return httptrace.Export(model, conf)
	default:
		return fmt.Errorf("Unsupported generator: %s", generator)
	}
}
