package main

import(
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/boynton/sadl"
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
	sadl.Verbose = *pVerbose
	schema, err := sadl.Parse(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	if true {
		pjson := sadl.Pretty(schema)
		fmt.Println(pjson)
		if true {
			var myschema sadl.Schema
			err = json.Unmarshal([]byte(pjson),&myschema)
			if err == nil {
				fmt.Println(sadl.Pretty(myschema))
			} else {
				fmt.Println("Cannot parse the JSON it generated:", err)
			}
		}
	}
}

