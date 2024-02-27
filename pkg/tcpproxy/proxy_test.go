package tcpproxy

import (
	"bufio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"net"
	"testing"
	"time"
)

// This is a naive assumption that this port is free, but is fine for this use case.
const testProxyListener = ":60999"

func Test_ProxyForwardsRequests(t *testing.T) {
	// Start our echoServer which will receive proxied requests and echo back
	go echoServer(testProxyListener)

	// Startup our proxy and begin listening
	proxy := setupTestProxy(t)
	err := proxy.Listen()
	assert.NoError(t, err)

	// Connect to our proxy instance
	conn, err := net.Dial("tcp", proxy.listener.Addr().String())
	reader := bufio.NewReader(conn)
	assert.NoError(t, err)

	// Send some data through our proxy
	_, err = conn.Write([]byte("hello world\n"))
	assert.NoError(t, err)

	// Confirm our upstream echoServer was reached and sent back our data
	result, _ := reader.ReadBytes(byte('\n'))
	assert.Equal(t, "hello world\n", string(result))
	assert.NoError(t, conn.Close())
	assert.NoError(t, proxy.Close())
}

func setupTestProxy(t *testing.T) *Proxy {
	targets := []string{testProxyListener}
	loadBalancer, err := NewLeastConnectionBalancer(targets)
	require.NoError(t, err)

	config := &Config{
		ListenerConfig: &ListenerConfig{
			ListenerAddr: "127.0.0.1:0",

			// TODO: Not implemented yet
			Ca:          "",
			Certificate: "",
			PrivateKey:  "",
		},
		UpstreamConfig: &UpstreamConfig{
			Name:    "test",
			Targets: targets,

			// TODO: Not implemented yet
			AuthorizedGroups: []string{""},
		},
		Timeout: 2 * time.Second,
	}
	return New(config, loadBalancer, slog.Default())
}
