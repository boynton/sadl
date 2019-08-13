all:: bin/sadl bin/sadl2java bin/sadl2go bin/oas2sadl bin/sadl2oas

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

bin/oas2sadl::
	mkdir -p bin
	go build -o bin/oas2sadl github.com/boynton/sadl/cmd/oas2sadl

bin/sadl2oas::
	mkdir -p bin
	go build -o bin/sadl2oas github.com/boynton/sadl/cmd/sadl2oas

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

clean::
	rm -rf bin
