debug: bin/oas2sadl
	./bin/oas2sadl -d examples/petstore.yaml

dbg: bin/sadl
	./bin/sadl examples/crudl.sadl


all:: bin/sadl bin/sadl2java bin/sadl2go

install:: all
	cp -p bin/* $(HOME)/bin/

lib::
	go build github.com/boynton/sadl
	go build github.com/boynton/sadl/parse

test::
	go test github.com/boynton/sadl
	go test github.com/boynton/sadl/parse

bin/sadl:: lib
	mkdir -p bin
	go build -o bin/sadl github.com/boynton/sadl/cmd/sadl

bin/sadl2java:: lib
	mkdir -p bin
	go build -o bin/sadl2java github.com/boynton/sadl/cmd/sadl2java

bin/sadl2go:: lib
	mkdir -p bin
	go build -o bin/sadl2go github.com/boynton/sadl/cmd/sadl2go

bin/oas2sadl::
	mkdir -p bin
	go build -o bin/oas2sadl github.com/boynton/sadl/cmd/oas2sadl

proper::
	go fmt github.com/boynton/sadl
	go vet github.com/boynton/sadl
	go fmt github.com/boynton/sadl/parse
	go vet github.com/boynton/sadl/parse
	go fmt github.com/boynton/sadl/cmd/sadl
	go vet github.com/boynton/sadl/cmd/sadl
	go fmt github.com/boynton/sadl/cmd/sadl2java
	go vet github.com/boynton/sadl/cmd/sadl2java
	go fmt github.com/boynton/sadl/cmd/oas2sadl
	go vet github.com/boynton/sadl/cmd/oas2sadl

clean::
	rm -rf bin
