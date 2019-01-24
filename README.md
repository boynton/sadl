# SADL - Simple API Description Language

SADL is a general high level API description language that defines its own schema language as well as operation and resource
descriptions. It's goal is to provide a means to express a straightforward, consistent view of APIs.

It is an independent derivitive of [RDL](https://github.com/ardielle), and shares the syntax for type definitions, although
the operation and resource descriptions are different. It is not intended to be compatible, but rather simpler.

SADL can be used to generate other API definitions, for example OpenAPI, RAML, gRPC, and RDL, as well as generate client
and server code directly in a few languages for quick iterative API investigations.

## Usage

    go get github.com/boynton/sadl
    sadl foo.sadl # just parses and shows errors, if any.

    sadl jaxrs-server foo.sadl # generates a working java server based on jersey/jackson/jetty
    sadl gen jaxrs-client foo.sadl # generates a working java client for the above server
    sadl gen go-server foo.sadl # generates a go server

##





