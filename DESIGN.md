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
should work. The final project will include the proxy library and a CLI used to run the proxy.

### Library API

The `tcpproxy` package will provide a struct representing the configuration and state needed to run the proxy.

It will look something like the following:

```golang

type Proxy struct {
  Config  *tcpproxy.Config

  listener      net.Listener
  wg            *sync.WaitGroup
  shutdownChn   chan struct{}
  connectionChn chan net.Conn
}
```

A new instance of the proxy, instantiated with `tcpproxy.New(...)`, will have the following functions available:

* `Listen()` to start the proxy and listen for connections on the provided listener.

* `Close()` to shut the proxy down gracefully, signalled when the system sends a `SIGINT` or `SIGTERM`.

A sample of instantiating a proxy from a CLI package and listening will look like the following:

```golang
package main

import "tcpproxy"

func main() {
  // argument parsing

  logger := slog.New(...)
  config := tcpproxy.NewConfiguration(logger, ...)
  proxy := tcpproxy.New(config)

  // signal handling

  err := proxy.Listen()
  if err != nil {
    // log error
    os.Exit(1)
  }

}
```

See the [configuration](####-Configuration) structure for details on what will be passed to `tcpproxy.New(...)`.

#### Configuration

As described above, a new instance of the `Proxy` will take a configuration object.

The object will look as follows:

```golang
// Top level configuration object
type Configuration struct {
  ListenerConfig  *ListenerConfig
  UpstreamConfig  *UpstreamConfig
  RateLimitConfig *RateLimitConfig

  // When to give up a proxied connection and close it.
  Timeout        time.Duration
  Logger         *slog.Logger
}

// How the proxy listens for connections on the machine it is running from
type ListenerConfig struct {
  ListenerAddr string // eg :5000

  // TLS configuraition for the listener to use.
  Ca         string
  Certificate string
  PrivateKey string
}

// Individual configuration for an upstream "group"
type UpstreamConfig struct {
  Name    string
  Targets []string

  AuthorizedGroups []string // maps to group value extracted from `cn`
}

// Configuration for a token bucket rate limiting algorithm, applied to each client.
type RateLimitConfig struct {
  Capacity int // Capacity of the per-client bucket
  FillRate time.Duration // How often to add 1 token to the bucket, eg. '1 * time.Second'
}
```

##### Timeouts

The `Timeout` option in the `Configuration` struct denotes how long a proxied connection can be idle, before it will be closed
by the server.

### Security Considerations

In order to ensure unauthorized clients cannot proxy to upstreams, the proxy will utilize mTLS for authn. The server and client certificates will
be generated with RSA 2048bit encryption, and checked into the repo for this example. Client certificates will be generated with the `cn`
configured with a value representing the user and a group, such as `cn = jbranham@admin`.

The server will require TLS 1.3 and the default ciphersuite selection provided by the `crypto/tls` Go package as the minimum version.

Authorization will be handled in the configuration of the proxy, denoting what upstreams clients have access to. The `cn` will be used
to identify clients.

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
