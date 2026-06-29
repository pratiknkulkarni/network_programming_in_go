Perfect. Let us start with **Phase 1 only**.

# Phase 1 — Basic TCP Server + One-Shot Health Check

## What we are building

A tiny TCP server:

```text
server listens on 127.0.0.1:9000
```

A health-check client connects and sends:

```text
HEALTH
```

The server replies:

```text
OK
```

Then both sides close the connection.

This phase maps to Chapter 3 topics around **establishing TCP connections using Go’s standard library**, specifically `net.Listen`, `Accept`, `net.Dial`, `net.Conn`, and clean connection closing. Chapter 3 covers binding/listening/accepting on the server side and dialing from the client side.  The Go `net` package also documents this same basic pattern: servers use `Listen` and `Accept`, while clients use `Dial`. [\[pkg.go.dev\]](https://pkg.go.dev/net) [\[pkg.go.dev\]](https://pkg.go.dev/github.com/go-lang/go/src/pkg/net)

***

## Folder structure

Create this structure:

```text
tcp-playground/
  go.mod
  cmd/
    server/
      main.go
    check/
      main.go
```

***

## Step 1: Create `go.mod`

From inside `tcp-playground/`, run:

```bash
go mod init tcp-playground
```

Your `go.mod` should look like this:

```go
module tcp-playground

go 1.22
```

If your local Go version is newer, that is fine.

***

# Server code

Create:

```text
cmd/server/main.go
```

```go
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

		handleConn(conn)
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
```

***

# Health-check client code

Create:

```text
cmd/check/main.go
```

```go
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
```

***

# How to run it

Open one terminal and start the server:

```bash
go run ./cmd/server
```

Expected server output:

```text
server listening on 127.0.0.1:9000
```

Open another terminal and run the health checker:

```bash
go run ./cmd/check
```

Expected client output:

```text
healthy
```

Expected server-side logs:

```text
accepted connection from 127.0.0.1:xxxxx
received from 127.0.0.1:xxxxx: "HEALTH"
closed connection from 127.0.0.1:xxxxx
```

***

## What just happened

At a high level:

```text
check client                  server

net.Dial      ----------->    listener.Accept
HEALTH        ----------->    ReadString('\n')
ReadString    <-----------    OK
conn.Close    ----------->    conn.Close
```

The server:

1. Binds to `127.0.0.1:9000`.
2. Listens for TCP connections.
3. Accepts one connection.
4. Reads one line.
5. Replies with `OK` if the message is `HEALTH`.
6. Closes the connection.

The client:

1. Dials the server.
2. Sends `HEALTH`.
3. Reads the response.
4. Prints `healthy` if the response is `OK`.
5. Closes the connection.

***

## Phase 1 mapping to Chapter 3

| Code part                 | Chapter 3 topic                                         |
| ------------------------- | ------------------------------------------------------- |
| `net.Listen("tcp", addr)` | Binding and listening for TCP connections               |
| `listener.Accept()`       | Accepting a TCP session                                 |
| `net.Dial("tcp", addr)`   | Establishing a connection with a server                 |
| `net.Conn`                | Working with TCP sessions through Go’s standard library |
| `conn.Close()`            | Gracefully terminating a TCP session                    |
| `ReadString` / `Fprintln` | Basic data exchange over a TCP stream                   |

This is intentionally simple. In **Phase 2**, we will make the health checker more realistic by adding a dial timeout, so it does not wait forever when the server is unavailable.
