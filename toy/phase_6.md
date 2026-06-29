Great. Let us move to **Phase 6 — Graceful shutdown and connection cleanup**.

# Phase 6 — Graceful Shutdown

## What we are adding

Right now, if you press `Ctrl+C`, the server process stops immediately. That is fine for a toy program, but in real network programs you usually want to:

1. Stop accepting new connections.
2. Tell existing connection handlers to stop.
3. Close active connections.
4. Exit cleanly.

This maps to Chapter 3 topics around **gracefully terminating TCP sessions**, **closing listeners/connections**, and avoiding leaked sockets or stuck connections. Chapter 3 discusses graceful and less graceful TCP termination, along with the need to close connections properly in Go network programs. [\[pkg.go.dev\]](https://pkg.go.dev/net)

***

# Phase 6 topic mapping

| Code area                                    | Chapter 3 topic                  |
| -------------------------------------------- | -------------------------------- |
| Closing listener                             | Stop accepting new TCP sessions  |
| Tracking active connections                  | Connection lifecycle management  |
| Closing active connections                   | Graceful TCP session termination |
| Handling `Accept` after close                | Listener cleanup                 |
| Handling `io.EOF` / closed connection errors | Peer/server shutdown behavior    |
| `defer conn.Close()`                         | Avoiding socket leaks            |

***

# Update `cmd/server/main.go`

Replace your server with this version.

```go
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const readTimeout = 6 * time.Second
const writeTimeout = 2 * time.Second

func main() {
	addr := "127.0.0.1:9001"

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	log.Printf("server listening on %s", addr)

	var wg sync.WaitGroup

	var mu sync.Mutex
	activeConns := make(map[net.Conn]struct{})

	shutdown := make(chan struct{})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh

		log.Printf("shutdown signal received")

		close(shutdown)

		if err := listener.Close(); err != nil {
			log.Printf("close listener: %v", err)
		}

		mu.Lock()
		for conn := range activeConns {
			log.Printf("closing active connection from %s", conn.RemoteAddr())
			if err := conn.Close(); err != nil {
				log.Printf("close connection: %v", err)
			}
		}
		mu.Unlock()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-shutdown:
				log.Printf("listener closed; waiting for connection handlers")
				wg.Wait()
				log.Printf("server shutdown complete")
				return
			default:
				log.Printf("accept: %v", err)
				continue
			}
		}

		mu.Lock()
		activeConns[conn] = struct{}{}
		mu.Unlock()

		wg.Add(1)

		go func() {
			defer wg.Done()

			handleConn(conn)

			mu.Lock()
			delete(activeConns, conn)
			mu.Unlock()
		}()
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()
	log.Printf("accepted connection from %s", remoteAddr)

	reader := bufio.NewReader(conn)

	for {
		err := conn.SetReadDeadline(time.Now().Add(readTimeout))
		if err != nil {
			log.Printf("set read deadline for %s: %v", remoteAddr, err)
			return
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			handleReadError(remoteAddr, err)
			return
		}

		msg := strings.TrimSpace(line)
		log.Printf("received from %s: %q", remoteAddr, msg)

		err = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		if err != nil {
			log.Printf("set write deadline for %s: %v", remoteAddr, err)
			return
		}

		switch msg {
		case "HEALTH":
			_, err = fmt.Fprintln(conn, "OK")
			if err != nil {
				handleWriteError(remoteAddr, err)
				return
			}

			log.Printf("health check complete for %s", remoteAddr)
			return

		case "PING":
			_, err = fmt.Fprintln(conn, "PONG")
			if err != nil {
				handleWriteError(remoteAddr, err)
				return
			}

			log.Printf("sent PONG to %s", remoteAddr)

		case "FLOOD":
			err = flood(conn)
			if err != nil {
				handleWriteError(remoteAddr, err)
				return
			}

		default:
			_, err = fmt.Fprintln(conn, "UNKNOWN")
			if err != nil {
				handleWriteError(remoteAddr, err)
				return
			}

			log.Printf("unknown message from %s: %q", remoteAddr, msg)
			return
		}
	}
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func handleWriteError(remoteAddr string, err error) {
	if isTimeout(err) {
		log.Printf("write timeout to %s", remoteAddr)
		return
	}

	if errors.Is(err, net.ErrClosed) {
		log.Printf("connection to %s was closed", remoteAddr)
		return
	}

	log.Printf("write to %s: %v", remoteAddr, err)
}

func handleReadError(remoteAddr string, err error) {
	if errors.Is(err, io.EOF) {
		log.Printf("client %s closed the connection", remoteAddr)
		return
	}

	if errors.Is(err, net.ErrClosed) {
		log.Printf("connection from %s was closed", remoteAddr)
		return
	}

	if isTimeout(err) {
		log.Printf("read timeout from %s", remoteAddr)
		return
	}

	log.Printf("read from %s: %v", remoteAddr, err)
}

func flood(conn net.Conn) error {
	chunk := strings.Repeat("x", 1024*1024)

	for i := 0; i < 10_000; i++ {
		if err := conn.SetWriteDeadline(time.Now().Add(1 * time.Second)); err != nil {
			return err
		}

		_, err := fmt.Fprintln(conn, chunk)
		if err != nil {
			return err
		}
	}

	return nil
}
```

***

# What changed?

## 1. We listen for shutdown signals

```go
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
```

This lets the server respond to:

```bash
Ctrl+C
```

or a termination signal.

***

## 2. We close the listener during shutdown

```go
if err := listener.Close(); err != nil {
	log.Printf("close listener: %v", err)
}
```

Closing the listener unblocks this call:

```go
conn, err := listener.Accept()
```

That allows the main accept loop to exit.

***

## 3. We track active connections

```go
activeConns := make(map[net.Conn]struct{})
```

When a connection is accepted, we add it:

```go
activeConns[conn] = struct{}{}
```

When the handler finishes, we remove it:

```go
delete(activeConns, conn)
```

This lets the shutdown path close any still-active connections.

***

## 4. We wait for handlers to finish

```go
var wg sync.WaitGroup
```

Each connection handler increments the wait group:

```go
wg.Add(1)
```

And marks itself done:

```go
defer wg.Done()
```

During shutdown:

```go
wg.Wait()
```

This gives current handlers a chance to finish cleanup before the server exits.

***

## 5. We handle closed-connection errors more intentionally

```go
if errors.Is(err, net.ErrClosed) {
	log.Printf("connection from %s was closed", remoteAddr)
	return
}
```

This is useful during shutdown because closing active connections can cause blocked reads/writes to return errors.

***

# How to test Phase 6

## Terminal 1 — start server

```bash
go run ./cmd/server
```

Expected:

```text
server listening on 127.0.0.1:9001
```

## Terminal 2 — start heartbeat client

```bash
go run ./cmd/client
```

Expected:

```text
connected to 127.0.0.1:9001
sent PING awaiting PONG
received PONG
...
```

## Terminal 1 — press `Ctrl+C`

Expected server logs:

```text
shutdown signal received
closing active connection from 127.0.0.1:xxxxx
listener closed; waiting for connection handlers
connection from 127.0.0.1:xxxxx was closed
server shutdown complete
```

Expected client logs:

```text
server closed the connection
```

or possibly:

```text
read PONG: read tcp ...: connection reset by peer
```

The exact client-side message can vary slightly by OS and timing.

***

# Important note

This is “graceful enough” for our TCP playground. We are closing active connections directly, so clients will notice the server has gone away. A more application-level graceful shutdown could first send something like:

```text
SERVER_SHUTDOWN
```

before closing each connection.

We can add that later if you want, but for Chapter 3 learning, this version is excellent because it demonstrates listener cleanup, connection cleanup, and blocked read/write cancellation.

***

# Current project status

We now have:

```text
cmd/server
  accepts multiple TCP clients
  supports HEALTH -> OK
  supports PING -> PONG
  uses read/write deadlines
  tracks active connections
  handles Ctrl+C shutdown
  closes listener and active connections cleanly

cmd/check
  one-shot TCP health checker with timeout

cmd/client
  long-lived heartbeat client
  detects missing PONG responses
```

Next phase:

> **Phase 7 — Failure simulation modes**

That will let us intentionally run the server in modes like `normal`, `slow`, `silent`, and `drop` so we can observe how the client behaves under failure.
