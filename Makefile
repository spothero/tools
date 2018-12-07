VERSION_MAJOR ?= local
VERSION_MINOR ?= local
VERSION_PATCH ?= local
VERSION ?= ${VERSION_MAJOR}.${VERSION_MINOR}.${VERSION_PATCH}
GIT_SHA ?= $(shell git rev-parse HEAD)

default_target: all

all: bootstrap vendor test build

# Bootstrapping for base golang package deps
BOOTSTRAP=\
	github.com/golang/dep/cmd/dep \
	github.com/alecthomas/gometalinter

$(BOOTSTRAP):
	go get -u $@

bootstrap: $(BOOTSTRAP)
	gometalinter --install

vendor:
	dep ensure -v -vendor-only

test:
	go test -race -v ./... -coverprofile=coverage.txt -covermode=atomic

coverage: test
	go tool cover -html=coverage.txt

clean:
	rm -rf vendor

build:
	go build -ldflags="-X main.version=${VERSION} -X main.gitSha=${GIT_SHA}" examples/example_server.go
