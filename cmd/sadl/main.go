package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/boynton/sadl"
)

type ArrayOption []string

func (ao *ArrayOption) String() string {
	return "An array of options"
}

func (ao *ArrayOption) Set(value string) error {
	*ao = append(*ao, value)
	return nil
}

func main() {
	helpMessage := `

Supported API description formats for each input file extension:
   .sadl     sadl
   .smithy   smithy
   .graphql  graphql
   .json     sadl, smithy, openapi (inferred by looking at the file contents)
   .yaml     openapi

The 'name' and 'namespace' options allow specifying those attributes for input formats
that do not require or support them. Otherwise a default is used based on the model being parsed.

The '-c' option specifies the name of a YAML config file, the default of which is
$HOME/.sadl-config.yaml. Some code generators use the '-o' option to specify the
output directory, conversions to other API description formats just output to stdout.

Supported generators and options used from config if present
   sadl: Prints the SADL representation to stdout. This is the default.
   json: Prints the parsed SADL data representation in JSON to stdout
   smithy: Prints the Smithy IDL representation to stdout. Options:
      name: supply this value as the name for a service for inputs that do not have a name
      namespace: supply this value as the namespace for inputs that do not have a namespace
   smithy-ast: Prints the Smithy AST representation to stdout, same options as 'smithy'
   openapi: Prints the OpenAPI Spec v3 representation to stdout
   graphql: Prints the GraphQL representation to stdout. Options:
      custom-scalars: a map of any of ["Int64", "Decimal", "Timestamp", "UUID"] to a custom scalar name.
   java: Generate Java code for the model, server, client plumbing. Options:
      header: a string to include at the top of every generated java file
      lombok: use Lombok for generated model POJOs to reduce boilerplate, default is false
      getters: create model POJOs with traditional getter/setter style, default is true
      immutable: Create POJOs as immutable with a builder static inner class, default is true
      source: specify the default source directory, default to "src/main/java"
      resource: specify the default resouece directory, default to "src/main/resource"
      server: include server plumbing code, using Jersey for JAX-RS implementation.
      client: include client plumbing code, using Jersey for the implementation.
      project: "maven" generates a pom.xml file to build the project, others (i.e. gradle) will be added
      domain: The domain name for the project, for use in things like the maven pom.xml file.
      instants: use java.time.Instant for Timestamp impl, else generate a Timestamp class.
   java-server: a shorthand for specifying the "server" option to the "java" generator. Same options.
   java-client: a shorthand for specifying the "client" option to the "java" generator. Same options.
   go: Generate Go code for the model
      header: a string to include at the top of every generated java file
      server: include server plumbing code, using Gorilla for HTTP router implementation.
      client: include client plumbing code.
   go-server: a shorthand for specifying the "server" option to the "go" generator. Same options.
   go-client: a shorthand for specifying the "client" option to the "go" generator. Same options.
   http-trace: Generates an HTTP (curl-style) simulation of the API's example HTTP actions, based on examples in the model

`
	var genOpts ArrayOption
	pType := flag.String("t", "", "Only read files of this type. By default, any valid input file type is accepted.")
	pOut := flag.String("o", "/tmp/generated", "The output file or directory.")
	pName := flag.String("n", "", "The name of the model, overrides any name present in the source")
	pNamespace := flag.String("ns", "", "The namespace of the model, overrides any namespace present in the source")
	pService := flag.String("s", "", "The single service to consider in the model. Default is to use the only one present.")
	pGen := flag.String("g", "sadl", "The generator for output")
	pConf := flag.String("c", "", "The JSON config file for default settings. Default is $HOME/.sadl-config.yaml")
	pForce := flag.Bool("f", false, "Force overwrite of existing files")
	flag.Var(&genOpts, "x", "An option to pass to the generator")
	pVersion := flag.Bool("v", false, "Show SADL version and exit")
	pHelp := flag.Bool("h", false, "Show more helpful information")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: sadl [options] file ...\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *pHelp {
		fmt.Fprintf(os.Stderr, helpMessage)
		os.Exit(1)
	}
	if *pVersion {
		fmt.Printf("SADL %s\n", sadl.Version)
		os.Exit(0)
	}
	args := flag.Args()
	out := *pOut
	name := *pName
	namespace := *pNamespace
	gen := *pGen
	configPath := *pConf
	force := *pForce
	formatType := *pType
	service := *pService

	if len(args) == 0 {
		flag.Usage()
		os.Exit(0)
	}
	importConf := sadl.NewData()
	if namespace != "" {
		importConf.Put("namespace", namespace)
	}
	if service != "" {
		importConf.Put("service", service)
	}
	if name != "" {
		importConf.Put("name", name)
	}
	if formatType != "" {
		importConf.Put("type", formatType)
	}
	model, err := ImportFiles(args, importConf)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	var conf *sadl.Data
	if configPath == "" {
		configPath = os.Getenv("HOME") + "/.sadl-config.yaml"
		if !fileExists(configPath) {
			configPath = ""
		}
	}
	if configPath != "" {
		conf, err = sadl.DataFromFile(configPath)
		if err != nil {
			fmt.Printf("Cannot read config file %q: %v\n", configPath, err)
		}
	} else {
		conf = sadl.NewData()
	}
	gc := gen
	if gen == "java-server" { //todo: get rid of this hack
		gc = "java"
	}
	genConf := conf.GetData(gc)
	genConf.Put("force-overwrite", force)
	for _, kv := range genOpts {
		k := kv
		var v interface{}
		v = true
		kvs := strings.Split(kv, "=")
		if len(kvs) > 1 {
			k = kvs[0]
			v = kvs[1]
		}
		genConf.Put(k, v)
	}
	err = ExportFiles(model, gen, out, genConf)
	if err != nil {
		fmt.Printf("*** %v\n", err)
		os.Exit(4)
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
