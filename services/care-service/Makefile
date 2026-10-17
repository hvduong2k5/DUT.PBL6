.PHONY: fmt tidy test test-integration test-race vet build verify

GO_FILES := $(shell find cmd internal -name '*.go')

fmt:
	gofmt -w $(GO_FILES)

tidy:
	go mod tidy

test:
	go test ./...

test-integration:
	CARE_TEST_DATABASE_URL="postgres://care_user:care_password@localhost:5438/care_db?sslmode=disable" go test -count=1 -run '^TestIntegration' ./migrations ./internal/repository/postgres

test-race:
	go test -race ./...

vet:
	go vet ./...

build:
	go build -o bin/care-service ./cmd/server
	go build -o bin/care-migrate ./cmd/migrate

verify: fmt tidy test vet build

