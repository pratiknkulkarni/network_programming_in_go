package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"net"
)

func main() {
	payload := make([]byte, 1<<24) // 16 MB

	n, err := rand.Read(payload)
	if err != nil {
		fmt.Println("rand.Read:", err)
		return
	}

	fmt.Printf("generated %d random bytes\n", n)

	listener, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		fmt.Println("listen:", err)
		return
	}
	defer listener.Close()

	go runServer(listener, payload)

	err = runClient(listener.Addr().String())
	if err != nil {
		fmt.Println("client:", err)
	}
}

func runServer(listener net.Listener, payload []byte) {
	conn, err := listener.Accept()
	if err != nil {
		fmt.Println("accept:", err)
		return
	}
	defer conn.Close()

	n, err := conn.Write(payload)
	if err != nil {
		fmt.Println("write:", err)
		return
	}

	fmt.Printf("server wrote %d bytes\n", n)
}

func runClient(address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}
	defer conn.Close()

	fmt.Println("client connected")

	buf := make([]byte, 1024)

	var total int64

	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				fmt.Println("server closed connection")
				break
			}

			return err
		}

		total += int64(n)

		fmt.Printf("client read %d bytes, total=%d\n", n, total)

		// The actual data read is in buf[:n].
		// Avoid printing it for random binary data because it will be noisy.
		// fmt.Printf("data: %v\n", buf[:n])
	}

	fmt.Printf("client received total %d bytes\n", total)
	return nil
}
