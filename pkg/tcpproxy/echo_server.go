package tcpproxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
)

// echoServer should be used only for testing purposes. It can be launched in a goroutine to test
// our proxy behavior.
func echoServer(port string) {
	listener, err := net.Listen("tcp", port)
	if err != nil {
		panic(fmt.Errorf("failed to create listener, err: %w", err))
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
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
