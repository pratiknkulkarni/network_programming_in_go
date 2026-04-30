package zero

import (
	"fmt"
	"net"
)

func Server() {
	listener, err := net.Listen("tcp", "127.0.0.1:3000")
	if err != nil {
		fmt.Println("error creating server\t", err)
		return
	}

	conn, err := listener.Accept()
	if err != nil {
		fmt.Println("error accepting connection\t", err)
		return
	}

	fmt.Println(conn.LocalAddr().String(), conn.RemoteAddr().String())

	defer conn.Close()
	defer listener.Close()
}
