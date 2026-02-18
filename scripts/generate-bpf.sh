#!/bin/bash
set -e

echo "Generating eBPF bindings with bpf2go..."

BPF_CLANG="${BPF_CLANG:-clang}"
BPF_CFLAGS="${BPF_CFLAGS:-}"

cd "$(dirname "$0")/.."

mkdir -p internal/ebpf

cd internal/ebpf

cat > gen.go << 'EOF'
//go:build linux

package ebpf

// go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -g -Wall" -target bpfel bpf ../../bpf/xdp_router.c -- -I../../bpf/include
EOF

echo "eBPF generation stubs created. Run 'make generate' to compile."
