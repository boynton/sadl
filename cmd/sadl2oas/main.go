package main

import (
	"flag"
	"fmt"
//	"io/ioutil"
	"os"
	"strings"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/oas"
)

var verbose bool = false

func main() {
	pVerbose := flag.Bool("v", false, "set to true to enable verbose output")
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("usage: sadl2oas file")
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
	schema, err := sadl.ParseFile(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	gen := oas.NewGenerator(schema, "/tmp") //DO: remove the outdir arg, will print to stdout
	doc, err := gen.ExportToOAS3()
	if err != nil {
		fmt.Printf("sadl2oas: Cannot convert SADL to OAS: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(sadl.Pretty(doc))
}
