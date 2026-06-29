package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9001", "TCP server address")
	dialTimeout := flag.Duration("dial-timeout", 2*time.Second, "dial timeout")
	interval := flag.Duration("interval", 2*time.Second, "heartbeat interval")
	responseTimeout := flag.Duration("response-timeout", 3*time.Second, "PONG response timeout")
	flag.Parse()

	conn, err := net.DialTimeout("tcp", *addr, *dialTimeout)
	if err != nil {
		log.Fatalf("dial %s: %v", *addr, err)
	}
	defer conn.Close()

	log.Printf("connected to %s", *addr)

	reader := bufio.NewReader(conn)
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		// write to the server
		_, err := fmt.Fprintln(conn, "PING")

		if err != nil {
			log.Fatalf("send PING: %v", err)
		}

		log.Printf("sent PING awaiting PONG")

		err = conn.SetReadDeadline(time.Now().Add(*responseTimeout))
		if err != nil {
			log.Fatalf("set read deadline: %v", err)
		}

		line, err := reader.ReadString('\n')

		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Fatalf("server closed the connection")
			}

			if isTimeout(err) {
				log.Fatalf("unhealthy: no PONG within %s", *responseTimeout)
			}
			log.Fatalf("read PONG: %v", err)
		}

		// wait for the pong
		msg := strings.TrimSpace(line)
		log.Printf("received PONG")

		// if no pong, check error type and log
		if msg != "PONG" {
			log.Fatalf("unhealthy: expected PONG, got %q", msg)
		}

		// increment ticker
		<-ticker.C
	}
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
