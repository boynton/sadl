all bin/sadl: go.mod
	mkdir -p bin
	go build -o bin/sadl github.com/boynton/sadl/cmd/sadl

install:: all
	cp -p bin/sadl $(HOME)/bin/sadl

test::
	go test github.com/boynton/sadl/test
	go test github.com/boynton/sadl/openapi

proper::
	go fmt github.com/boynton/sadl
	go vet github.com/boynton/sadl
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
	rm -rf bin sadl_darwin_amd64.zip sadl_darwin_arm64.zip

release:: sadl_darwin_amd64.zip sadl_darwin_arm64.zip

go.mod:
	go mod init github.com/boynton/sadl && go mod tidy

sadl_darwin_amd64.zip::
	rm -rf darwin_amd64
	mkdir -p darwin_amd64
	env GOOS=darwin GOARCH=amd64 go build -o darwin_amd64/sadl github.com/boynton/sadl/cmd/sadl
	(cd darwin_amd64; zip -rp ../sadl_darwin_amd64.zip sadl)
	rm -rf darwin_amd64

sadl_darwin_arm64.zip::
	rm -rf darwin_arm64
	mkdir -p darwin_arm64
	env GOOS=darwin GOARCH=arm64 go build -o darwin_arm64/sadl github.com/boynton/sadl/cmd/sadl
	(cd darwin_arm64; zip -rp ../sadl_darwin_arm64.zip sadl)
	rm -rf darwin_arm64

