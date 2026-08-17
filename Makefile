.PHONY: test build

test:
	go test ./...

build:
	go build -o terraform-provider-homecloud .
