.PHONY: fmt tidy test test-race vet build verify

GO_FILES := $(shell find cmd internal -name '*.go')

fmt:
	gofmt -w $(GO_FILES)

tidy:
	go mod tidy

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

build:
	go build -o bin/care-service ./cmd/server

verify: fmt tidy test vet build

