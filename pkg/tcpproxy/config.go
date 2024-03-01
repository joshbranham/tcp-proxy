package tcpproxy

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log/slog"
)

// Config is the top-level configuration object used to configure a Proxy.
type Config struct {
	// LoadBalancer is an LeastConnectionBalancer for deciding how to route requests to targets.
	LoadBalancer *LeastConnectionBalancer
	// ListenerConfig comprises the configuration for setting up tls.Listen, like interface:port and TLS configuration.
	ListenerConfig *ListenerConfig
	// UpstreamConfig comprises where to route requests as well as which clients are authorized to do so.
	UpstreamConfig *UpstreamConfig

	// Logger is a slog.Logger used for logging proxy activities to stdout.
	Logger *slog.Logger
}

// ListenerConfig is the configuration specific to how the proxy should listen and accept connections.
type ListenerConfig struct {
	// ListenerAddr is passed to tls.Listen, for example, ":5000" to listen on port 5000.
	ListenerAddr string

	// TLS configuration for the listener to use. The values should be certificates in PEM format.
	CA          []byte
	Certificate []byte
	PrivateKey  []byte
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
		return errors.New("config does not contain a ListenerConfig")
	}
	if c.LoadBalancer == nil {
		return errors.New("config does not contain a LoadBalancer")
	}
	if c.UpstreamConfig == nil {
		return errors.New("config does not contain a UpstreamConfig")
	}
	if c.Logger == nil {
		return errors.New("config does not contain a Logger")
	}

	return nil
}

func (c *Config) TLSConfig() (*tls.Config, error) {
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(c.ListenerConfig.CA)

	cert, err := tls.X509KeyPair(
		c.ListenerConfig.Certificate,
		c.ListenerConfig.PrivateKey,
	)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            pool,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		ClientCAs:          pool,
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS13,
	}, nil
}
