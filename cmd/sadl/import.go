package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/graphql"
	"github.com/boynton/sadl/openapi"
	"github.com/boynton/sadl/smithy"
)

var ImportFormats = []string{
	"sadl",
	"smithy",
	"openapi",
	"swagger",
	"graphql",
}

var ImportFileExtensions = map[string][]string{
	".sadl":    []string{"sadl"},
	".smithy":  []string{"smithy"},
	".graphql": []string{"graphql"},
	".json":    []string{"sadl", "smithy", "openapi", "swagger"},
	".yaml":    []string{"openapi"},
}

func expandPaths(paths []string) ([]string, error) {
	var result []string
	for _, path := range paths {
		ext := filepath.Ext(path)
		if _, ok := ImportFileExtensions[ext]; ok {
			result = append(result, path)
		} else {
			fi, err := os.Stat(path)
			if err != nil {
				return nil, err
			}
			if fi.IsDir() {
				err = filepath.Walk(path, func(wpath string, info os.FileInfo, errIncoming error) error {
					if errIncoming != nil {
						return errIncoming
					}
					ext := filepath.Ext(wpath)
					if _, ok := ImportFileExtensions[ext]; ok {
						result = append(result, wpath)
					}
					return nil
				})
			}
		}
	}
	return result, nil
}

func ValidImportFileType(path string) string {
	ext := filepath.Ext(path)
	if ftypes, ok := ImportFileExtensions[ext]; ok {
		for _, ftype := range ftypes {
			switch ftype {
			case "sadl":
				if sadl.IsValidFile(path) {
					return ftype
				}
			case "smithy":
				if smithy.IsValidFile(path) {
					return ftype
				}
			case "graphql":
				if graphql.IsValidFile(path) {
					return ftype
				}
			case "openapi":
				if openapi.IsValidFile(path) {
					return ftype
				}
			case "swagger":
				if openapi.IsValidSwaggerFile(path) {
					return ftype
				}
			}
		}
	} else {
		panic("unknown file extension: " + ext)
	}
	return ""
}

func ImportFiles(paths []string, conf *sadl.Data, extensions ...sadl.Extension) (*sadl.Model, error) {
	chosenType := ""
	flatPathList, err := expandPaths(paths)
	if err != nil {
		return nil, err
	}
	var importPaths []string
	for _, path := range flatPathList {
		ftype := ValidImportFileType(path)
		if ftype != "" {
			if chosenType == "" {
				chosenType = ftype
			} else {
				if ftype != chosenType {
					return nil, fmt.Errorf("Multiple file types in input file list")
				}
			}
			importPaths = append(importPaths, path)
		}
	}
	if chosenType == "" {
		return nil, fmt.Errorf("Cannot determine file type for input file(s))\n")
	}
	return importFiles(importPaths, chosenType, conf, extensions)
}

func importFiles(paths []string, ftype string, conf *sadl.Data, extensions []sadl.Extension) (*sadl.Model, error) {
	switch ftype {
	case "sadl":
		if len(paths) != 1 {
			//To Do: merge files/models
			return nil, fmt.Errorf("SADL doesn't support merging models, and more than one file was specified.")
		}
		if strings.HasSuffix(paths[0], ".json") {
			return sadl.LoadModel(paths[0])
		}
		return sadl.ParseSadlFile(paths[0], conf, extensions...)
	case "smithy":
		return smithy.Import(paths, conf)
	case "swagger":
		return openapi.ImportSwagger(paths, conf)
	case "openapi":
		return openapi.Import(paths, conf)
	case "graphql":
		return graphql.Import(paths, conf)
	default:
		return nil, fmt.Errorf("Cannot import file(s): file type %q not recognized\n", ftype)
	}
}
