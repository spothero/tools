### SpotHero Core Golang Library

#### Overview
This library contains common modules for use in all GoLang projects across SpotHero. To use this
library simply add this as a dependency in your [dep Gopkg.toml](https://github.com/golang/dep/blob/master/docs/Gopkg.toml.md) using the latest release.

Currently, this library supports the following features:

* Caching
  * Local In-Memory Caching
  * Remote Redis Caching
  * Tiered Caching (First layer cache: Local In-Memory, Second-Layer cache: Remote Redis)
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
* New Relic Integration
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
  name = "github.com/spothero/core"
```

Then, in your application you can simply do the following:

```go
package coolpkg

import (
  "github.com/spothero/core"
  ...
)
...
```

### Usage

A simple example is provided under [example_test.go](example_test.go) which shows usage of this
library to create a simple 12-factor Go Web application which has tracing, logging, metrics,
sentry, new relic, and local caching enabled.

For production applications, we recommend separating the Cobra/Viper command portion into its own
`cmd/` directory, and your application logic into a `pkg/` directory as is standard with most Go
applications.

Additionally, the [Makefile][Makefile] for this project is an excellent example which you can (and should)
borrow for your own projects.
