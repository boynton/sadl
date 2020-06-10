package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/graphql"
	"github.com/boynton/sadl/io"
	"github.com/boynton/sadl/openapi"
	"github.com/boynton/sadl/smithy"
)

var ImportFormats = []string{
	"sadl",
	"smithy",
	"openapi",
	"graphql",
}

var ImportFileExtensions = map[string][]string{
	".sadl":    []string{"sadl"},
	".smithy":  []string{"smithy"},
	".graphql": []string{"graphql"},
	".json":    []string{"sadl", "smithy", "openapi"},
	".yaml":    []string{"openapi"},
}

func ImportFile(path string, conf map[string]interface{}, extensions ...io.Extension) (*sadl.Model, error) {
	ext := filepath.Ext(path)
	if ftypes, ok := ImportFileExtensions[ext]; ok {
		if len(ftypes) == 1 { //we are not guessing
			return importFile(path, ftypes[0], conf, extensions)
		}
		//else guess by trying each one, in order. The error reporting is more generic in this case.
		for _, ftype := range ftypes {
			model, err := importFile(path, ftype, conf, extensions)
			if err == nil {
				return model, nil
			}
		}
	}
	return nil, fmt.Errorf("Cannot import file: %q\n", path)
}

func importFile(path string, ftype string, conf map[string]interface{}, extensions []io.Extension) (*sadl.Model, error) {
	switch ftype {
	case "sadl":
		if strings.HasSuffix(path, ".json") { //the primary SADL case, reports errors prettily
			return sadl.LoadModel(path)
		}
		return io.ParseSadlFile(path, conf, extensions...)
	case "smithy":
		return smithy.Import(path, conf)
	case "openapi":
		return openapi.Import(path, conf)
	case "graphql":
		return graphql.Import(path, conf)
	default:
		return nil, fmt.Errorf("Cannot import file: %q (file type %q not recognized)\n", path, ftype)
	}
}
