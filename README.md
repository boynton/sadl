# SADL - Simple API Description Language

SADL is a general high level API description language that defines its own schema language as well as operation and resource
descriptions. It's goal is to provide a means to express a straightforward, consistent view of APIs.

It is an independent derivitive of [RDL](https://github.com/ardielle), and shares the syntax for type definitions, although
the operation and resource descriptions are different. It is not intended to be compatible.

SADL can be used to generate other API definitions, for example OpenAPI, RAML, gRPC, and RDL, as well as generate client
and server code directly in a few languages for quick iterative API investigations.

## Base Types

- Bool - Either `true` or `false`
- Int8 - an 8 bit signed integer
- Int16 - a 16 bit signed integer
- Int32 - a 32 bit signed integer
- Int64 - a 64 bit signed integer
- Float32 - single precision IEEE 754 floating point number
- Float64 - double precision IEEE 754 floating point number
- Decimal - An arbitrary precision decimal number
- Bytes - a sequence of 8 bit bytes
- String - A sequence of Unicode characters.
- Timestamp - An instant in time, formatted as string per [RFC 3339](http://tools.ietf.org/html/rfc3339)
- Quantity - A tuple of Decimal value and units the value is measured in
- UUID - a Universally Unique Identifier [RFC 4122](http://tools.ietf.org/html/rfc4122)
- Array - an ordered collections of values
- Map - an unordered mapping of keys to values type.
- Struct - an ordered collection of named fields, each with its own type
- Enum - a set of symbols
- Union - a tagged union of types
- Any - any of the above types


## Usage

    go get github.com/boynton/sadl/...
    $(GOPATH)/bin/sadl foo.sadl # just parses and shows errors, if any.

    sadl java-model foo.sadl # generates Java model objects for the types in the schema
    sadl java-jaxrs-server foo.sadl # generates a working java server based on jersey/jackson/jetty
    sadl gen java-client foo.sadl # generates a working java client for the above server
    sadl gen go-server foo.sadl # generates a go server
    sadl gen go-client foo.sadl # generates a go server

##





