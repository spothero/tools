.PHONY: default_target all build test coverage clean lint

default_target: all

all: lint test

tidy:
	go mod tidy

build: tidy
	go build -ldflags="-X main.version=${VERSION} -X main.gitSha=${GIT_SHA}" examples/example_server.go

test: tidy
	go test -race -v ./... -coverprofile=coverage.txt -covermode=atomic

coverage: test
	go tool cover -html=coverage.txt

clean:
	rm -rf vendor

lint:
ifdef LINTER_INSTALLED
	golangci-lint run
else
	$(error golangci-lint not found, skipping linting. Installation instructions: https://github.com/golangci/golangci-lint#ci-installation)
endif
