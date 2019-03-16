package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/boynton/sadl/parse"
)

func main() {
	pVerbose := flag.Bool("v", false, "set to true to enable verbose output")
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("usage: sadl [options] file.sadl")
		os.Exit(1)
	}
	path := args[0]
	parse.Verbose = *pVerbose
	model, err := parse.File(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	fmt.Println(parse.Pretty(model))
}
