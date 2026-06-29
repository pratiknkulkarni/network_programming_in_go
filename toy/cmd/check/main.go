package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9000", "TCP address to check")
	timeout := flag.Duration("timeout", 2*time.Second, "dial timeout")
	flag.Parse()

	start := time.Now()

	// conn, err := net.Dial("tcp", addr)
	conn, err := net.DialTimeout("tcp", *addr, *timeout)
	elapsed := time.Since(start)

	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			log.Fatalf("unhealthy: dial TIMEOUT after %s", *timeout)
		}

		log.Fatalf("unhealthy: dial %s failed after %s: %v", *addr, elapsed, err)
	}

	defer conn.Close()

	_, err = fmt.Fprintln(conn, "HEALTH")
	if err != nil {
		log.Fatalf("write health request: %v", err)
	}

	reader := bufio.NewReader(conn)

	line, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("read health response: %v", err)
	}

	response := strings.TrimSpace(line)

	if response == "OK" {
		fmt.Println("healthy")
		return
	}

	fmt.Printf("unhealthy: unexpected response %q\n", response)
}
