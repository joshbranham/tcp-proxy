package tcpproxy

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Configure our echoServer to listen on a random available port on localhost
const echoServerAddr = "localhost:0"

func Test_ProxyForwardsRequests_AuthorizedClient(t *testing.T) {
	// Start our echoServer which will receive proxied requests and echo back.
	echoSrv := setupEchoServer(t)

	// Startup our proxy and begin listening, forwarding requests to our echoServer resolved
	// address:port. Allow clients in the group engineering.
	proxy := setupTestProxy(t, echoSrv.listener.Addr().String(), "engineering")

	// Connect to our proxy instance, using user1
	conn, err := tls.Dial("tcp", proxy.Address(), clientTlsConfig(t, "user1"))
	require.NoError(t, err)
	reader := bufio.NewReader(conn)

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

func Test_ProxyForwardsRequests_UnauthorizedClient(t *testing.T) {
	// Start our echoServer which will receive proxied requests and echo back.
	echoSrv := setupEchoServer(t)

	// Startup our proxy and begin listening, forwarding requests to our echoServer resolved
	// address:port. Allow clients in the group administrators.
	proxy := setupTestProxy(t, echoSrv.listener.Addr().String(), "administrators")

	// Connect to our proxy instance, using user1
	conn, err := tls.Dial("tcp", proxy.Address(), clientTlsConfig(t, "user1"))
	require.NoError(t, err)
	reader := bufio.NewReader(conn)

	// Send some data through our proxy
	_, err = conn.Write([]byte("12345\n"))
	assert.NoError(t, err)

	// The proxy should have closed the connection, returning an error here.
	_, err = reader.ReadBytes(byte('\n'))
	assert.Error(t, err)
}

func Test_CannotCloseAlreadyClosed(t *testing.T) {
	proxy := setupTestProxy(t, "localhost:0", "")
	assert.Error(t, proxy.Close())
}

func Test_CannotServeIfAlreadyServing(t *testing.T) {
	proxy := setupTestProxy(t, "localhost:99999", "")

	// Give our proxy serving in a goroutine time to begin serving before trying to call Serve() again. We need
	// the goroutine serving in order to test the double serve behavior erroring.
	time.Sleep(5 * time.Millisecond)
	err := proxy.Serve()
	assert.Error(t, err)
	assert.NoError(t, proxy.Close())
}

func setupTestProxy(t *testing.T, target string, authorizedGroup string) *Proxy {
	targets := []string{target}
	loadBalancer, err := NewLeastConnectionBalancer(targets)
	require.NoError(t, err)

	config := &Config{
		LoadBalancer: loadBalancer,
		ListenerConfig: &ListenerConfig{
			ListenerAddr: "127.0.0.1:0",

			CA:          certificatePath("ca.pem"),
			Certificate: certificatePath("tcp-proxy.pem"),
			PrivateKey:  certificatePath("tcp-proxy.key"),
		},
		UpstreamConfig: &UpstreamConfig{
			Name:    "test",
			Targets: targets,

			AuthorizedGroups: []string{authorizedGroup},
		},
		Logger: slog.Default(),
	}
	proxy, err := New(config)
	require.NoError(t, err)

	go func() {
		err := proxy.Serve()
		require.NoError(t, err)
	}()

	return proxy
}

func setupEchoServer(t *testing.T) *echoServer {
	// Start our echoServer which will receive proxied requests and echo back.
	echoSrv := newEchoServer(echoServerAddr)
	err := echoSrv.listen()
	require.NoError(t, err)

	// Serve requests in a goroutine
	go func() {
		err := echoSrv.serve()
		require.NoError(t, err)
	}()

	return echoSrv
}

func clientTlsConfig(t *testing.T, user string) *tls.Config {
	pool := x509.NewCertPool()
	caData, err := os.ReadFile(certificatePath("ca.pem"))
	require.NoError(t, err)
	pool.AppendCertsFromPEM(caData)

	tlsCert, err := tls.LoadX509KeyPair(
		certificatePath(fmt.Sprintf("%s.pem", user)),
		certificatePath(fmt.Sprintf("%s.key", user)),
	)
	require.NoError(t, err)

	config := &tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{tlsCert},
	}

	return config
}

func certificatePath(name string) string {
	return fmt.Sprintf("../../certificates/%s", name)
}
