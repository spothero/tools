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
DEP=$(BIN_DIR)/dep
GOMETALINTER=$(BIN_DIR)/gometalinter

$(GOMETALINTER):
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install &> /dev/null

$(DEP):
	go get -u github.com/golang/dep/cmd/dep

bootstrap: $(GOMETALINTER) $(DEP) ## Pulls requirements for building and pulling dependencies

vendor:
	dep ensure -v -vendor-only

test:
	go test -race -v ./... -coverprofile=coverage.txt -covermode=atomic

coverage: test
	go tool cover -html=coverage.txt

clean:
	rm -rf vendor

# Linting
LINTERS=gofmt golint staticcheck vet misspell ineffassign deadcode
METALINT=gometalinter --tests --disable-all --vendor --deadline=5m -e "zz_.*\.go" ./...

lint: ## Lints the code to ensure it meets coding standards
	$(METALINT) $(addprefix --enable=,$(LINTERS))

$(LINTERS):
	$(METALINT) --enable=$@

build:
	go build -ldflags="-X main.version=${VERSION} -X main.gitSha=${GIT_SHA}" examples/example_server.go
