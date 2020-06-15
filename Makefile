_debug:: bin/sadl
#	./bin/sadl ../sadl_dev/_test.graphql > ../sadl_dev/_test.sadl
#	./bin/sadl --gen graphql $(HOME)/rigs/smithy/model/common.smithy
#	./bin/sadl --gen smithy ../sadl_dev/_test.sadl > ../sadl_dev/_test.smithy
#	./bin/sadl ../sadl_dev/_test.smithy
	./bin/sadl --gen graphql ../sadl_dev/_test.smithy


all bin/sadl:: 
	mkdir -p bin
	go build -o bin/sadl github.com/boynton/sadl/cmd/sadl

install:: all
	cp -p bin/* $(HOME)/bin/

test::
	go test github.com/boynton/sadl/test
	go test github.com/boynton/sadl/io
	go test github.com/boynton/sadl/openapi

proper::
	go fmt github.com/boynton/sadl
	go vet github.com/boynton/sadl
	go fmt github.com/boynton/sadl/util
	go vet github.com/boynton/sadl/util
	go fmt github.com/boynton/sadl/io
	go vet github.com/boynton/sadl/io
	go fmt github.com/boynton/sadl/test
	go vet github.com/boynton/sadl/test
	go fmt github.com/boynton/sadl/golang
	go vet github.com/boynton/sadl/golang
	go fmt github.com/boynton/sadl/graphql
	go vet github.com/boynton/sadl/graphql
	go fmt github.com/boynton/sadl/java
	go vet github.com/boynton/sadl/java
	go fmt github.com/boynton/sadl/openapi
	go vet github.com/boynton/sadl/openapi
	go fmt github.com/boynton/sadl/smithy
	go vet github.com/boynton/sadl/smithy
	go fmt github.com/boynton/sadl/cmd/sadl
	go vet github.com/boynton/sadl/cmd/sadl

clean::
	rm -rf bin
