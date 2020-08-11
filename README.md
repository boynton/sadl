# SADL - Simple API Description Language

SADL is a general high level API description language that defines its own schema language as well as operation and resource
descriptions, optimized for simplicity and speed.

## Base Types

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

## Example Schemas

TBD. For now, see some examples in the [examples](https://github.com/boynton/sadl/tree/master/examples) directory. Or
take a file in a know format and just parse it to output the SADL representation of it.

## Usage

To get SADL and shoiw basic usage:

    go get github.com/boynton/sadl/...
    $(GOPATH)/bin/sadl

In general, it takes an arbitrary input file, parses it, and outputs with a generator, which defaults to SADL itself.

To just parse, show errors, and output the JSON representation of the resulting model:

    $(GOPATH)/bin/sadl foo.sadl

To generate Java code:

    $(GOPATH)/bin/sadl -g java foo.sadl

There are a variety of options possible to place in the config file (use the -help option to see the list for each generator). An
example config file might look like this:

```
java:
   domain: "boynton.com"

graphql:
   custom-scalars:
      UUID: UUID
      Timestamp: Timestamp
      Decimal: Decimal
      Int64: Long
```


The `pom` option creates a Maven pom.xml file with dependencies to build the resulting code, and the `server` option produces example JAX-RS server code (using
Jersey, Jackson, and Jetty), useful as a quick ready-to-build project creation tool.

To generate Go code:

    $(GOPATH)/bin/sadl2go
    usage: sadl2go -dir outdir -package go_package_name -runtime some_model.sadl

The `runtime` option (which defaults to false) causes the generated code to use this repo's runtime library code for types like Decimal and Timestamp. By default
such code is generated in your package along with the modeled types.

## Notes

SADL is inspired by [RDL](https://github.com/ardielle), but is not compatible with it.

SADL is designed for prototyping and experimentation.








