# SpotHero Tools Library for Go

[![GoDoc](https://godoc.org/github.com/spothero/tools?status.svg)](https://godoc.org/github.com/spothero/tools)
[![Build Status](https://circleci.com/gh/spothero/tools/tree/master.svg?style=shield)](https://circleci.com/gh/spothero/tools/tree/master)
[![codecov](https://codecov.io/gh/spothero/tools/branch/master/graph/badge.svg)](https://codecov.io/gh/spothero/tools)
[![Go Report Card](https://goreportcard.com/badge/github.com/spothero/tools)](https://goreportcard.com/report/github.com/spothero/tools)

The SpotHero Tools Library is used internally at SpotHero across our Go programs. This library is a
collection of common utilities and functions that we use to ensure common functionality and best
practices within our organization.

We welcome community usage and collaboration.

#### Getting up and running

1. [Install Golang](https://golang.org/doc/install)
2. Clone this repository
3. `make` to run lints and tests

#### Overview

This library contains common modules for use in all Go projects across SpotHero. To use this
library simply import one of the packages in this library into your project and go modules will
handle the rest.

Because this library is still under active development and is not yet 1.0, please
expect that we may occasionally make backwards-incompatible changes.

Currently, this library supports the following features:

* CLI Utilities
* Kafka
  * Support for consuming and producing metrics
  * Schema Registry
* SQL Bindings for MySQL and Postgres
  * CLI Support for SQL database configuration
* Avro Decoding
* HTTP Server and client tooling with instrumentation
* gRPC Server and client tooling with instrumentation
* Prometheus Metrics
* High-Performance Logging
* OpenTelemetry/Jaeger Support
* JSON Web Token (JWT) and Javascript Object Signing and Encryption (JOSE) Support
* Sentry Integration

In addition, all the above packages may automatically be integrated with Cobra/Viper CLIs for
12-factor application compatibility via the CLI module.

### Usage

The packages in this library may be used together or independently. At SpotHero, our applications
all start from the Service `ServerCmd` which contains a "best-practice" configuration of a SpotHero
web server. This web-server includes a GRPC and HTTP Server, as well as full instrumentation with
tools such as Prometheus, Jaeger/OpenTelemetry, Sentry, and so on.

A simple example is provided under [service/example_test.go](service/example_test.go) which shows usage of this
library to create a simple 12-factor Go Web application which has tracing, logging, metrics,
sentry, and local caching enabled.

For production applications, we recommend separating the Cobra/Viper command portion into its own
`cmd/` directory, and your application logic into a `pkg/` directory as is standard with most Go
applications.

### Linting

Run the linter using the command `make lint`.

A common linting error is the `fieldalignment` warning from the `govet` analyzer. `fieldalignment` errors arise when the order of a struct’s fields could be arranged differently to optimize the amount of allocated memory.

Imagine the following struct:
```
type MyObject struct {
    myBool   bool
    myString string
}
```

Running the linter would produce this output:
```
>> make lint
golangci-lint run

main.go:16:15: fieldalignment: struct with 16 pointer bytes could be 8 (govet)
type MyObject struct {
              ^
make: *** [lint] Error 1
```

The struct is more optimally arranged as:
```
type MyObject struct {
    myString string
    myBool   bool
}
```

A `fieldalignment` command line tool exists to help optimally arrange all the structs in a given file or package. Note that this tool will remove all existing comments within any structs it rearranges. Be sure to manually re-add any deleted comments after running the command.

Installation:
```
go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest
```

Utilization:
```
fieldalignment -fix {PATH_TO_FILE_OR_PACKAGE}
```

### License

Apache 2.0
