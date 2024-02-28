package tcpproxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"
)

type echoServer struct {
	port        string
	wg          *sync.WaitGroup
	shutdownChn chan struct{}
}

func newEchoServer(port string, shutdown chan struct{}) *echoServer {
	return &echoServer{
		port:        port,
		wg:          &sync.WaitGroup{},
		shutdownChn: shutdown,
	}
}

func (e *echoServer) listen() {
	listener, err := net.Listen("tcp", e.port)
	if err != nil {
		panic(fmt.Errorf("failed to create listener, err: %w", err))
	}

	for {
		select {
		case <-e.shutdownChn:
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				continue
			}
			e.wg.Add(1)
			go e.handleConnection(conn)
		}
	}
}

func (e *echoServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	defer e.wg.Done()
	reader := bufio.NewReader(conn)
	for {
		bytes, err := reader.ReadBytes(byte('\n'))
		if err != nil {
			if err != io.EOF {
				fmt.Println("failed to read data, err:", err)
			}
			return
		}

		_, _ = conn.Write(bytes)
	}
}
