---
authors: Josh Branham (josh.php@gmail.com)
state: draft
---

# Gravitational Interview: L4 TCP Proxy

## What

Design an L4 TCP proxy in Golang, configured with mTLS client authentication, per-client rate limiting and least-connection forwarding
to upstreams.

## Details

### Overall Approach

I will build a configurable library that allows for running a proxy with the described features. The proxy will be invoked via a simple
CLI, where base configuration will be hard coded for simplicity. The proxy will handle concurrent requests to defined upstreams, while
denying connections from unknown or unauthorized clients. It will use least-connection forwarding to balance connections across N
upstreams of the same type (ie a pool of web processes). Each client will have a rate limit defining how many connections they can
have open at a given time to the upstreams.

### Scope

The proxy will operate at TCP Layer 4, meaning any protocol that operates on TCP and supports certificate based authentication
should work.

### Library API

For the library component, the following configuration options will be exposed. These will be set when constructing a new `Proxy` struct.

* Listener configuration: a port on which to listen and accept connections, and a certificate and private key for TLS.

* Upstream configuration: a struct defining upstreams, including their name and address:port.

* Authorization configuration: what client(s), identified by their mTLS certificate, can connect to what upstreams.

* Rate limit configuration: for a given client, identified by it's mTLS certificate, how many concurrent connections can it have.

* Global timeout: a timeout to apply to all connections to ensure they are not left open and hung.

Additionally, the library will utilize a logger to log common events like new connections.

### Security Considerations

In order to ensure unauthroized clients cannot proxy to upstreams, the proxy will utilize mTLS for authn/z. The server and client certificates will
be generated with RSA 2048bit encryption, and checked into the repo for this example. The certificate `cn=` will be used to identify the user of the proxy.

The CA used to generate the certificates will be used by the client and server (proxy) to validate that both are trusted.

### Concurrency

Utilizing primitives like goroutines, channels and mutexes, the proxy will handle concurrent connections properly. This means spawning goroutines for requests,
keeping track of connections using a mutex to increment/decrement a counter(s), and channels to ensure proper shutdown of the proxy (closing connections etc).

This will be one of the key focuses of the library, ensuring this is done properly and is not racey.

### CLI UX

The server component will wrap the library in a simple CLI that can be invoked. Below is an example usage that invokes the proxy listening on port `5000`:

```bash
./out/proxy :5000
```

### Testing & Integration

The user of the proxy will be able to modify the CLI binary to provide their own configuration, however I will include any integration testing I used
to validate the proxy. This could be either a docker compose file, or a small binary to run N upstreams that simply echo data back to the caller.
