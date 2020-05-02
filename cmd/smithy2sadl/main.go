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
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("usage: smithy2sadl [-v] file")
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

	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Printf("smithy2sadl: Cannot read source file: %v\n", err)
		os.Exit(1)
	}
	if strings.HasSuffix(path, ".json") {
		err = json.Unmarshal(data, &model)
		if err != nil {
			fmt.Printf("smithy2sadl: Cannot parse source file: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(sadl.Pretty(model))
	}
}
