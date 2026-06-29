Great. We will treat this as a **single mini-project**, but build it in **phases**, and before each phase I will explain what we are about to implement and which Chapter 3 topics it maps to.

Project name:

# TCP Health Check and Heartbeat Playground

The goal is to build a small TCP playground that teaches the important Chapter 3 ideas through one coherent example: listening, dialing, accepting, deadlines, timeouts, graceful close, unreliable connection detection, and heartbeat-style liveness checks. Chapter 3 covers TCP reliability, TCP sessions, the handshake, sequence numbers, acknowledgments, receive buffers/window sizes, graceful and abrupt termination, Go’s `net.Listen`/`net.Dial`, deadlines, and heartbeat-like failure detection. [\[pkg.go.dev\]](https://pkg.go.dev/net)

***

## Final shape of the project

By the end, we will have:

```text
tcp-playground/
  go.mod
  cmd/
    server/
      main.go
    client/
      main.go
    check/
      main.go
```

Three small programs:

1. **server**  
   A TCP server that accepts clients, replies to health checks, and handles heartbeat messages.

2. **client**  
   A long-lived TCP client that connects to the server, sends heartbeats, receives responses, and exits if the connection becomes unhealthy.

3. **check**  
   A short-lived TCP health checker that attempts a dial with a timeout and reports whether the address is reachable.

We will use only Go’s standard library, especially `net`, `time`, `context`, `bufio`, `io`, and `log`. The Go `net` package provides the portable network I/O primitives we need, including `Dial`, `Listen`, `Accept`, `Conn`, and `Listener`. [\[pkg.go.dev\]](https://pkg.go.dev/github.com/go-lang/go/src/pkg/net)

***

# Phase plan

## Phase 1 — Basic TCP server and one-shot health check

### What we will build

A TCP server that listens on a local address, for example:

```text
127.0.0.1:9000
```

A health-check client connects, sends:

```text
HEALTH
```

The server replies:

```text
OK
```

Then the connection closes.

### Why this phase matters

This gives us the basic shape of TCP programming in Go:

```text
server: listen -> accept -> read -> write -> close
client: dial -> write -> read -> close
```

### Book topic mapping

| Phase   | Book topics                                               |
| ------- | --------------------------------------------------------- |
| Phase 1 | Establishing a TCP connection using Go’s standard library |
| Phase 1 | Binding, listening for, and accepting connections         |
| Phase 1 | Establishing a connection with a server                   |
| Phase 1 | Basic TCP session lifecycle                               |
| Phase 1 | Graceful connection close                                 |

This maps directly to Chapter 3’s practical Go TCP section: `net.Listen` for servers, `Accept` for new client sessions, and `net.Dial` for clients. [\[pkg.go.dev\]](https://pkg.go.dev/net)

***

## Phase 2 — Add dial timeouts to the health checker

### What we will build

The `check` command will attempt to connect with a configurable timeout:

```text
check -addr 127.0.0.1:9000 -timeout 2s
```

Possible outputs:

```text
healthy: connected in 3ms
```

or:

```text
unhealthy: dial timeout after 2s
```

### Why this phase matters

A network program should not hang forever while trying to connect to an unavailable service. This phase makes timeout handling visible and practical.

### Book topic mapping

| Phase   | Book topics                                 |
| ------- | ------------------------------------------- |
| Phase 2 | Implementing time-outs                      |
| Phase 2 | `net.DialTimeout`                           |
| Phase 2 | Detecting failed or slow connections early  |
| Phase 2 | Handling timeout errors through `net.Error` |

Chapter 3 discusses connection timeouts via `net.DialTimeout`, context-based dialing, and identifying timeout-style errors using Go’s networking error interfaces. [\[pkg.go.dev\]](https://pkg.go.dev/net)

***

## Phase 3 — Concurrent server: handle multiple clients

### What we will build

The server will accept multiple connections at the same time. Each connection gets its own goroutine.

This lets multiple clients run:

```text
client A <----\
client B <----- server
client C <----/
```

### Why this phase matters

A TCP server that handles one client at a time is rarely useful. This phase introduces the common Go server pattern:

```text
for {
    conn, err := listener.Accept()
    go handleConn(conn)
}
```

The Go `net` documentation also presents this general server style: create a listener, loop on `Accept`, and handle each connection separately. [\[pkg.go.dev\]](https://pkg.go.dev/github.com/go-lang/go/src/pkg/net)

### Book topic mapping

| Phase   | Book topics                                   |
| ------- | --------------------------------------------- |
| Phase 3 | Listening and accepting multiple TCP sessions |
| Phase 3 | Blocking `Accept`                             |
| Phase 3 | Goroutine-per-connection server pattern       |
| Phase 3 | Properly closing connections to avoid leaks   |

This also prepares us for later Chapter 3 concerns around graceful termination and avoiding connections that remain open unnecessarily. [\[pkg.go.dev\]](https://pkg.go.dev/net)

***

## Phase 4 — Read and write deadlines

### What we will build

The server will apply read deadlines to each client connection.

Example behavior:

```text
client connects
client sends nothing for 10 seconds
server closes the connection
```

The client will also use deadlines when waiting for server responses.

### Why this phase matters

Deadlines prevent a goroutine from blocking forever on a silent or broken connection. This is one of the most important practical lessons in Chapter 3.

### Book topic mapping

| Phase   | Book topics                |
| ------- | -------------------------- |
| Phase 4 | Implementing deadlines     |
| Phase 4 | `SetReadDeadline`          |
| Phase 4 | `SetWriteDeadline`         |
| Phase 4 | `SetDeadline`              |
| Phase 4 | Timeout error handling     |
| Phase 4 | Idle connection protection |

Chapter 3 covers connection deadlines and explains that idle connections can trigger timeout errors, and that deadlines can be advanced or reset as data is received. [\[pkg.go.dev\]](https://pkg.go.dev/net)

***

## Phase 5 — Long-lived heartbeat client/server

### What we will build

The client keeps a connection open and periodically sends:

```text
PING
```

The server replies:

```text
PONG
```

If the client does not receive `PONG` in time, it treats the server as unhealthy and exits.

Example session:

```text
client -> PING
server -> PONG

client -> PING
server -> PONG

client waits...
no PONG received
client: connection unhealthy
```

### Why this phase matters

This is the core of the project. It demonstrates how applications can detect dead or unreliable connections before a user notices something is wrong.

### Book topic mapping

| Phase   | Book topics                                     |
| ------- | ----------------------------------------------- |
| Phase 5 | Heartbeats                                      |
| Phase 5 | Long-lived TCP sessions                         |
| Phase 5 | Advancing deadlines after successful reads      |
| Phase 5 | Detecting unreliable connections early          |
| Phase 5 | Keeping users happy with timeout-aware behavior |

Chapter 3 describes heartbeats as periodic ping-style messages that help keep connections alive and detect failures, and it discusses resetting timers/deadlines when data is received. [\[pkg.go.dev\]](https://pkg.go.dev/net)

***

## Phase 6 — Graceful shutdown and connection cleanup

### What we will build

We will make the server and client respond cleanly to interruption, for example `Ctrl+C`.

Expected behavior:

```text
server: received shutdown signal
server: closing listener
server: closing active connections
server: done
```

The client should also close the connection cleanly.

### Why this phase matters

This phase teaches disciplined cleanup. In network code, forgetting to close a listener or connection can lead to subtle resource leaks and confusing behavior.

### Book topic mapping

| Phase   | Book topics                         |
| ------- | ----------------------------------- |
| Phase 6 | Gracefully terminating TCP sessions |
| Phase 6 | Closing listeners and connections   |
| Phase 6 | Handling `io.EOF`                   |
| Phase 6 | Avoiding leaked sockets             |
| Phase 6 | Avoiding stuck connection states    |

Chapter 3 discusses graceful session termination and less graceful terminations, including the practical requirement to close connections correctly. [\[pkg.go.dev\]](https://pkg.go.dev/net)

***

## Phase 7 — Failure simulation modes

### What we will build

We will add simple flags to simulate bad behavior.

For example, the server could support:

```text
server -mode normal
server -mode slow
server -mode silent
server -mode drop
```

Where:

* `normal`: replies to `PING` with`PONG`
* `slow`: replies, but too slowly
* `silent`: accepts the connection but never replies
* `drop`: closes the connection abruptly after accepting

### Why this phase matters

This makes network failure visible. Rather than only writing happy-path code, we will intentionally create slow, silent, and broken connections and watch how our deadlines and heartbeat logic behave.

### Book topic mapping

| Phase   | Book topics                         |
| ------- | ----------------------------------- |
| Phase 7 | Handling less graceful terminations |
| Phase 7 | Timeout behavior                    |
| Phase 7 | Detecting broken connections        |
| Phase 7 | Observing symptoms of dead peers    |
| Phase 7 | Robust error handling               |

Chapter 3 distinguishes graceful TCP termination from less graceful termination and discusses how timeout handling helps detect unreliable network behavior. [\[pkg.go.dev\]](https://pkg.go.dev/net)

***

# Recommended implementation order

I suggest we implement it in this order:

1. **Phase 1:** Basic server and health checker
2. **Phase 2:** Dial timeout in health checker
3. **Phase 3:** Concurrent server
4. **Phase 4:** Read/write deadlines
5. **Phase 5:** Heartbeat protocol
6. **Phase 6:** Graceful shutdown
7. **Phase 7:** Failure simulation modes

This order keeps each step small, and every step adds one Chapter 3 concept.

***

# Working protocol for us

For each phase, I will follow this pattern:

1. **Explain the idea**
2. **Map it to Chapter 3 topics**
3. **Show the expected behavior**
4. **Then write the code**
5. **Explain the important parts of the code**
6. **Suggest how to run and test it**
7. **Move to the next phase only after that**

So, next, when you are ready, we can start with:

> **Phase 1 — Basic TCP server and one-shot health check**

No heartbeat yet. Just `HEALTH -> OK`, so the foundation is very clear.
