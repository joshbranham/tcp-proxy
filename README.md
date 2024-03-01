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

### Certificates

The proxy is configured to listen with TLS, requiring the client and proxy to have certificates signed and trusted by
each-other. This is accomplished by using the same CA. This repository provides sample certificates in `certificates/`,
as well as scripts to generate a new CA and client/server certificates.

The certificates generated and committed to this repo are also used in tests as fixtures.

#### Generating

You can create a new CA key and certificate:

        cd certificates && ./generate-ca.sh

To then generate certificates for the proxy to use, and 2 client certificates:

        cd certificates && ./generate-clients.sh