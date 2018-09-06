VERSION_MAJOR ?= local
VERSION_MINOR ?= local
VERSION_PATCH ?= local
VERSION ?= ${VERSION_MAJOR}.${VERSION_MINOR}.${VERSION_PATCH}
GIT_SHA ?= $(shell git rev-parse HEAD)

default_target: all

all: bootstrap vendor test vendor

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
	go test -v ./... -cover

clean:
	rm -rf vendor
