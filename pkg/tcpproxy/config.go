package tcpproxy

import (
	"fmt"
	"log/slog"
	"time"
)

// Config is the top-level configuration object used to configure a Proxy.
type Config struct {
	// LoadBalancer is an LeastConnectionBalancer for deciding how to route requests to targets.
	LoadBalancer *LeastConnectionBalancer
	// ListenerConfig comprises the configuration for setting up tls.Listen, like interface:port and TLS configuration.
	ListenerConfig *ListenerConfig
	// UpstreamConfig comprises where to route requests as well as which clients are authorized to do so.
	UpstreamConfig *UpstreamConfig

	// IdleTimeout is a duration when to give up on a proxied connection that is idle and close it.
	IdleTimeout time.Duration

	// Logger is a slog.Logger used for logging proxy activities to stdout.
	Logger *slog.Logger
}

// ListenerConfig is the configuration specific to how the proxy should listen and accept connections.
type ListenerConfig struct {
	// ListenerAddr is passed to tls.Listen, for example, ":5000" to listen on port 5000.
	ListenerAddr string

	// TLS configuration for the listener to use. The values should be string data with certificates
	// in PEM format.
	CA          string
	Certificate string
	PrivateKey  string
}

// UpstreamConfig is the configuration for where to route proxied connections.
type UpstreamConfig struct {
	// Name is a label for the upstreams.
	Name string
	// Targets is a list of available upstream network addresses to proxy requests to.
	Targets []string

	// AuthorizedGroups defines who can proxy to the Targets. Maps to group value extracted from TSL certificate `cn`.
	AuthorizedGroups []string
}

// Validate confirms a given Config has all required fields set.
func (c *Config) Validate() error {
	if c.ListenerConfig == nil {
		return fmt.Errorf("config does not contain a ListenerConfig")
	}
	if c.LoadBalancer == nil {
		return fmt.Errorf("config does not contain a LoadBalancer")
	}
	if c.UpstreamConfig == nil {
		return fmt.Errorf("config does not contain a UpstreamConfig")
	}
	if c.Logger == nil {
		return fmt.Errorf("config does not contain a Logger")
	}

	return nil
}
