package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

const readTimeout = 2 * time.Second
const writeTimeout = 2 * time.Second

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

	remoteAddr := conn.RemoteAddr().String()

	log.Printf("accepted connection from %s", remoteAddr)

	err := conn.SetReadDeadline(time.Now().Add(readTimeout))
	if err != nil {
		log.Printf("set read deadline for %s: %v", remoteAddr, err)
		return
	}

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')

	if err != nil {
		// log.Printf("read from %s: %v", remoteAddr, err)
		handleReadError(remoteAddr, err)
		return
	}

	msg := strings.TrimSpace(line)
	log.Printf("received from %s: %q", remoteAddr, msg)

	err = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err != nil {
		log.Printf("write to %s: %v", remoteAddr, err)
		return
	}

	switch msg {
	case "HEALTH":
		_, err = fmt.Fprintln(conn, "OK")
	case "FLOOD":
		flood(conn)
	default:
		_, err = fmt.Fprintln(conn, "UNKNOWN")
	}

	if err != nil {
		// log.Printf("write to %s: %v", remoteAddr, err)
		handleWriteError(remoteAddr, err)
		return
	}

	log.Printf("closed connection from %s", remoteAddr)
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func handleWriteError(remoteAddr string, err error) {
	if isTimeout(err) {
		log.Printf("write timeout to %s", remoteAddr)
		return
	}

	log.Printf("write to %s: %v", remoteAddr, err)
}

func handleReadError(remoteAddr string, err error) {
	if errors.Is(err, io.EOF) {
		log.Printf("client %s closed the connection", remoteAddr)
		return
	}

	if isTimeout(err) {
		log.Printf("read timeout from %s", remoteAddr)
		return
	}

	log.Printf("read from %s: %v", remoteAddr, err)
}

func flood(conn net.Conn) error {
	chunk := strings.Repeat("x", 1024*1024)

	for i := 0; i < 10_000; i++ {
		if err := conn.SetWriteDeadline(time.Now().Add(1 * time.Second)); err != nil {
			return err
		}

		_, err := fmt.Fprintln(conn, chunk)
		if err != nil {
			return err
		}
	}

	return nil
}
