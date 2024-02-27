package tcpproxy

import (
	"time"
)

type Config struct {
	ListenerConfig *ListenerConfig
	UpstreamConfig *UpstreamConfig

	// When to give up on a proxied connection and close it.
	Timeout time.Duration
}

type ListenerConfig struct {
	ListenerAddr string // eg :5000

	// TLS configuration for the listener to use.
	Ca          string
	Certificate string
	PrivateKey  string
}

type UpstreamConfig struct {
	Name    string
	Targets []string

	AuthorizedGroups []string // maps to group value extracted from `cn`
}
