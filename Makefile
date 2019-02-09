all:: bin/sadl bin/sadl2pojo

lib::
	go build github.com/boynton/sadl
	go build github.com/boynton/sadl/parse

test::
	go test github.com/boynton/sadl
	go test github.com/boynton/sadl/parse

bin/sadl:: lib
	mkdir -p bin
	go build -o bin/sadl github.com/boynton/sadl/cmd/sadl

bin/sadl2pojo:: lib
	mkdir -p bin
	go build -o bin/sadl2pojo github.com/boynton/sadl/cmd/sadl2pojo

proper::
	go fmt github.com/boynton/sadl
	go vet github.com/boynton/sadl
	go fmt github.com/boynton/sadl/parse
	go vet github.com/boynton/sadl/parse
	go fmt github.com/boynton/sadl/cmd/sadl
	go vet github.com/boynton/sadl/cmd/sadl

clean::
	rm -rf bin
