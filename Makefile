VERSION ?= $(shell git describe --abbrev=0 --tags | sed 's/v//g')
GIT_SHA ?= $(shell git rev-parse HEAD)
LINTER_INSTALLED := $(shell sh -c 'which golangci-lint')

.PHONY: all
all: lint test ## Lints and runs the tests

.PHONY: test
test: ## Runs application tests
	go test -race -v ./... -coverprofile=coverage.txt -covermode=atomic

.PHONY: coverage
coverage: test ## Displays test coverage report
	go tool cover -html=coverage.txt

.PHONY: lint
lint: ## Runs the go code linter
ifdef LINTER_INSTALLED
	golangci-lint run --timeout 2m0s
else
	$(error golangci-lint not found, skipping linting. Installation instructions: https://github.com/golangci/golangci-lint#ci-installation)
endif

.PHONY: help
help: ## Prints this help command
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) |\
		sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: format
format: ## formats the codebase
	gofmt -s -w .
