.PHONY: all build build-debug test lint

all: build build-debug test lint

build:
	go build -v ./...

build-debug:
	go build -tags debug ./...

test:
	go test -race -covermode=atomic -coverprofile=coverage.out ./...

lint:
	golangci-lint run
