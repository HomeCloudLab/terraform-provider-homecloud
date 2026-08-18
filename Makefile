.PHONY: test build release-snapshot

BIN := terraform-provider-homecloud
ifeq ($(OS),Windows_NT)
BIN := terraform-provider-homecloud.exe
endif

test:
	go test ./...

build:
	go build -o $(BIN) .

release-snapshot:
	goreleaser release --snapshot --clean --skip=sign
