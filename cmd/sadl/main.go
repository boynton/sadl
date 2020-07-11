package main

import (
	//	"encoding/json"
	"flag"
	"fmt"
	//	"io/ioutil"
	"os"

	"github.com/boynton/sadl"
)

func main() {
	helpMessage := `

Supported API description formats for each input file extension:
   .sadl     sadl
   .smithy   smithy
   .graphql  graphql
   .json     sadl, smithy, openapi (inferred by looking at the file contents)
   .yaml     openapi

The 'name' and 'namespace' options allow specifying those attributes for input formats
that do not require or support them. Otherwise a default is used.

The '-c' option specifies the name of a JSON config file. Code generators generally use the
'-o' option to specify the output directory, conversions to other API description formats
just output to stdout.

Supported generators and options used from config if present
   sadl: Prints the SADL representation to stdout. This is the default.
   json: Prints the SADL data representation in JSON to stdout
   smithy: Prints the Smithy IDL representation to stdout. Options:
      name: supply this value as the name for a service for inputs that do not have a name
      namespace: supply this value as the namespace for inputs that do not have a namespace
   smithy-ast: Prints the Smithy AST represenbtation to stdout, same options as 'smithy'
   openapi: Prints the OpenAPI Spec v3 representation to stdout
   graphql: Prints the GraphQL representation to stdout.
   java: Generate Java code for the model, server, client plumbing. Options:
      header: a string to include at the top of every generated java file
      lombok: use Lombok for generated model POJOs to reduce boilerplate
      getters: create model POJOs with traditional getter/setter style
      source: specify the default source directory, default to "src/main/java"
      resource: specify the default resouece directory, default to "src/main/resource"
      server: include server plumbing code, using Jersey for JAX-RS implementation
      maven: generate a maven pom.xml file to build the project
      instants: use java.time.Instant for Timestamp impl, else generate a Timestamp class.
   java-server: a shorthand for specifying the "server" option to the "java" generator
   http-trace: Generates an HTTP (curl-style) simulation of the API's example HTTP actions.

`
	pType := flag.String("t", "sadl", "Only read files of this type. By default, any valid input file type is accepted.")
	pOut := flag.String("o", ".", "The output file or directory. Defaults to current directory")
	pName := flag.String("n", "", "The name of the model, overrides any name present in the source")
	pNamespace := flag.String("ns", "", "The namespace of the model, overrides any namespace present in the source")
	pGen := flag.String("g", "sadl", "The generator for output")
	pConf := flag.String("c", "", "The JSON config file for default settings. Default is $HOME/.sadl-config.json")
	pForce := flag.Bool("f", false, "Force overwrite of existing files")
	pHelp := flag.Bool("help", false, "Show more helpful information")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: sadl [options] file ...\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *pHelp {
		fmt.Fprintf(os.Stderr, helpMessage)
		os.Exit(1)
	}

	args := flag.Args()
	out := *pOut
	name := *pName
	namespace := *pNamespace
	gen := *pGen
	configPath := *pConf
	force := *pForce
	formatType := *pType

	if len(args) == 0 {
		flag.Usage()
		os.Exit(0)
	}
	importConf := make(map[string]interface{}, 0)
	if namespace != "" {
		importConf["namespace"] = namespace
	}
	if name != "" {
		importConf["name"] = name
	}
	if formatType != "" {
		importConf["type"] = formatType
	}
	model, err := ImportFiles(args, importConf)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	var conf *sadl.Data
	if configPath != "" {
		conf, err = sadl.DataFromFile(configPath)
		if err != nil {
			fmt.Printf("Cannot read config file %q: %v\n", configPath, err)
		}
	} else {
		conf = &sadl.Data{}
	}
	conf.Put("force-overwrite", force)
	err = ExportFiles(model, gen, out, conf.AsMap())
	if err != nil {
		fmt.Printf("*** %v\n", err)
		os.Exit(4)
	}
}
