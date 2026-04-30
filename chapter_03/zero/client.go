package zero

import (
	"fmt"
	"net"
)

func Client() {
	conn, err := net.Dial("tcp", "127.0.0.1:3000")
	if err != nil {
		fmt.Println("error in dialing -> ")
		fmt.Println(err)
	}

	fmt.Println(conn.LocalAddr().String(), conn.RemoteAddr().String())

	defer conn.Close()
}
