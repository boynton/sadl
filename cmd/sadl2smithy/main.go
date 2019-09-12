package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/smithy"
)

var verbose bool = false

func main() {
	pVerbose := flag.Bool("v", false, "set to true to enable verbose output")
	pNamespace := flag.String("ns", "smithy.example", "set the namespace for the generated file")
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("usage: sadl2smithy [-v] [--ns namespace] file")
		os.Exit(1)
	}
	verbose = *pVerbose
	path := args[0]
	name := path
	n := strings.LastIndex(name, "/")
	if n >= 0 {
		name = name[n+1:]
	}
	n = strings.LastIndex(name, ".")
	if n >= 0 {
		name = name[:n]
		name = strings.Replace(name, ".", "_", -1)
	}
	var model *smithy.Model

	if strings.HasSuffix(path, ".sadl") {
		schema, err := sadl.ParseFile(path)
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
		
		model, err = smithy.FromSADL(schema, *pNamespace)
		if err != nil {
			fmt.Printf("sadl2smithy: Cannot convert SADL to Smithy: %v\n", err)
			os.Exit(1)
		}
	} else if strings.HasSuffix(path, ".json") {
		//assume json smithy model
		b, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Printf("sadl2smithy: Cannot read Smithy file: %v\n", err)
			os.Exit(1)
		}
		var m smithy.Model
		err = json.Unmarshal(b, &m)
		if err != nil {
			fmt.Printf("sadl2smithy: Cannot unmarshal Smithy JSON file: %v\n", err)
			os.Exit(1)
		}
		model = &m
	}

	//JSON form
	if verbose {
		fmt.Println(sadl.Pretty(model))
	}

	//IDL
	fmt.Print(model.IDL())
}
