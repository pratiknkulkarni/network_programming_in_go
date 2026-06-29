Great. Moving to **Phase 5 — Long-lived heartbeat client/server**.

# Phase 5 — Heartbeat Client and Server

## What we are adding

Until now, the health checker is short-lived:

```text
connect -> HEALTH -> OK -> close
```

Now we add a long-lived client:

```text
connect
PING -> PONG
PING -> PONG
PING -> PONG
...
```

If the client does not receive `PONG` within a deadline, it treats the connection as unhealthy and exits.

This maps to Chapter 3 topics around **heartbeats**, **long-lived TCP sessions**, **deadlines**, and **early detection of unreliable connections**. Chapter 3 discusses heartbeat-style ping messages, resetting/advancing deadlines after receiving data, and using deadlines to avoid waiting forever on silent peers. [\[pkg.go.dev\]](https://pkg.go.dev/net)

***

# Phase 5 topic mapping

| Code area                  | Chapter 3 topic                      |
| -------------------------- | ------------------------------------ |
| Persistent connection      | Working with TCP sessions            |
| `PING` / `PONG` protocol   | Heartbeats                           |
| Client read deadline       | Detecting dead or silent connections |
| Server read deadline reset | Keeping active sessions alive        |
| `io.EOF` handling          | Graceful session termination         |
| `net.Error.Timeout()`      | Timeout error handling               |

***

# Add new command: `cmd/client/main.go`

Create this file:

```text
cmd/client/main.go
```

```go
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9000", "TCP server address")
	dialTimeout := flag.Duration("dial-timeout", 2*time.Second, "dial timeout")
	interval := flag.Duration("interval", 2*time.Second, "heartbeat interval")
	responseTimeout := flag.Duration("response-timeout", 3*time.Second, "PONG response timeout")
	flag.Parse()

	conn, err := net.DialTimeout("tcp", *addr, *dialTimeout)
	if err != nil {
		log.Fatalf("dial %s: %v", *addr, err)
	}
	defer conn.Close()

	log.Printf("connected to %s", *addr)

	reader := bufio.NewReader(conn)
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		_, err := fmt.Fprintln(conn, "PING")
		if err != nil {
			log.Fatalf("send PING: %v", err)
		}

		log.Printf("sent PING")

		if err := conn.SetReadDeadline(time.Now().Add(*responseTimeout)); err != nil {
			log.Fatalf("set read deadline: %v", err)
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Fatalf("server closed the connection")
			}

			if isTimeout(err) {
				log.Fatalf("unhealthy: no PONG within %s", *responseTimeout)
			}

			log.Fatalf("read PONG: %v", err)
		}

		msg := strings.TrimSpace(line)
		if msg != "PONG" {
			log.Fatalf("unhealthy: expected PONG, got %q", msg)
		}

		log.Printf("received PONG")

		<-ticker.C
	}
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
```

***

# Update `cmd/server/main.go`

Replace your current server with this version. It keeps `HEALTH -> OK`, but now also supports long-lived `PING -> PONG`.

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
	readTimeout  = 10 * time.Second
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

	reader := bufio.NewReader(conn)

	for {
		if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
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

		if err := conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
			log.Printf("set write deadline for %s: %v", remoteAddr, err)
			return
		}

		switch msg {
		case "HEALTH":
			if _, err := fmt.Fprintln(conn, "OK"); err != nil {
				handleWriteError(remoteAddr, err)
				return
			}

			log.Printf("health check complete for %s", remoteAddr)
			return

		case "PING":
			if _, err := fmt.Fprintln(conn, "PONG"); err != nil {
				handleWriteError(remoteAddr, err)
				return
			}

			log.Printf("sent PONG to %s", remoteAddr)

		default:
			if _, err := fmt.Fprintln(conn, "UNKNOWN"); err != nil {
				handleWriteError(remoteAddr, err)
				return
			}

			log.Printf("unknown message from %s: %q", remoteAddr, msg)
			return
		}
	}
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

# How to run it

## Terminal 1 — start server

```bash
go run ./cmd/server
```

Expected:

```text
server listening on 127.0.0.1:9000
```

***

## Terminal 2 — existing one-shot health check still works

```bash
go run ./cmd/check
```

Expected:

```text
healthy: connected in ...
```

The server should handle `HEALTH` and then close that connection.

***

## Terminal 3 — start heartbeat client

```bash
go run ./cmd/client
```

Expected client logs:

```text
connected to 127.0.0.1:9000
sent PING
received PONG
sent PING
received PONG
sent PING
received PONG
```

Expected server logs:

```text
accepted connection from 127.0.0.1:xxxxx
received from 127.0.0.1:xxxxx: "PING"
sent PONG to 127.0.0.1:xxxxx
received from 127.0.0.1:xxxxx: "PING"
sent PONG to 127.0.0.1:xxxxx
```

***

# What changed in the server?

Previously, the server handled one message and returned.

Now it has a loop:

```go
for {
    line, err := reader.ReadString('\n')
    ...
}
```

This allows the same TCP connection to process many messages:

```text
PING
PING
PING
```

That is our long-lived TCP session.

***

# Important detail: deadline is refreshed every loop

This part matters:

```go
if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
	log.Printf("set read deadline for %s: %v", remoteAddr, err)
	return
}
```

We set the read deadline before every read.

That means:

```text
each valid incoming message gives the client another 10 seconds
```

This is the “advance/reset the deadline after activity” idea from Chapter 3’s heartbeat discussion. [\[pkg.go.dev\]](https://pkg.go.dev/net)

***

# What changed in the client?

The client repeatedly sends:

```go
fmt.Fprintln(conn, "PING")
```

Then it sets a deadline before reading:

```go
conn.SetReadDeadline(time.Now().Add(*responseTimeout))
```

Then waits for:

```text
PONG
```

If no `PONG` arrives before the deadline, the client exits:

```go
log.Fatalf("unhealthy: no PONG within %s", *responseTimeout)
```

***

# Quick failure test

Run the server and client normally.

Then stop the server with `Ctrl+C`.

The client should eventually report something like:

```text
server closed the connection
```

or:

```text
unhealthy: no PONG within 3s
```

The exact error depends on timing and OS behavior.

***

# Current project status

We now have:

```text
cmd/server
  supports HEALTH -> OK
  supports long-lived PING -> PONG
  uses read/write deadlines
  handles multiple clients concurrently

cmd/check
  one-shot health checker with dial timeout

cmd/client
  long-lived heartbeat client
  detects missing PONG responses
```

Next phase:

> **Phase 6 — Graceful shutdown and connection cleanup**

That will make the server and client respond cleanly to `Ctrl+C` instead of just exiting abruptly.
