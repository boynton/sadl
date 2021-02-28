# SADL - Simple API Description Language

SADL is a general high level API description language that defines its own schema language as well as operation and resource
descriptions, optimized for simplicity and speed.

SADL can convert between various API description formats, and also generate code.

## Install

On a Mac, use homebrew:

    $ brew tap boynton/tap
    $ brew install sadl
    
The current executables are available as assets in the current [GitHub Release](https://github.com/boynton/sadl/releases/latest).

To install from source, clone this repo and type "make". The build requires [Go](https://golang.org).

## Usage

Invoked with no arguments, `sadl` shows basic usage:

```
$ sadl
Usage: sadl [options] file ...

Options:
  -c string
    	The JSON config file for default settings. Default is $HOME/.sadl-config.yaml. See below for format.
  -f	Force overwrite of existing files
  -g string
    	The generator for output (default "sadl")
  -h	Show more helpful information
  -n string
    	The name of the model, overrides any name present in the source
  -ns string
    	The namespace of the model, overrides any namespace present in the source
  -o string
    	The output file or directory. (default "/tmp/generated")
  -s string
    	The single service to consider in the model. Default is to use the only one present.
  -t string
    	Only read files of this type. By default, any valid input file type is accepted.
  -v	Show SADL version and exit
  -x value
    	An option to pass to the generator
```

In general, it takes an arbitrary input file, parses it, and outputs with a generator which defaults to SADL itself. SADL
is fairly concise, so it is useful to verify that other formats parse correctly. SADL does not support all features of other
formats, just a reasonable common subset.

```
$ cat examples/hello.sadl
name hello
namespace examples

//
// A minimal hello world action
//
http GET "/hello?caller={caller}" (action=hello) {
	caller String (default="Mystery Person")

	expect 200 {
		greeting String
	}
}

//An example of the Hello operation
example HelloRequest (name=helloExample) {
	"caller": "Lee"
}
example HelloResponse (name=helloExample) {
	"greeting": "Hello, Lee"
}
```

To parse and echo the result (which is equivalent to the source):
```
namespace examples
name hello

//
// A minimal hello world action
//
action hello GET "/hello?caller={caller}" {
    caller String (default="Mystery Person")

    expect 200 {
        greeting String (required)
   }
}

//
// An example of the Hello operation
//
example HelloRequest (name=helloExample) {
  "caller": "Lee"
}


example HelloResponse (name=helloExample) {
  "greeting": "Hello, Lee"
}

```

To show SADL's data representation in JSON:
```
$ sadl -g json examples/hello.sadl
{
  "sadl": "1.6.2",
  "name": "hello",
  "namespace": "examples",
  "examples": [
    {
      "target": "HelloRequest",
      "name": "helloExample",
      "example": {
        "caller": "Lee"
      },
      "comment": "An example of the Hello operation"
    },
    {
      "target": "HelloResponse",
      "name": "helloExample",
      "example": {
        "greeting": "Hello, Lee"
      }
    }
  ],
  "http": [
    {
      "name": "hello",
      "comment": "A minimal hello world action",
      "method": "GET",
      "path": "/hello?caller={caller}",
      "inputs": [
        {
          "query": "caller",
          "name": "caller",
          "default": "Mystery Person",
          "type": "String"
        }
      ],
      "expected": {
        "outputs": [
          {
            "name": "greeting",
            "required": true,
            "type": "String"
          }
        ],
        "status": 200
      }
    }
  ]
}
```

To convert this to [Smithy](https://awslabs.github.io/smithy/):
```
$ sadl -g smithy examples/hello.sadl

namespace examples

service hello {
    version: "0.0",
    operations: [Hello]
}

///
/// A minimal hello world action
///
@http(method: "GET", uri: "/hello", code: 200)
@readonly
operation Hello {
    input: HelloInput,
    output: HelloOutput,
}

structure HelloOutput {
  @httpPayload
  greeting: String,
}

structure HelloInput {
  @httpQuery("caller")
  caller: String,
}


apply Hello @examples([
  {
    "title": "helloExample",
    "documentation": "An example of the Hello operation",
    "input": {
      "caller": "Lee"
    },
    "output": {
      "greeting": "Hello, Lee"
    }
  }
])
```

To parse the smithy back into sadl:

```
$ sadl /tmp/hello.smithy
namespace examples
name hello

//
// A minimal hello world action
//
action hello GET "/hello?caller={caller}" {
    caller String

    expect 200 {
        greeting String
   }
}

//
// An example of the Hello operation
//
example HelloRequest (name=helloExample) {
  "caller": "Lee"
}


example HelloResponse (name=helloExample) {
  "greeting": "Hello, Lee"
}

```
Note that Smithy doesn't support default values, so the transformation from SADL to Smithy and back is lossy.

The complete list of supported formats and generators is access with the help flag:

```
sadl -h


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
```

## Configuration File

Generator options as noted above can be specified with the `-x` command line option:

    $ sadl -g java -x lombok-true examples/crudl.sadl

Some generators have enough options that specifying them on the command line is tedious. These can be placed into 
a global configuration file. Each generator's settings  arein an entry matching the generator's name. For example:

```
java:
   domain: "boynton.com"
   lombok: true
graphql:
   custom-scalars:
      UUID: UUID
      Timestamp: Timestamp
      Decimal: Decimal
      Int64: Long
```

## Examples

See some examples in the [examples](https://github.com/boynton/sadl/tree/master/examples) directory. Or
take a file in a known format and just parse it to output the SADL representation of it to get a better feel
of how SADL represents things.

See also two example implementations of the `crudl` example, in Java and Go, in the following two repos:
- https://github.com/boynton/java-crudl
- https://github.com/boynton/go-crudl


## Base Types

SADL is built on the following base types:

- Bool - Either `true` or `false`
- Int8 - an 8 bit signed integer
- Int16 - a 16 bit signed integer
- Int32 - a 32 bit signed integer
- Int64 - a 64 bit signed integer
- Float32 - single precision IEEE 754 floating point number
- Float64 - double precision IEEE 754 floating point number
- Decimal - An arbitrary precision decimal number. Represented as a string in JSON to avoid implementation-specific precision issues (i.e. "3.141592653589793238462643383279502884197169399375105819")
- Bytes - a sequence of 8 bit bytes
- String - A sequence of Unicode characters.
- Timestamp - An instant in time, formatted as string per [RFC 3339](http://tools.ietf.org/html/rfc3339) in JSON (i.e. "2019-02-04T01:05:16.565Z")
- UnitValue<Decimal,String> - A tuple of numeric value and String or Enum units the value is measured in. Expressed as a string in JSON (i.e. "100.00 USD")
- UUID - a Universally Unique Identifier [RFC 4122](http://tools.ietf.org/html/rfc4122), represented as a string in JSON (i.e. "1ce437b0-1dd2-11b2-81ef-003ee1be85f9")
- Array<Any> - an ordered collections of values
- Map<String,Any> - an unordered mapping of keys to values type.
- Struct - an ordered collection of named fields, each with its own type.
- Enum - a set of symbols
- Union<typename,...> - a tagged union of types. Expressed as a JSON object with optional keys for each variant.
- Any - any of the above types

## Notes

SADL is inspired by [RDL](https://github.com/ardielle), but is not compatible with it.

SADL is designed for prototyping and experimentation.
