Great. Let us move to **Phase 3 — Make the server concurrent**.

# Phase 3 — Concurrent TCP Server

## What changes in this phase?

Right now, the server handles one connection at a time:

```go
handleConn(conn)
```

That means while one client is being processed, the server does **not** accept and process another client.

In Phase 3, we change that to:

```go
go handleConn(conn)
```

Now every accepted connection is handled in its own goroutine.

This maps to Chapter 3’s TCP server pattern: bind/listen, accept connections, and handle TCP sessions using Go’s standard library. The chapter also highlights the importance of closing connections properly to avoid resource leaks.  The Go `net` package documentation also shows the common pattern of creating a listener, accepting connections in a loop, and handling each connection separately. [\[pkg.go.dev\]](https://pkg.go.dev/net) [\[pkg.go.dev\]](https://pkg.go.dev/github.com/go-lang/go/src/pkg/net)

***

## Phase 3 topic mapping

| Code change                  | Chapter 3 topic                        |
| ---------------------------- | -------------------------------------- |
| `listener.Accept()` loop     | Accepting TCP sessions                 |
| `go handleConn(conn)`        | Handling multiple clients concurrently |
| `defer conn.Close()`         | Graceful TCP session cleanup           |
| Logging remote addresses     | Observing individual TCP sessions      |
| One goroutine per connection | Practical Go TCP server pattern        |

***

# Update `cmd/server/main.go`

Replace your current `cmd/server/main.go` with this:

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

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()

	log.Printf("accepted connection from %s", remoteAddr)

	reader := bufio.NewReader(conn)

	line, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("read from %s: %v", remoteAddr, err)
		return
	}

	msg := strings.TrimSpace(line)
	log.Printf("received from %s: %q", remoteAddr, msg)

	switch msg {
	case "HEALTH":
		_, err = fmt.Fprintln(conn, "OK")
	default:
		_, err = fmt.Fprintln(conn, "UNKNOWN")
	}

	if err != nil {
		log.Printf("write to %s: %v", remoteAddr, err)
		return
	}

	log.Printf("closed connection from %s", remoteAddr)
}
```

***

## What changed?

Only one important line changed:

```go
go handleConn(conn)
```

Previously:

```go
handleConn(conn)
```

The old version processed each connection synchronously.

The new version starts a goroutine for each connection, so the server can immediately go back to:

```go
listener.Accept()
```

and wait for the next client.

***

## Why I also changed this line

I added this:

```go
remoteAddr := conn.RemoteAddr().String()
```

Instead of calling this repeatedly:

```go
conn.RemoteAddr()
```

This is mostly for cleaner logging. It also avoids calling methods on `conn` later after the connection is already closing.

***

# How to run it

## Terminal 1 — start server

```bash
go run ./cmd/server
```

Expected:

```text
server listening on 127.0.0.1:9000
```

## Terminal 2 — run check

```bash
go run ./cmd/check
```

Expected:

```text
healthy: connected in ...
```

## Run several checks quickly

You can run this a few times:

```bash
go run ./cmd/check
go run ./cmd/check
go run ./cmd/check
```

Or, from a Unix-like shell:

```bash
for i in 1 2 3 4 5; do go run ./cmd/check & done
wait
```

Expected output:

```text
healthy: connected in ...
healthy: connected in ...
healthy: connected in ...
healthy: connected in ...
healthy: connected in ...
```

The server logs should show multiple accepted connections.

***

## What we have after Phase 3

The project now has:

```text
server:
  listens on 127.0.0.1:9000
  accepts many clients
  handles each client in its own goroutine
  replies OK to HEALTH
  closes each connection cleanly

check:
  dials with timeout
  sends HEALTH
  expects OK
  reports healthy/unhealthy
```

***

## Important caution

This goroutine-per-connection pattern is very common and useful. However, it also means each connected client consumes some resources. In later phases, deadlines will help us avoid clients connecting and then staying silent forever.

So the next phase is important:

> **Phase 4 — Add read/write deadlines**

That will protect the server from idle or slow clients.
