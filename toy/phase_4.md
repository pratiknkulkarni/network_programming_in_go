Great — moving to **Phase 4: Add read/write deadlines**.

# Phase 4 — Read and Write Deadlines

## What we are adding

Right now, a client can connect to the server and then do nothing. Because the server uses:

```go
reader.ReadString('\n')
```

that goroutine can block indefinitely waiting for a newline.

In this phase, we will protect the server by setting a **read deadline**. If a client connects but does not send a full line within the deadline, the server closes that connection.

We will also add a **write deadline** before sending the response, so the server does not block forever while trying to write to a stuck or broken client.

This maps to Chapter 3’s topic of **Implementing Deadlines**, including `SetReadDeadline`, `SetWriteDeadline`, timeout errors, and closing idle connections. [\[pkg.go.dev\]](https://pkg.go.dev/net)

***

## Phase 4 topic mapping

This phase covers:

* `SetReadDeadline`
* `SetWriteDeadline`
* detecting timeout errors using `net.Error`
* protecting the server from idle clients
* preventing connection goroutines from hanging forever
* keeping TCP session cleanup predictable

The Go `net.Conn` abstraction supports deadline-based connection operations, and Chapter 3 builds on this to show how practical TCP programs avoid indefinite blocking. [\[pkg.go.dev\]](https://pkg.go.dev/net), [\[pkg.go.dev\]](https://pkg.go.dev/github.com/go-lang/go/src/pkg/net)

***

# Update `cmd/server/main.go`

Replace your current server code with this version:

```go
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

const (
	addr          = "127.0.0.1:9000"
	readTimeout  = 5 * time.Second
	writeTimeout = 5 * time.Second
)

func main() {
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

	if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		log.Printf("set read deadline for %s: %v", remoteAddr, err)
		return
	}

	reader := bufio.NewReader(conn)

	line, err := reader.ReadString('\n')
	if err != nil {
		handleReadError(remoteAddr, err)
		return
	}

	msg := strings.TrimSpace(line)
	log.Printf("received from %s: %q", remoteAddr, msg)

	if err := conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		log.Printf("set write deadline for %s: %v", remoteAddr, err)
		return
	}

	switch msg {
	case "HEALTH":
		_, err = fmt.Fprintln(conn, "OK")
	default:
		_, err = fmt.Fprintln(conn, "UNKNOWN")
	}

	if err != nil {
		handleWriteError(remoteAddr, err)
		return
	}

	log.Printf("closed connection from %s", remoteAddr)
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

func handleWriteError(remoteAddr string, err error) {
	if isTimeout(err) {
		log.Printf("write timeout to %s", remoteAddr)
		return
	}

	log.Printf("write to %s: %v", remoteAddr, err)
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
```

***

## What changed?

### 1. We added timeouts as constants

```go
const (
	addr          = "127.0.0.1:9000"
	readTimeout  = 5 * time.Second
	writeTimeout = 5 * time.Second
)
```

This makes the current behavior explicit:

* client has 5 seconds to send a request
* server has 5 seconds to write the response

***

### 2. We added a read deadline

```go
if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
	log.Printf("set read deadline for %s: %v", remoteAddr, err)
	return
}
```

This means the next read operation must complete before the deadline.

If the client connects and sends nothing, this line:

```go
line, err := reader.ReadString('\n')
```

will eventually return a timeout error.

***

### 3. We added timeout detection

```go
func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
```

This checks whether the returned error behaves like a network timeout.

This is more flexible than a direct type assertion because network errors may be wrapped.

***

### 4. We handle `io.EOF`

```go
if errors.Is(err, io.EOF) {
	log.Printf("client %s closed the connection", remoteAddr)
	return
}
```

`io.EOF` commonly means the peer closed its side of the connection cleanly.

That maps to the Chapter 3 idea of graceful TCP session termination. [\[pkg.go.dev\]](https://pkg.go.dev/net)

***

### 5. We added a write deadline

```go
if err := conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
	log.Printf("set write deadline for %s: %v", remoteAddr, err)
	return
}
```

For this tiny project, writes are unlikely to block. But in real network services, write deadlines are important because clients can become slow or unreachable.

***

# Run it

## Terminal 1 — server

```bash
go run ./cmd/server
```

## Terminal 2 — health checker

```bash
go run ./cmd/check
```

Expected:

```text
healthy: connected in ...
```

Server should still show:

```text
accepted connection from 127.0.0.1:xxxxx
received from 127.0.0.1:xxxxx: "HEALTH"
closed connection from 127.0.0.1:xxxxx
```

***

# Test the timeout behavior

Use `nc` / `netcat` if available:

```bash
nc 127.0.0.1 9000
```

Then do nothing for more than 5 seconds.

Expected server log:

```text
accepted connection from 127.0.0.1:xxxxx
read timeout from 127.0.0.1:xxxxx
```

If `nc` is not available, we can later create a tiny “silent client” in Go, but for now this test is optional.

***

## What we have after Phase 4

The server now:

* accepts multiple clients concurrently
* reads a request
* replies with `OK` for `HEALTH`
* closes connections cleanly
* disconnects idle clients
* detects read timeouts
* protects writes using write deadlines

The health checker still:

* dials with a timeout
* sends `HEALTH`
* expects `OK`
* reports connection duration

***

## Important idea from this phase

TCP is reliable in the sense that it handles ordering, acknowledgments, retransmission, and flow control for us. But **TCP does not automatically make our application logic safe from waiting forever**.

That is why deadlines matter.

Next phase will be the main project feature:

> **Phase 5 — Long-lived heartbeat client/server: `PING` / `PONG`**
