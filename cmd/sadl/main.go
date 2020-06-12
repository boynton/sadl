package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/boynton/cli"
	//	"github.com/boynton/sadl"
)

/*
To parse/load/import a supported spec format and invoke the default generator:
$ sadl foo.sadl
$ sadl foo.smithy
$ sadl foo.graphql -
$ sadl foo.json -> figures out the correct format to import (openapi, smithy ast, sadl json)

The default generator outputs formatted SADL source. Output is to stdout by default. The output directory can be specified:
$ sadl -dir . foo.sadl -> creates a
Other generators include:
$ sadl -g json foo.smithy -> outputs the pretty-printed JSON representation of the SADL model
$ sadl -g java-server foo.sadl -> generats
*/

func main() {
	helpMessage := `Supported API description formats for each input file extension:
   .sadl     sadl
   .smithy   smithy
   .graphql  graphql
   .json     sadl, smithy, openapi (inferred by looking at the file contents)
   .yaml     openapi

The 'name' and 'namespace' options allow specifying those attributes for input formats
that do not require or support them. Otherwise a default is used.

The 'conf' option is the name of a JSON config file. Code generators generally use the
'dir' option, conversions to other API description formats just output to stdout.

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
	var dir string
	var name string
	var ns string
	var gen string
	var configPath string
	cmd := cli.New("sadl", helpMessage)
	cmd.StringOption(&dir, "dir", ".", "The output directory for generators.")
	cmd.StringOption(&name, "name", "", "The name of the model, overrides any name present in the source")
	cmd.StringOption(&ns, "ns", "", "The namespace of the model, overrides any namespace present in the source")
	cmd.StringOption(&gen, "gen", "sadl", "The default generator for output.")
	cmd.StringOption(&configPath, "conf", "", "The JSON config file to use to configure the generator")
	args, _ := cmd.Parse()
	if len(args) == 0 {
		fmt.Println(cmd.Usage())
		os.Exit(0)
	}
	path := args[0]
	importConf := make(map[string]interface{}, 0)
	if ns != "" {
		importConf["namespace"] = ns
	}
	if name != "" {
		importConf["name"] = name
	}
	model, err := ImportFile(path, importConf)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	var conf map[string]interface{}
	if configPath != "" {
		b, err := ioutil.ReadFile(configPath)
		if err != nil {
			fmt.Printf("Cannot read config file %q: %v\n", configPath, err)
			os.Exit(3)
		}
		err = json.Unmarshal(b, &conf)
		if err != nil {
			fmt.Printf("Cannot parse config file %q: %v\n", configPath, err)
			os.Exit(3)
		}
	}
	err = ExportFiles(model, gen, dir, conf)
	if err != nil {
		fmt.Printf("*** %v\n", err)
		os.Exit(4)
	}
}
