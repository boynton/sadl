package main

import (
	"fmt"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/graphql"
	"github.com/boynton/sadl/io"
	"github.com/boynton/sadl/java"
	"github.com/boynton/sadl/openapi"
	"github.com/boynton/sadl/smithy"
	"github.com/boynton/sadl/util"
)

func ExportFiles(model *sadl.Model, generator, dir string, conf map[string]interface{}) error {
	switch generator {
	case "json", "sadl-ast":
		fmt.Println(util.Pretty(model))
		return nil
	case "sadl":
		fmt.Println(io.DecompileSadl(model))
		return nil
	case "smithy":
		return smithy.Export(model, dir, conf, false)
	case "smithy-ast":
		return smithy.Export(model, dir, conf, true)
	case "java":
		return java.Export(model, dir, conf)
	case "java-server":
		if conf == nil {
			conf = make(map[string]interface{}, 0)
		}
		conf["server"] = true
		return java.Export(model, dir, conf)
	case "openapi":
		return openapi.Export(model, conf)
	case "graphql":
		return graphql.Export(model, conf)
	default:
		return fmt.Errorf("Unsupported generator: %s", generator)
	}
}
