package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/oas"
)

func main() {
	pVersion := flag.Int("v", 3, "export to the specified OAS version (default is 3)")
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("usage: sadl2oas [-v version] file")
		os.Exit(1)
	}
	version := *pVersion
	if version != 2 && version != 3 {
		fmt.Println("The 'version' to export to must be either 2 or 3:", version)
		os.Exit(1)
	}
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
	switch version {
	case 2:
		doc, err := gen.ExportToOAS2()
		if err != nil {
			fmt.Printf("sadl2oas: Cannot convert SADL to OAS v2: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(sadl.Pretty(doc))
	case 3:
		doc, err := gen.ExportToOAS3()
		if err != nil {
			fmt.Printf("sadl2oas: Cannot convert SADL to OAS v3: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(sadl.Pretty(doc))
	}
}
