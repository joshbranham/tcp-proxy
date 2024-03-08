package tcpproxy

import (
	"bufio"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Configure our echoServer to listen on a random available port on localhost
const echoServerAddr = "localhost:0"

func Test_ProxyForwardsRequests(t *testing.T) {
	// Start our echoServer which will receive proxied requests and echo back.
	echoSrv := newEchoServer(echoServerAddr)
	err := echoSrv.listen()
	require.NoError(t, err)

	// Serve requests in a goroutine
	go func() {
		err := echoSrv.serve()
		require.NoError(t, err)
	}()

	// Startup our proxy and begin listening, forwarding requests to our echoServer resolved
	// address:port.
	proxy := setupTestProxy(t, echoSrv.listener.Addr().String())
	require.NoError(t, err)
	go func() {
		err := proxy.Serve()
		require.NoError(t, err)
	}()

	// Connect to our proxy instance
	conn, err := net.Dial("tcp", proxy.Address())
	require.NoError(t, err)
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
	assert.NoError(t, echoSrv.close())
}

func Test_CannotCloseAlreadyClosed(t *testing.T) {
	proxy := setupTestProxy(t, "localhost:0")
	assert.Error(t, proxy.Close())
}

func Test_CannotServeIfAlreadyServing(t *testing.T) {
	proxy := setupTestProxy(t, "localhost:99999")

	go func() {
		err := proxy.Serve()
		require.NoError(t, err)
	}()

	// Give our proxy serving in a goroutine time to begin serving before trying to call Serve() again. We need
	// the goroutine serving in order to test the double serve behavior erroring.
	time.Sleep(5 * time.Millisecond)
	err := proxy.Serve()
	assert.Error(t, err)
	assert.NoError(t, proxy.Close())
}

func setupTestProxy(t *testing.T, target string) *Proxy {
	targets := []string{target}
	loadBalancer, err := NewLeastConnectionBalancer(targets)
	require.NoError(t, err)

	config := &Config{
		LoadBalancer: loadBalancer,
		ListenerConfig: &ListenerConfig{
			ListenerAddr: "127.0.0.1:0",

			// TODO: Not implemented yet
			CA:          "",
			Certificate: "",
			PrivateKey:  "",
		},
		UpstreamConfig: &UpstreamConfig{
			Name:    "test",
			Targets: targets,

			// TODO: Not implemented yet
			AuthorizedGroups: []string{""},
		},
		RateLimitConfig: &RateLimitConfig{
			Capacity: 10,
			FillRate: 5 * time.Second,
		},
		Logger: slog.Default(),
	}
	proxy, err := New(config)
	require.NoError(t, err)
	return proxy
}
