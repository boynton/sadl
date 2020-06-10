package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/util"
)

/*
To parse/load/import a supported spec format and invoke the default generator:
$ sadl foo.sadl
$ sadl foo.smithy
$ sadl foo.graphql -
$ sadl foo.json -> figures out the correct format to import (openapi, smithy ast, sadl json)

The default generator outputs formatted SADL source. Output is to stdout by default. The output directory can be specified:
$ sadl -dir . foo.sadl -> creates a
Other generators include:
$ sadl -g json foo.smithy -> outputs the pretty-printed JSON representation of the SADL model
$ sadl -g java-server foo.sadl -> generats
*/

func main() {
	pVerbose := flag.Bool("v", false, "set to true to enable verbose output")
	pDir := flag.String("dir", ".", "output directory for generated artifacts")
	pName := flag.String("name", "", "name of the model read. Overrides any existing name that is present")
	pNamespace := flag.String("namespace", "", "namespace to force input to, if the input has no namespace specified")
	pGen := flag.String("gen", "sadl", "the generator to run on the model")
	pConf := flag.String("conf", "", "the JSON config file to use to configure the generator")
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("usage: sadl [options] file.[sadl,smithy,graphql,json,yaml]")
		fmt.Println("       sadl version")
		os.Exit(1)
	}
	path := args[0]
	if path == "version" {
		fmt.Printf("SADL v%s\n", sadl.Version)
		os.Exit(0)
	}
	dir := *pDir
	importConf := make(map[string]interface{}, 0)
	if *pNamespace != "" {
		importConf["namespace"] = *pNamespace
	}
	if *pName != "" {
		importConf["name"] = *pName
	}
	util.Verbose = *pVerbose
	fmt.Println(util.Pretty(importConf))
	model, err := ImportFile(path, importConf)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	var conf map[string]interface{}
	if *pConf != "" {
		b, err := ioutil.ReadFile(*pConf)
		if err != nil {
			fmt.Printf("Cannot read config file %q: %v\n", *pConf, err)
			os.Exit(3)
		}
		err = json.Unmarshal(b, &conf)
		if err != nil {
			fmt.Printf("Cannot parse config file %q: %v\n", *pConf, err)
			os.Exit(3)
		}
	}
	err = ExportFiles(model, *pGen, dir, conf)
	if err != nil {
		fmt.Printf("*** %v\n", err)
		os.Exit(4)
	}
}
