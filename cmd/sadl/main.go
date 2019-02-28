package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/extensions/graphql"
	"github.com/boynton/sadl/parse"
)

func main() {
	pVerbose := flag.Bool("v", false, "set to true to enable verbose output")
	pGraphql := flag.Bool("graphql", false, "use the graphql extension")
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("usage: sadl [options] file.sadl")
		os.Exit(1)
	}
	path := args[0]
	var model *sadl.Model
	var err error
	parse.Verbose = *pVerbose
	if *pGraphql {
		model, err = graphql.ParseFile(path)
	} else {
		model, err = parse.File(path)
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	fmt.Println(parse.Pretty(model))
}
