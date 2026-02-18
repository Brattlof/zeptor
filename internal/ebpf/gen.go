//go:build linux

package ebpf

// To generate eBPF bindings, run:
// go generate ./...
//
// This requires:
// - clang and llvm installed
// - Linux kernel headers
// - Running on Linux (not available on Windows/macOS for eBPF compilation)

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -g -Wall" -target bpfel bpf ../../bpf/xdp_router.c -- -I../../bpf/include
