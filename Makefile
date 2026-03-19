.PHONY: build test lint clean release snapshot install

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS = -ldflags "-X github.com/lucere/gbatch/cmd.Version=$(VERSION)"

## Build

build:
	go build $(LDFLAGS) -o gbatch .

install:
	go install $(LDFLAGS) .

## Test

test:
	go test ./... -v -count=1

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-fuzz:
	go test ./internal/migrate/ -fuzz=FuzzParseDirective -fuzztime=30s

## Quality

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Install: brew install golangci-lint"; exit 1; }
	golangci-lint run ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

check: vet lint test

## Release

snapshot:
	goreleaser release --snapshot --clean

release:
	goreleaser release --clean

## Clean

clean:
	rm -f gbatch coverage.out coverage.html
	rm -rf dist/
