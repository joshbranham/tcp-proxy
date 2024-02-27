# tcp-proxy

An implementation of a L4 TCP proxy written in Golang. Provides the following functionality:

* mTLS authentication, with an authorization system to define what groups can access the upstreams

* Builtin least-connection forwarding to available upstreams

* Per-client rate-limiting, using a token bucket implementation.

## Development

In order to build and develop the proxy, you should have Go installed and available.

To install additional tools, run the setup script:

    ./bin/setup

In order to build the (yet to come) server binary:

    make build

In order to run linting:

    make lint

In order to run tests:

    make test
