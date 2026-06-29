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

	conn, err := net.Dial("tcp", addr)

	if err != nil {
		log.Fatalf("dial %s: %v", addr, err)
	}
	defer conn.Close()

	// TODO: what are some other options to write to a connection?
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
