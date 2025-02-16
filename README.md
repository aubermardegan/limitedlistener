# `limitedlistener` Package

[![Go Reference](https://pkg.go.dev/badge/github.com/aubermardegan/limitedlistener.svg)](https://pkg.go.dev/github.com/aubermardegan/limitedlistener)

The limitedlistener package provides a TCP listener that enforces **global** and **per-connection** **bandwidth limits** on data transfer. It uses the `golang.org/x/time/rate` package to implement rate limiting, ensuring that the bandwidth consumed by all connections stays within the specified limits.

This package is useful for applications that need to control the rate of data transfer across multiple connections, such as file servers, API gateways, or any network service where bandwidth throttling is required.

---

## Features

- **Global Bandwidth Limit:** Enforces a total bandwidth limit across all connections.
- **Per-Connection Bandwidth Limit:** Enforces individual bandwidth limits for each connection.
- **Dynamic Limit Updates:** Allows updating global and per-connection limits at runtime.
- **Thread-Safe:** Uses synchronization mechanisms to safely manage connections and limits.

---

## Installation

To use the `limitedlistener` package, add it to your Go project:

```bash
go get github.com/aubermardegan/limitedlistener
```

---

## Usage

### 1. Creating a Limited Listener

Use `NewLimitedListener` to create a listener with global and per-connection bandwidth limits.

```go
listener, err := net.Listen("tcp", ":8080")
if err != nil {
    log.Fatalf("Failed to create listener: %v", err)
}

// Create a limited listener with a global limit of 1 MB/s and a per-connection limit of 100 KB/s
limitedListener, err := limitedlistener.NewLimitedListener(listener, 1_000_000, 100_000)
if err != nil {
    log.Fatalf("Failed to create limited listener: %v", err)
}
defer limitedListener.Close()
```

### 2. Accepting Connections

Use the Accept method to accept incoming connections. Each connection is wrapped in a LimitedConnection to enforce bandwidth limits.

```go
for {
    conn, err := limitedListener.Accept()
    if err != nil {
        log.Printf("Failed to accept connection: %v", err)
        continue
    }

    go handleConnection(conn)
}
```

### 3. Handling Connections

Read data from the connection while respecting the bandwidth limits.

```go
func handleConnection(conn net.Conn) {
    defer conn.Close()

    buffer := make([]byte, 1024)
    for {
        n, err := conn.Read(buffer)
        if err != nil {
            log.Printf("Connection read error: %v", err)
            return
        }

        log.Printf("Read %d bytes: %s", n, buffer[:n])
    }
}
```

### 4. Updating Limits Dynamically

You can update the global and per-connection bandwidth limits at runtime using the SetLimits method

```go
// Update global limit to 2 MB/s and per-connection limit to 200 KB/s
limitedListener.SetLimits(2_000_000, 200_000)
```

---

## API Reference

### Types

#### LimitedConnection

Wraps a net.Conn and enforces both global and per-connection bandwidth limits on the Read operation.

    Methods:
        Read(b []byte) (int, error): Reads data while respecting bandwidth limits.
        Close() error: Closes the connection and removes it from the listener's connection map.

#### LimitedListener

Wraps a net.Listener and enforces global and per-connection bandwidth limits on all accepted connections.

    Methods:
        Accept() (net.Conn, error): Accepts incoming connections and wraps them with a LimitedConnection.
        SetLimits(global, perConn int): Updates the global and per-connection bandwidth limits.

### Error Handling

The package defines the following errors:

- `ErrLimitOutOfRange`: Returned when bandwidth limits are less than or equal to zero.
- `ErrInvalidLimits`: Returned when the global bandwidth limit is less than the per-connection limit.

---

### Acknowledgments

- Uses the `golang.org/x/time/rate` package for rate limiting.

---

Enjoy using limitedlistener! 🚀
