package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

//	"github.com/boynton/sadl"
	"github.com/boynton/sadl/parse"
	"github.com/boynton/sadl/oas"
	"github.com/boynton/sadl/gen/sadl"
)

var _ = sadl.Decompile

var verbose bool = false

func main() {
	pVerbose := flag.Bool("v", false, "set to true to enable verbose output")
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("usage: oas2sadl file")
		os.Exit(1)
	}
	verbose = *pVerbose
	path := args[0]
	name := path
	n := strings.LastIndex(name, "/")
	format := ""
	if n >= 0 {
		name = name[n+1:]
		n = strings.LastIndex(name, ".")
		if n >= 0 {
			format = name[n+1:]
			name = name[:n]
			name = strings.Replace(name, ".", "_", -1)
		}
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Printf("oas2sadl: Cannot read source file: %v\n", err)
		os.Exit(1)
	}
	oas, err := oas.Parse(data, format)
	if err != nil {
		fmt.Printf("oas2sadl: Cannot parse from %s source: %v\n", format, err)
		os.Exit(1)
	}
//	fmt.Println("parsed:", Pretty(oas))
//	os.Exit(1)
	model, err := oas.ToSadl(name)
	if err != nil {
		fmt.Printf("oas2sadl: Cannot convert to SADL: %v\n", err)
		os.Exit(1)
	}
	if false {
		fmt.Println(parse.Pretty(model)) //debug
	} else {
		fmt.Println(sadl.Decompile(model))
	}
}

func Pretty(obj interface{}) string {
	return parse.Pretty(obj)
}
