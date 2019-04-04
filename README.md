# SpotHero Tools Library for Go

[![GoDoc](https://godoc.org/github.com/spothero/tools?status.svg)](https://godoc.org/github.com/spothero/tools)
[![Build Status](https://travis-ci.org/spothero/tools.svg?branch=master)](https://travis-ci.org/spothero/tools)
[![codecov](https://codecov.io/gh/spothero/tools/branch/master/graph/badge.svg)](https://codecov.io/gh/spothero/tools)
[![Go Report Card](https://goreportcard.com/badge/github.com/spothero/tools)](https://goreportcard.com/report/github.com/spothero/tools)


The SpotHero Tools Library is used internally at SpotHero across our Go programs. This library is a
collection of common utilities and functions that don't yet stand on their own as individual
libraries.

Additionally, an example server, template Makefile, and yeoman project generator are provided as a
convenience for users.

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
library simply add this as a dependency in your [dep Gopkg.toml](https://github.com/golang/dep/blob/master/docs/Gopkg.toml.md) using the latest release.

Currently, this library supports the following features:

* AWS Utilities
* CLI Utilities
* Kafka
  * Support for consuming and producing metrics
  * Support for goroutine-based callback functions where types are automatically deduced and
    unpacked
  * Schema Registry
* Avro Decoding
* HTTP Server with instrumentation
* Prometheus Metrics
* Kubernetes API Listeners
* High-Performance Logging
* Sentry Integration
* OpenTracing/Jaeger Tracing Support

In addition, all the above packages may automatically be integrated with Cobra/Viper CLIs for
12-factor application compatibility via the CLI module.

### Getting Setup

Usage of this library simply requires you to specify this package in your [dep Gopkg.toml](https://github.com/golang/dep/blob/master/docs/Gopkg.toml.md).

For example:

```
...
[[constraint]]
  branch = "master"
  name = "github.com/spothero/tools"
```

Then, in your application you can simply do the following:

```go
package coolpkg

import (
  "github.com/spothero/tools"
  ...
)
...
```

### Usage

A simple example is provided under [examples/example_server.go](examples/example_server.go) which shows usage of this
library to create a simple 12-factor Go Web application which has tracing, logging, metrics,
sentry, and local caching enabled.

For production applications, we recommend separating the Cobra/Viper command portion into its own
`cmd/` directory, and your application logic into a `pkg/` directory as is standard with most Go
applications.

Additionally, the [Makefile](Makefile) for this project is an excellent example which you can (and should)
borrow for your own projects.

### License
Apache 2.0
