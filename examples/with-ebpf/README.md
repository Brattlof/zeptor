# With eBPF

Example with eBPF acceleration enabled.

## Requirements

- Linux kernel 5.4+
- Root privileges (or `CAP_BPF`, `CAP_NET_ADMIN`)

## Usage

```bash
cd examples/with-ebpf
sudo zt dev
```

Or use Docker:

```bash
docker-compose -f docker/docker-compose.yml --profile ebpf up
```

## Features

- **XDP Router**: Kernel-level packet routing
- **TC Cache**: HTTP response caching in eBPF maps
- **Stats API**: View cache hit/miss statistics at `/api/stats`
