# Phase 2 — Health Check with Dial Timeout

## What changes in this phase?

In Phase 1, the health checker used:

```go
conn, err := net.Dial("tcp", addr)
```

Now we will change it to:

```go
conn, err := net.DialTimeout("tcp", addr, timeout)
```

This prevents the health checker from waiting indefinitely when the server is unavailable, slow to respond, or unreachable.

This maps to Chapter 3’s **Implementing Deadlines / Time-outs** section, especially using Go’s standard library to detect failed or slow TCP connection attempts. Chapter 3 covers `net.DialTimeout`, timeout handling, and checking network errors through `net.Error`. [\[pkg.go.dev\]](https://pkg.go.dev/net)

***

## Phase 2 topic mapping

| Code change                | Chapter 3 topic                         |
| -------------------------- | --------------------------------------- |
| `net.DialTimeout(...)`     | Implementing connection time-outs       |
| `time.Duration` flag       | Configurable timeout behavior           |
| `net.Error` check          | Detecting timeout errors                |
| Measuring elapsed time     | Observing TCP connection latency        |
| Better health-check output | Early detection of unavailable services |

***

# Update `cmd/check/main.go`

Replace your current `cmd/check/main.go` with this:

```go
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9000", "TCP address to check")
	timeout := flag.Duration("timeout", 2*time.Second, "dial timeout")
	flag.Parse()

	start := time.Now()

	conn, err := net.DialTimeout("tcp", *addr, *timeout)
	elapsed := time.Since(start)

	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			log.Fatalf("unhealthy: dial timeout after %s", *timeout)
		}

		log.Fatalf("unhealthy: dial %s failed after %s: %v", *addr, elapsed, err)
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
		fmt.Printf("healthy: connected in %s\n", elapsed)
		return
	}

	fmt.Printf("unhealthy: unexpected response %q\n", response)
}
```

***

## How to run it

### Terminal 1 — start the server

```bash
go run ./cmd/server
```

Expected:

```text
server listening on 127.0.0.1:9000
```

### Terminal 2 — run the health checker

```bash
go run ./cmd/check
```

Expected:

```text
healthy: connected in 1.234ms
```

The exact duration will differ.

***

## Try a custom timeout

```bash
go run ./cmd/check -timeout 500ms
```

Expected:

```text
healthy: connected in ...
```

***

## Try a different address

If nothing is listening on port `9999`:

```bash
go run ./cmd/check -addr 127.0.0.1:9999 -timeout 2s
```

You may see something like:

```text
unhealthy: dial 127.0.0.1:9999 failed after 300µs: dial tcp 127.0.0.1:9999: connect: connection refused
```

That is **not usually a timeout**. It means your machine quickly replied: “nothing is listening on that port.”

***

## Why timeout may not always happen locally

> To strictly get a TIMEOUT error, I can do this -> 

```bash
go run toy/cmd/check/main.go --addr 8.8.8.8:8888 --timeout 1ms
2026/06/29 06:06:17 unhealthy: dial TIMEOUT after 1ms
exit status 1
```

If you test against:

```bash
go run ./cmd/check -addr 127.0.0.1:9999 -timeout 2s
```

you will often get **connection refused immediately**, not a timeout.

> 

A timeout usually happens when the network path silently drops packets or does not respond. Localhost usually responds very quickly, either with success or refusal.

So this distinction matters:

```text
connection refused = target reachable, but no service is listening
timeout            = connection attempt did not complete in time
```

***

## Important code pieces

### 1. Configurable flags

```go
addr := flag.String("addr", "127.0.0.1:9000", "TCP address to check")
timeout := flag.Duration("timeout", 2*time.Second, "dial timeout")
flag.Parse()
```

Now we can run:

```bash
go run ./cmd/check -addr 127.0.0.1:9000 -timeout 1s
```

***

### 2. Timed dial

```go
conn, err := net.DialTimeout("tcp", *addr, *timeout)
```

This is the main Phase 2 change.

Instead of waiting indefinitely, the dial attempt is bounded by the timeout duration.

***

### 3. Timeout error detection

```go
if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
	log.Fatalf("unhealthy: dial timeout after %s", *timeout)
}
```

This checks whether the error is a network timeout error.

Chapter 3 discusses using the `net.Error` interface to distinguish timeout and temporary network errors. [\[pkg.go.dev\]](https://pkg.go.dev/net)

***

### 4. Measuring elapsed connection time

```go
start := time.Now()

conn, err := net.DialTimeout("tcp", *addr, *timeout)
elapsed := time.Since(start)
```

This gives us simple latency visibility:

```text
healthy: connected in 1.4ms
```

This is not a full monitoring system, but it is a nice practical health-check behavior.

***

## What we have after Phase 2

We now have:

```text
server:
  listens on 127.0.0.1:9000
  accepts a health request
  replies OK

check:
  dials with timeout
  sends HEALTH
  expects OK
  reports healthy/unhealthy
  reports connection time
```

Next phase will be:

> **Phase 3 — Make the server concurrent**

Right now, the server handles one connection at a time. In Phase 3, we will update it so every accepted connection is handled in its own goroutine.
