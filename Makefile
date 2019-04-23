.PHONY: default_target all build test coverage lint

VERSION ?= $(shell git describe --abbrev=0 --tags | sed 's/v//g')
GIT_SHA ?= $(shell git rev-parse HEAD)
LINTER_INSTALLED := $(shell sh -c 'which golangci-lint')

default_target: all

all: lint test

tidy:
	go mod tidy

build: tidy
	go build -ldflags="-X main.version=${VERSION} -X main.gitSHA=${GIT_SHA}" examples/example_server.go

test: tidy
	go test -race -v ./... -coverprofile=coverage.txt -covermode=atomic

coverage: test
	go tool cover -html=coverage.txt

lint:
ifdef LINTER_INSTALLED
	golangci-lint run
else
	$(error golangci-lint not found, skipping linting. Installation instructions: https://github.com/golangci/golangci-lint#ci-installation)
endif
