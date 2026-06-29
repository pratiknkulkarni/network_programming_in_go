package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	_, err = fmt.Fprintln(conn, "FLOOD")
	if err != nil {
		log.Fatalf("write request: %v", err)
	}

	// reader := bufio.NewReader(conn)

	// line, err := reader.ReadString('\n')
	// if err != nil {
	// 	log.Fatalf("read health response: %v", err)
	// }

	// log.Print(line)

	log.Println("sent FLOOD request; now intentionally not reading")

	time.Sleep(30 * time.Second)
}
