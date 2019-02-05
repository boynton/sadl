all:: bin/sadl

test1:
#	go test -v github.com/boynton/sadl
	go test -v github.com/boynton/sadl/parse

test::
	go test github.com/boynton/sadl
	go test github.com/boynton/sadl/parse

bin/sadl::
	mkdir -p bin
	go build github.com/boynton/sadl
	go build github.com/boynton/sadl/parse
	go build -o bin/sadl github.com/boynton/sadl/cmd/sadl

proper::
	go fmt github.com/boynton/sadl
	go vet github.com/boynton/sadl
	go fmt github.com/boynton/sadl/parse
	go vet github.com/boynton/sadl/parse
	go fmt github.com/boynton/sadl/cmd/sadl
	go vet github.com/boynton/sadl/cmd/sadl

clean::
	rm -rf bin
