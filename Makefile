.PHONY: all generate build dev clean test bpf install-deps

PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin

all: generate build

install-deps:
	@echo "Installing Go dependencies..."
	go mod download
	@echo "Installing templ CLI..."
	go install github.com/a-h/templ/cmd/templ@latest
	@echo "Installing bpf2go..."
	go install github.com/cilium/ebpf/cmd/bpf2go@latest

generate: install-deps
	@echo "Generating templ files..."
	templ generate
	@echo "Generating eBPF bindings..."
	go generate ./...

build:
	@echo "Building zeptor server..."
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/zeptor ./cmd/zeptor
	@echo "Building zt CLI..."
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/zt ./cmd/zt

install: build
	@echo "Installing to $(BINDIR)..."
	install -m 0755 bin/zeptor $(BINDIR)/zeptor
	install -m 0755 bin/zt $(BINDIR)/zt

dev:
	docker-compose -f docker/docker-compose.yml up --build

dev-nobpf:
	ZEPTOR_EBPF_ENABLED=false docker-compose -f docker/docker-compose.yml up --build zeptor-nobpf

clean:
	rm -rf bin/ .zeptor/ dist/
	docker-compose -f docker/docker-compose.yml down -v 2>/dev/null || true

bpf:
	@echo "Compiling eBPF programs..."
	mkdir -p bpf/out
	clang -O2 -target bpf -c bpf/xdp_router.c -o bpf/out/xdp_router.o -I bpf/include
	clang -O2 -target bpf -c bpf/tc_cache.c -o bpf/out/tc_cache.o -I bpf/include

test:
	go test -v -exec sudo ./...

fmt:
	go fmt ./...
	templ fmt .

lint:
	go vet ./...

.PHONY: help
help:
	@echo "Zeptor Framework - Makefile Commands"
	@echo ""
	@echo "  make install-deps  Install Go dependencies and CLI tools"
	@echo "  make generate      Generate templ and eBPF bindings"
	@echo "  make build         Build zeptor server and zt CLI"
	@echo "  make install       Install binaries to $(BINDIR)"
	@echo "  make dev           Start development server (Docker with eBPF)"
	@echo "  make dev-nobpf     Start development server (no eBPF)"
	@echo "  make clean         Remove build artifacts"
	@echo "  make test          Run tests (requires sudo for eBPF)"
	@echo "  make fmt           Format code"
	@echo "  make lint          Run linter"
