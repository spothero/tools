### SpotHero Core Golang Library

#### TLDR

Impatient?

1. Install Golang
    1. `brew install golang`
    2. [Set your `GOPATH` in your `.zshrc`/`.bashrc`/etc](https://github.com/golang/go/wiki/SettingGOPATH)
    3. Add `GOPATH/bin` to your `PATH`
        1. `export PATH=$GOPATH/bin:$PATH`
2. `make`
3. `./example_server`
4. Open your browser to `http://localhost:8080`

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

### Help! Dep is hanging and won't finish when I add this dependency!

Dep is hanging because this is a private repository. It needs to be configured to access it
properly.

#### For Local Development
On OSX you need to ensure that your git ssh credentials do not require a password. Add the
following to `~/.ssh/config`:

```
Host *
   UseKeychain yes
```

#### For building Docker containers
In your Makefile you can add the following lines to use AWS CLI to fetch the Git pull github
secret:

```Makefile
GITHUB_AUTH_USER ?= spotheropullonly
GITHUB_TOKEN ?= $(shell aws secretsmanager get-secret-value --secret-id arn:aws:secretsmanager:us-west-2:913289439155:secret:github-larry-pull-only-auth-token-O9UM5J | jq -r .SecretString)

...

docker_build:
  docker build --build-arg GITHUB_TOKEN='${GITHUB_TOKEN}' -t "spothero/<your-app>:<your-version>" .
```

In Dockerfiles add the following:

```Dockerfile
ARG GITHUB_TOKEN="not-set"
ENV GITHUB_TOKEN $GITHUB_TOKEN
RUN git config --global url."https://".insteadOf git://
RUN echo "machine github.com login spotheropullonly password $GITHUB_TOKEN" > /root/.netrc

# Note: this assumes you're using a Makefile
RUN make

RUN rm /root/.netrc
```

You should **always** use multistage builds for both performance and size reasons, as well as for
security reasons. This is the best way to guarantee a secret is never left behind in a built docker
image, private or not.


### Usage

A simple example is provided under [examples/example_server.go](examples/example_server.go) which shows usage of this
library to create a simple 12-factor Go Web application which has tracing, logging, metrics,
sentry, new relic, and local caching enabled.

For production applications, we recommend separating the Cobra/Viper command portion into its own
`cmd/` directory, and your application logic into a `pkg/` directory as is standard with most Go
applications.

Additionally, the [Makefile](Makefile) for this project is an excellent example which you can (and should)
borrow for your own projects.
