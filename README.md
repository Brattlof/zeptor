# Zeptor

[![Go Reference](https://pkg.go.dev/badge/github.com/brattlof/zeptor.svg)](https://pkg.go.dev/github.com/brattlof/zeptor)
[![Go Report Card](https://goreportcard.com/badge/github.com/brattlof/zeptor)](https://goreportcard.com/report/github.com/brattlof/zeptor)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A Next.js-like Go framework with eBPF acceleration.

> ⚠️ **Work in Progress** - This project is in early development. APIs may change without notice.

## Features

- 📁 **File-based routing** - Automatic route discovery from `app/` directory
- ⚡ **eBPF acceleration** - Kernel-level routing and caching (Linux only)
- 🎨 **Type-safe templates** - Full type safety with [templ](https://github.com/a-h/templ)
- 🔀 **Dynamic routes** - Support for `{slug}` style parameters
- 🔄 **Hot reload** - Instant browser refresh during development
- 🚀 **Fast** - Radix tree router with O(k) lookups

## Quick Start

### Prerequisites

- Go 1.23+
- Docker (for eBPF development on non-Linux)

### Installation

```bash
# Clone the repository
git clone https://github.com/brattlof/zeptor.git
cd zeptor

# Install dependencies
make install-deps

# Build
make build
```

### Run Development Server

```bash
# With hot reload (recommended)
./bin/zt dev

# On a different port
./bin/zt dev -p 8080

# Without eBPF
./bin/zt dev --no-ebpf

# Docker (for eBPF on non-Linux)
docker-compose -f docker/docker-compose.yml up -d zeptor-nobpf
```

### CLI Commands

```bash
# List discovered routes
./bin/zt routes
./bin/zt routes --json

# Start development server
./bin/zt dev

# Build for production
./bin/zt build
```

## Project Structure

```
zeptor/
├── app/                    # Your application
│   ├── page.templ          # Home page (/)
│   ├── about/
│   │   └── page.templ      # About page (/about)
│   ├── slug_/
│   │   └── page.templ      # Dynamic route (/{slug})
│   └── api/
│       └── users/
│           └── route.go    # API endpoint (/api/users)
├── bpf/                    # eBPF programs
│   ├── xdp_router.c        # XDP packet routing
│   └── tc_cache.c          # TC HTTP caching
├── cmd/
│   ├── zeptor/             # Server binary
│   └── zt/                 # CLI tool
├── internal/               # Framework internals
│   ├── app/
│   │   ├── config/
│   │   ├── router/
│   │   └── render/
│   ├── ebpf/
│   └── dev/                # Dev server & HMR
├── public/                 # Static files (served at /public/*)
└── docker/                 # Docker configuration
```

## Routing

### File-based Routes

| File Path | URL Pattern | Description |
|-----------|-------------|-------------|
| `app/page.templ` | `/` | Home page |
| `app/about/page.templ` | `/about` | Static route |
| `app/blog/slug_/page.templ` | `/blog/{slug}` | Dynamic route |
| `app/api/users/route.go` | `/api/users` | API endpoint |

> **Note:** Dynamic route directories use `slug_` suffix (e.g., `slug_` → `{slug}`) for Go package compatibility.

### API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Health check |
| `GET /api/routes` | List discovered routes |
| `GET /api/stats` | eBPF cache statistics |

## Configuration

```yaml
# zeptor.config.yaml
app:
  port: 3000
  host: "0.0.0.0"

routing:
  appDir: "./app"
  publicDir: "./public"

ebpf:
  enabled: true
  interface: "eth0"
  cacheSize: 10000
  cacheTTLSec: 60

rendering:
  mode: "ssr"  # ssr, ssg, or isr
```

## Requirements

- Go 1.23+
- Linux kernel 5.4+ (for eBPF features)
- Docker (optional, for containerized development)

## Development

```bash
# Run tests
make test

# Run linter
make lint

# Format code
make fmt

# Build Docker images
make dev
```

## Status

This project is in **early alpha**. Expect breaking changes.

### Roadmap

- [x] File-based routing with radix tree
- [x] SSR with templ rendering
- [x] Hot module replacement (HMR)
- [x] Dev server with file watching
- [ ] Full eBPF integration (XDP + TC)
- [ ] SSG build process
- [ ] Middleware system
- [ ] Plugin architecture

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[MIT](LICENSE)

## Acknowledgments

- [cilium/ebpf](https://github.com/cilium/ebpf) - Pure Go eBPF library
- [a-h/templ](https://github.com/a-h/templ) - Type-safe HTML templates
- [go-chi/chi](https://github.com/go-chi/chi) - Lightweight HTTP router
