package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

func main() {
	addr := "127.0.0.1:9000"

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	log.Printf("server listening on %s", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}

		go handleConn(conn)
	}

}

func handleConn(conn net.Conn) {
	defer conn.Close()

	log.Printf("accepted connection from %s", conn.RemoteAddr())

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')

	if err != nil {
		log.Printf("read from %s: %v", conn.RemoteAddr(), err)
		return
	}

	msg := strings.TrimSpace(line)
	log.Printf("received from %s: %q", conn.RemoteAddr(), msg)

	switch msg {
	case "HEALTH":
		_, err = fmt.Fprintln(conn, "OK")
	default:
		_, err = fmt.Fprintln(conn, "UNKNOWN")
	}

	if err != nil {
		log.Printf("write to %s: %v", conn.RemoteAddr(), err)
		return
	}

	log.Printf("closed connection from %s", conn.RemoteAddr())
}
