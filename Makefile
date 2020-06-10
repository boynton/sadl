_test:: bin/sadl
	./bin/sadl -namespace example -gen smithy _test.graphql
#	./bin/sadl -gen smithy _test.sadl

all:: bin/sadl

install:: all
	cp -p bin/* $(HOME)/bin/

lib::
	go build github.com/boynton/sadl/util
	go build github.com/boynton/sadl
	go build github.com/boynton/sadl/io
	go build github.com/boynton/sadl/smithy
	go build github.com/boynton/sadl/openapi
	go build github.com/boynton/sadl/java
	go build github.com/boynton/sadl/golang
#	go build github.com/boynton/sadl/graphql

test::
	go test github.com/boynton/sadl/test
	go test github.com/boynton/sadl/io
	go test github.com/boynton/sadl/openapi

bin/sadl:: 
	mkdir -p bin
	go build -o bin/sadl github.com/boynton/sadl/cmd/sadl

proper::
	go fmt github.com/boynton/sadl
	go vet github.com/boynton/sadl
	go fmt github.com/boynton/sadl/test
	go vet github.com/boynton/sadl/test
	go fmt github.com/boynton/sadl/io
	go vet github.com/boynton/sadl/io
	go fmt github.com/boynton/sadl/util
	go vet github.com/boynton/sadl/util
	go fmt github.com/boynton/sadl/golang
	go vet github.com/boynton/sadl/golang
	go fmt github.com/boynton/sadl/java
	go vet github.com/boynton/sadl/java
	go fmt github.com/boynton/sadl/openapi
	go vet github.com/boynton/sadl/openapi
	go vet github.com/boynton/sadl/smithy
	go fmt github.com/boynton/sadl/cmd/sadl
	go vet github.com/boynton/sadl/cmd/sadl
#	go fmt github.com/boynton/sadl/cmd/sadl2java
#	go vet github.com/boynton/sadl/cmd/sadl2java
#	go fmt github.com/boynton/sadl/cmd/sadl2go
#	go vet github.com/boynton/sadl/cmd/sadl2go
#	go fmt github.com/boynton/sadl/cmd/oas2sadl
#	go vet github.com/boynton/sadl/cmd/oas2sadl
#	go fmt github.com/boynton/sadl/cmd/sadl2oas
#	go vet github.com/boynton/sadl/cmd/sadl2oas
#	go vet github.com/boynton/sadl/cmd/sadl2smithy
#	go vet github.com/boynton/sadl/cmd/smithy2sadl
#	go vet github.com/boynton/sadl/cmd/sadl2html

clean::
	rm -rf bin
