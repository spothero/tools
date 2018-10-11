### <%=appName%>

#### Overview
<%=appName%> is a service built in Golang. TODO: Fill this out.

### Getting Setup

#### Prerequisites
- Install Golang 1.11 or later
    - `brew install go`
    - Make sure the _gobin_ is setup on your `PATH`
    - `vim ~/.zshrc`
    - `export PATH=$GOPATH/bin:$PATH`
- Check out this project _into your gopath_:
    - `mkdir -p $GOPATH/src/github.com/spothero && pushd $GOPATH/src/github.com/spothero && git co git@github.com:spothero/<%=appName%>.git && popd`

#### Building the project from the command line

- Simply invoke the `Makefile` in the root directory
  - `make`

[Dependencies are managed with Dep](https://github.com/golang/dep). Changes to Gopkg.toml/lock
should be checked in.

Other build commands of note:

- `make build` - Builds <%=appName%> the server without tests or linting
- `make lint` - Lints the code
- `make test` - Tests the code
- `make coverage` - Tests the code and presents an interactive coverage report
- `make vendor` - Pulls and/or updates dependencies
- `make clean` - Removes all vendored dependencies and artifacts
- `make docker` - Builds the <%=appName%> docker image and pushes it to Dockerhub
- `make docker_build` - Builds the <%=appName%> docker image
- `make docker_run` - Runs the <%=appName%> docker image
- `make docker_push` - Pushes the <%=appName%> docker image to Dockerhub

### Usage

The service can be configured either by CLI or Environment Variables. The CLI is primarily provided
as a development convenience. In deployed environments the 12-factor compatible
environment variable configuration is preferred. CLI arguments can easily be translated to their
environment variable cousins with the following approach:

`<%=appName%>_<CONFIG_NAME>=<VALUE>`

Config names are translated from `skewered-values` to `SKEWERED_VALUES` when used as environment
variables.

For example, to set `tracer-enabled` to `false` instead of `true`:

`<%=appName%>_TRACER_ENABLED=false`

Once the service is compiled you can see usage by invoking its CLI:

```
> <%=appName%> server --help

TODO: Run command and output CLI usage here
```

### Editing Tools for Go
There are a number of editing tools available for this project. Everything from vim/emacs on up is valid. If you want
visual debugging or advanced profiling Goland is a good tool for golang. That being said, [Delve is
the underlying debugger for Goland](https://github.com/derekparker/delve) and works perfectly well
from the command line.

#### Editing your project in Goland
- Simply open the program and import the project
- Configure the runtime to build and use cmd/<%=appName%>
  - Ensure that you supply the `server` argument
  - `<%=appName%> server`

### Docker Images

This project uses [multi-stage Docker builds](https://docs.docker.com/develop/develop-images/multistage-build/),
meaning that the container used to compile and test our containers is actually different and much
smaller than the container we ship for production usage.
