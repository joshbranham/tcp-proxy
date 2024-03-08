package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/joshbranham/tcp-proxy/pkg/tcpproxy"
)

func main() {
	logger := slog.Default()

	// TODO: Configure the proxy here. With more time, using a configuration file (YAML etc) and/or CLI arguments
	// would be a better approach.
	targets := []string{"localhost:9000", "localhost:9001", "localhost:9002"}
	loadBalancer, _ := tcpproxy.NewLeastConnectionBalancer(targets)
	config := &tcpproxy.Config{
		LoadBalancer: loadBalancer,
		ListenerConfig: &tcpproxy.ListenerConfig{
			ListenerAddr: "localhost:5000",

			// TODO: this is dependent https://github.com/joshbranham/tcp-proxy/pull/3
			CA:          "certificates/ca.pem",
			Certificate: "tcp-proxy.pem",
			PrivateKey:  "tcp-proxy.key",
		},
		UpstreamConfig: &tcpproxy.UpstreamConfig{
			Name:    "test",
			Targets: targets,

			AuthorizedGroups: []string{"engineering"},
		},

		// TODO: this is dependent on https://github.com/joshbranham/tcp-proxy/pull/4
		//RateLimitConfig: &tcpproxy.RateLimitConfig{
		//	Capacity: 10,
		//	FillRate: 5 * time.Second,
		//},
		Logger: slog.Default(),
	}

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGINT, syscall.SIGTERM)

	proxy, err := tcpproxy.New(config)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		err = proxy.Serve()
		if err != nil {
			fmt.Print(err)
			os.Exit(1)
		}
		wg.Done()
	}()

	<-sigC
	logger.Info("shutting down proxy...")
	_ = proxy.Close()
	wg.Wait()
	logger.Info("proxy stopped")
}
