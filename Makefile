debug:: bin/sadl2go
	./bin/sadl2go -r -o example _examples/test1.sadl
	cat example/test1_model.go

all:: bin/sadl bin/sadl2java bin/sadl2go

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

proper::
	go fmt github.com/boynton/sadl
	go vet github.com/boynton/sadl
	go fmt github.com/boynton/sadl/parse
	go vet github.com/boynton/sadl/parse
	go fmt github.com/boynton/sadl/cmd/sadl
	go vet github.com/boynton/sadl/cmd/sadl

clean::
	rm -rf bin
