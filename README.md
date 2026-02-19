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
- [templ](https://github.com/a-h/templ) CLI (`go install github.com/a-h/templ/cmd/templ@latest`)
- Docker (optional, for containerized development)

### Create a New Project

```bash
# Install Zeptor CLI
git clone https://github.com/brattlof/zeptor.git
cd zeptor
make install-deps
make build

# Create a new project
./bin/zt create my-app
cd my-app
../bin/zt dev
```

Open http://localhost:3000

### Project Templates

```bash
zt create my-app                    # Minimal (default)
zt create my-app -t basic           # Basic with routing examples
zt create my-api -t api             # API-only project
zt create my-app -t basic -p 8080   # Custom port
```

### Run an Example

```bash
cd examples/hello-world
../../bin/zt dev
```

## Examples

| Example | Description |
|---------|-------------|
| [hello-world](examples/hello-world/) | Minimal single-page app |
| [basic-routing](examples/basic-routing/) | Static, dynamic, and API routes |
| [with-ebpf](examples/with-ebpf/) | eBPF acceleration enabled |

## Project Structure

Each example is a standalone project:

```
my-zeptor-app/
├── app/                    # Your application
│   ├── page.templ          # Home page (/)
│   ├── about/
│   │   └── page.templ      # About page (/about)
│   ├── slug_/
│   │   └── page.templ      # Dynamic route (/{slug})
│   └── api/
│       └── users/
│           └── route.go    # API endpoint (/api/users)
├── public/                 # Static files (served at /public/*)
└── zeptor.config.yaml      # Configuration
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

## CLI Commands

```bash
# Create a new project
zt create my-app
zt create my-app -t basic -p 8080

# Run development server (from project directory)
zt dev

# Custom port
zt dev -p 8080

# Disable eBPF
zt dev --no-ebpf

# List discovered routes
zt routes
zt routes --json
```

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

logging:
  level: "info"
  format: "text"
```

## Docker Development

```bash
# Without eBPF (works everywhere)
docker-compose -f docker/docker-compose.yml up -d zeptor-nobpf

# With eBPF (Linux only)
docker-compose -f docker/docker-compose.yml --profile ebpf up -d zeptor-ebpf
```

## Requirements

- Go 1.23+
- [templ](https://github.com/a-h/templ) CLI (for template generation)
- Linux kernel 5.4+ (for eBPF features)
- Docker (optional, for containerized development)

## Development

```bash
make test      # Run tests
make lint      # Run linter
make fmt       # Format code
make build     # Build binaries
```

## Status

This project is in **early alpha**. Expect breaking changes.

### Roadmap

- [x] File-based routing with radix tree
- [x] SSR with templ rendering
- [x] Hot module replacement (HMR)
- [x] Dev server with file watching
- [x] `zt create` project scaffolding
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
