package tcpproxy

import (
	"log/slog"
	"time"
)

// Config is the top-level configuration object used to configure a Proxy.
type Config struct {
	// Balancer is an implementation of LoadBalancer used to select where to route new connections to the proxy.
	Balancer LoadBalancer
	// ListenerConfig comprises the configuration for setting up tls.Listen, like interface:port and TLS configuration.
	ListenerConfig *ListenerConfig
	// UpstreamConfig comprises where to route requests as well as which clients are authorized to do so.
	UpstreamConfig *UpstreamConfig

	// When to give up on a proxied connection and close it.
	Timeout time.Duration

	// Logger is a slog.Logger used for logging proxy activities to stdout.
	Logger *slog.Logger
}

// ListenerConfig is the configuration specific to how the proxy should listen and accept connections.
type ListenerConfig struct {
	// ListenerAddr is passed to tls.Listen, for example, ":5000" to listen on port 5000.
	ListenerAddr string

	// TLS configuration for the listener to use.
	Ca          string
	Certificate string
	PrivateKey  string
}

// UpstreamConfig is the configuration for where to route proxied connections.
type UpstreamConfig struct {
	// Name is a label for the upstreams.
	Name string
	// Targets is a list of available upstreams to proxy requests to.
	Targets []string

	// AuthorizedGroups defines who can proxy to the Targets. Maps to group value extracted from TSL certificate `cn`.
	AuthorizedGroups []string
}
