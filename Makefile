.PHONY: default_target all build test coverage lint help

VERSION ?= $(shell git describe --abbrev=0 --tags | sed 's/v//g')
GIT_SHA ?= $(shell git rev-parse HEAD)
LINTER_INSTALLED := $(shell sh -c 'which golangci-lint')

all: lint test

build: ## Builds application artifacts
	go build -ldflags="-X main.version=${VERSION} -X main.gitSHA=${GIT_SHA}" examples/example_server.go

test: ## Runs application tests
	go test -race -v ./... -coverprofile=coverage.txt -covermode=atomic

coverage: test ## Displays test coverage report
	go tool cover -html=coverage.txt

lint: ## Runs the go code linter
ifdef LINTER_INSTALLED
	golangci-lint run
else
	$(error golangci-lint not found, skipping linting. Installation instructions: https://github.com/golangci/golangci-lint#ci-installation)
endif

help: ## Prints this help command
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) |\
		sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

