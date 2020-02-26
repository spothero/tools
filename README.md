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
* OpenTracing/Jaeger Support
* JSON Web Token (JWT) and Javascript Object Signing and Encryption (JOSE) Support
* Sentry Integration

In addition, all the above packages may automatically be integrated with Cobra/Viper CLIs for
12-factor application compatibility via the CLI module.

### Usage

The packages in this library may be used together or independently. At SpotHero, our applications
all start from the Service `ServerCmd` which contains a "best-practice" configuration of a SpotHero
web server. This web-server includes a GRPC and HTTP Server, as well as full instrumentation with
tools such as Prometheus, Jaeger/OpenTracing, Sentry, and so on.

A simple example is provided under [service/example_test.go](service/example_test.go) which shows usage of this
library to create a simple 12-factor Go Web application which has tracing, logging, metrics,
sentry, and local caching enabled.

For production applications, we recommend separating the Cobra/Viper command portion into its own
`cmd/` directory, and your application logic into a `pkg/` directory as is standard with most Go
applications.

### License

Apache 2.0
