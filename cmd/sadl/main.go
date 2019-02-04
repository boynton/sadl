package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/boynton/sadl"
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
	schema, err := parse.File(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	if true {
		pjson := parse.Pretty(schema)
		fmt.Println(pjson)
		if false {
			var myschema sadl.Schema
			err = json.Unmarshal([]byte(pjson), &myschema)
			if err == nil {
				fmt.Println(parse.Pretty(myschema))
			} else {
				fmt.Println("Cannot parse the JSON it generated:", err)
			}
		}
	}
}
