# Sentinel

A concurrent file-scanning REST API written in Go. Sentinel accepts file uploads, inspects their contents against a set of security signatures using a pool of worker goroutines, and returns a structured violation report. It is built entirely on the Go standard library — no external dependencies.

The design is inspired by real-world upload-scanning gateways: a service that sits in front of user-generated content and decides whether a file is safe to accept.

---

## Features

- **Concurrent scanning** — a fixed pool of worker goroutines processes uploads in parallel via a buffered job channel.
- **Signature-based detection** — pluggable regex rule set with severity levels (`low` → `critical`), covering things like the EICAR test string, embedded reverse shells, dynamic code-execution calls, leaked private keys, and auto-executing Office macros.
- **Content hashing** — every upload is fingerprinted with SHA-256 for deduplication and auditing.
- **Per-client rate limiting** — a token-bucket limiter keyed by client IP, with configurable rate and burst.
- **Graceful shutdown** — in-flight scans finish before the process exits; the HTTP server drains cleanly on `SIGINT` / `SIGTERM`.
- **Middleware pipeline** — request logging, panic recovery, and rate limiting composed as standard `net/http` middleware.
- **Tested** — unit tests for the scanner and the rate limiter.
- **Container-ready** — multi-stage `Dockerfile` producing a minimal distroless image.

---

## Architecture

```
                 ┌─────────────┐
   HTTP request  │  middleware │   recover → log → rate-limit
   ──────────────▶│   pipeline │
                 └──────┬──────┘
                        │
                 ┌──────▼──────┐
                 │   router    │   net/http ServeMux (Go 1.22 routing)
                 └──────┬──────┘
                        │
                 ┌──────▼──────┐      ┌──────────────┐
                 │  handlers   │─────▶│   scanner    │
                 │             │      │ (worker pool)│
                 └──────┬──────┘      └──────────────┘
                        │
                 ┌──────▼──────┐
                 │ in-memory   │   thread-safe result store
                 │   store     │
                 └─────────────┘
```

| Package              | Responsibility                                            |
|----------------------|-----------------------------------------------------------|
| `cmd/sentinel`       | Entry point, flag parsing, wiring, graceful shutdown      |
| `internal/api`       | HTTP handlers, middleware, routing                        |
| `internal/scanner`   | Worker-pool scanning engine and signature rules           |
| `internal/ratelimit` | Per-key token-bucket rate limiter                         |
| `internal/store`     | Thread-safe in-memory store for scan results              |

---

## Getting started

### Requirements

- Go 1.22 or newer

### Run

```bash
go run ./cmd/sentinel
```

Or build a binary:

```bash
make build
./bin/sentinel
```

### Configuration

Flags (all optional):

| Flag        | Default | Description                              |
|-------------|---------|------------------------------------------|
| `-addr`     | `:8080` | Address to listen on                     |
| `-workers`  | `4`     | Number of scanner worker goroutines      |
| `-rate`     | `5`     | Allowed requests per second, per client  |
| `-burst`    | `10`    | Burst size, per client                   |

```bash
./bin/sentinel -addr :9090 -workers 8 -rate 20 -burst 40
```

---

## API

### `GET /healthz`

Liveness and uptime check.

```bash
curl localhost:8080/healthz
```

```json
{ "status": "ok", "uptime": "3m12s", "scans_stored": 5 }
```

### `POST /v1/scan`

Upload a file (multipart form, field name `file`) to scan it.

```bash
curl -F "file=@./suspicious.txt" localhost:8080/v1/scan
```

Returns `200 OK` for a clean file or `422 Unprocessable Entity` if signatures matched:

```json
{
  "id": "9f3c1a2b4d5e6f70",
  "filename": "suspicious.txt",
  "result": {
    "sha256": "a1b2c3...",
    "size_bytes": 142,
    "clean": false,
    "violations": [
      {
        "rule_id": "EICAR-001",
        "name": "EICAR test signature",
        "description": "Standard antivirus test string.",
        "severity": "critical"
      }
    ],
    "scanned_at": "2025-06-11T10:30:00Z",
    "duration_ms": 0
  }
}
```

You can verify detection safely with the standard EICAR test string:

```bash
echo 'X5O!P%@AP[4\PZX54(P^)7CC)7}EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*' > eicar.txt
curl -F "file=@eicar.txt" localhost:8080/v1/scan
```

### `GET /v1/scans`

List all stored scan records.

### `GET /v1/scans/{id}`

Fetch one scan record by its ID.

```bash
curl localhost:8080/v1/scans/9f3c1a2b4d5e6f70
```

---

## Testing

```bash
make test
# or
go test ./...
```

---

## Docker

```bash
docker build -t sentinel .
docker run -p 8080:8080 sentinel
```

The final image is built `FROM` distroless and runs as a non-root user.

---

## Design notes

- **Why a worker pool?** Scanning is CPU-bound (regex matching, hashing). A bounded pool keeps resource usage predictable under load instead of spawning one goroutine per request.
- **Why token-bucket rate limiting?** It allows short bursts while still enforcing a long-run average, which fits an upload API where clients may submit a few files at once.
- **Why in-memory storage?** To keep the project dependency-free and easy to run. The `store` package is small and intentionally swappable for a real database (Postgres, Redis) behind the same interface.

---

## Possible extensions

- Persist results to PostgreSQL or Redis behind the existing store interface.
- Load signature rules from an external file or feed instead of embedding them.
- Add `/metrics` for Prometheus scraping.
- Stream large uploads instead of buffering them fully in memory.

---

## License

MIT
