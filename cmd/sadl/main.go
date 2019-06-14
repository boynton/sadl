package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/boynton/sadl"
)

func main() {
	pVerbose := flag.Bool("v", false, "set to true to enable verbose output")
	pFormat := flag.Bool("f", false, "set to true to format the file as SADL (unparse after parsing), else output JSON parse result")
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("usage: sadl [options] file.sadl")
		fmt.Println("       sadl version")
		os.Exit(1)
	}
	path := args[0]
	if path == "version" {
		fmt.Printf("SADL %s\n", sadl.Version)
		os.Exit(0)
	}
	sadl.Verbose = *pVerbose
	model, err := sadl.ParseFile(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	if *pFormat {
		fmt.Println(sadl.Decompile(model))
	} else {
		fmt.Println(sadl.Pretty(model))
	}
}
