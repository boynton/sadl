imp:: bin/oas2sadl
#	./bin/oas2sadl $(HOME)/api/AMDSAPISwaggerModels/swagger-models/listings.json
#	./bin/oas2sadl $(HOME)/src/golang.a2z.com/GoAmzn-FbaApiTools/gen2.json
	./bin/oas2sadl $(HOME)/src/golang.a2z.com/GoAmzn-FbaApiTools/examples/beer-openapi-v3.json

doc:: bin/sadl2html
	./bin/sadl2html _httpexample.sadl > index2.html
#	./bin/sadl2html _httpexample.sadl

all:: bin/sadl bin/sadl2java bin/sadl2go bin/oas2sadl bin/sadl2oas bin/sadl2smithy

install:: all
	cp -p bin/* $(HOME)/bin/

lib::
	go build github.com/boynton/sadl
	go build github.com/boynton/sadl/golang
	go build github.com/boynton/sadl/java
	go build github.com/boynton/sadl/oas

test::
	go test github.com/boynton/sadl
	go test github.com/boynton/sadl/oas

bin/sadl:: lib
	mkdir -p bin
	go build -o bin/sadl github.com/boynton/sadl/cmd/sadl

bin/sadl2java:: lib
	mkdir -p bin
	go build -o bin/sadl2java github.com/boynton/sadl/cmd/sadl2java

bin/sadl2go:: lib
	mkdir -p bin
	go build -o bin/sadl2go github.com/boynton/sadl/cmd/sadl2go

bin/oas2sadl:: lib
	mkdir -p bin
	go build -o bin/oas2sadl github.com/boynton/sadl/cmd/oas2sadl

bin/sadl2oas:: lib
	mkdir -p bin
	go build -o bin/sadl2oas github.com/boynton/sadl/cmd/sadl2oas

bin/sadl2smithy:: lib
	mkdir -p bin
	go build -o bin/sadl2smithy github.com/boynton/sadl/cmd/sadl2smithy

bin/sadl2html:: lib cmd/sadl2html/main.go
	mkdir -p bin
	go build -o bin/sadl2html github.com/boynton/sadl/cmd/sadl2html

proper::
	go fmt github.com/boynton/sadl
	go vet github.com/boynton/sadl
	go fmt github.com/boynton/sadl/golang
	go vet github.com/boynton/sadl/golang
	go fmt github.com/boynton/sadl/java
	go vet github.com/boynton/sadl/java
	go fmt github.com/boynton/sadl/oas
	go vet github.com/boynton/sadl/oas
	go fmt github.com/boynton/sadl/oas/oas2
	go vet github.com/boynton/sadl/oas/oas2
	go fmt github.com/boynton/sadl/oas/oas3
	go vet github.com/boynton/sadl/oas/oas3
	go vet github.com/boynton/sadl/smithy
	go fmt github.com/boynton/sadl/cmd/sadl
	go vet github.com/boynton/sadl/cmd/sadl
	go fmt github.com/boynton/sadl/cmd/sadl2java
	go vet github.com/boynton/sadl/cmd/sadl2java
	go fmt github.com/boynton/sadl/cmd/sadl2go
	go vet github.com/boynton/sadl/cmd/sadl2go
	go fmt github.com/boynton/sadl/cmd/oas2sadl
	go vet github.com/boynton/sadl/cmd/oas2sadl
	go fmt github.com/boynton/sadl/cmd/sadl2oas
	go vet github.com/boynton/sadl/cmd/sadl2oas
	go vet github.com/boynton/sadl/cmd/sadl2smithy

clean::
	rm -rf bin
