# SpotHero Tools Library for Go

[![GoDoc](https://godoc.org/github.com/spothero/tools?status.svg)](https://godoc.org/github.com/spothero/tools)
[![Build Status](https://circleci.com/gh/spothero/tools/tree/master.svg?style=shield)](https://circleci.com/gh/spothero/tools/tree/master)
[![codecov](https://codecov.io/gh/spothero/tools/branch/master/graph/badge.svg)](https://codecov.io/gh/spothero/tools)
[![Go Report Card](https://goreportcard.com/badge/github.com/spothero/tools)](https://goreportcard.com/report/github.com/spothero/tools)

The SpotHero Tools Library is used internally at SpotHero across our Go programs. This library is a
collection of common utilities and functions that we use to ensure common functionality and best
practices within our organization.

We welcome community usage and collaboration.

#### Running the Example Server

1. Install Golang
    1. `brew install golang`
    2. [Set your `GOPATH` in your `.zshrc`/`.bashrc`/etc](https://github.com/golang/go/wiki/SettingGOPATH)
    3. Add `GOPATH/bin` to your `PATH`
        1. `export PATH=$GOPATH/bin:$PATH`
2. Clone this repository
3. `make`
4. `./example_server`
5. Open your browser to `http://localhost:8080`

#### Overview
This library contains common modules for use in all GoLang projects across SpotHero. To use this
library simply import one of the packages in this library into your project and go modules will
handle the rest.

Because this library is still under active development and is not yet 1.0, please
expect that we may occasionally make backwards-incompatible changes.

Currently, this library supports the following features:

* CLI Utilities
* Kafka
  * Support for consuming and producing metrics
  * Support for goroutine-based callback functions where types are automatically deduced and
    unpacked
  * Schema Registry
* Postgres
  * CLI Support for SQL database configuration
* Avro Decoding
* HTTP Server with instrumentation
* Prometheus Metrics
* High-Performance Logging
* Sentry Integration
* OpenTracing/Jaeger Support

In addition, all the above packages may automatically be integrated with Cobra/Viper CLIs for
12-factor application compatibility via the CLI module.

### Usage

A simple example is provided under [examples/example_server.go](examples/example_server.go) which shows usage of this
library to create a simple 12-factor Go Web application which has tracing, logging, metrics,
sentry, and local caching enabled.

For production applications, we recommend separating the Cobra/Viper command portion into its own
`cmd/` directory, and your application logic into a `pkg/` directory as is standard with most Go
applications.

Additionally, the [Makefile](Makefile) for this project serves as an example which you can
borrow for your own projects.

### License
Apache 2.0
