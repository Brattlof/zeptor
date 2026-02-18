#!/bin/bash
set -e

echo "=== eBPF Environment Test ==="
echo ""

echo "1. Kernel version:"
uname -r
echo ""

echo "2. Checking BPF filesystem:"
if mount | grep -q bpf; then
    echo "✓ BPF filesystem mounted"
    mount | grep bpf
else
    echo "✗ BPF filesystem not mounted"
fi
echo ""

echo "3. Checking /sys/fs/bpf:"
if [ -d /sys/fs/bpf ]; then
    echo "✓ /sys/fs/bpf exists"
    ls -la /sys/fs/bpf 2>/dev/null || echo "  (empty or no access)"
else
    echo "✗ /sys/fs/bpf does not exist"
fi
echo ""

echo "4. Checking bpftool:"
if command -v bpftool &> /dev/null; then
    echo "✓ bpftool available"
    echo "  Trying 'bpftool version':"
    bpftool version 2>&1 || echo "  (requires privileges)"
else
    echo "✗ bpftool not available"
fi
echo ""

echo "5. Checking kernel config for BPF:"
if [ -f /boot/config-$(uname -r) ]; then
    echo "BPF config options:"
    grep -E "CONFIG_BPF|CONFIG_NET_CLS_BPF|CONFIG_XDP" /boot/config-$(uname -r) 2>/dev/null | head -10 || echo "  (not found)"
else
    echo "Kernel config not available"
fi
echo ""

echo "6. Testing simple BPF program load:"
cat > /tmp/test_bpf.c << 'EOF'
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

SEC("xdp")
int test_prog(struct xdp_md *ctx) {
    return XDP_PASS;
}

char LICENSE[] SEC("license") = "GPL";
EOF

if clang -O2 -target bpf -c /tmp/test_bpf.c -o /tmp/test_bpf.o 2>&1; then
    echo "✓ BPF program compiled successfully"
    
    if command -v bpftool &> /dev/null; then
        echo "  Attempting to load with bpftool..."
        bpftool prog load /tmp/test_bpf.o /sys/fs/bpf/test_prog 2>&1 || echo "  (load requires privileges)"
    fi
else
    echo "✗ BPF program compilation failed"
fi
echo ""

echo "7. Checking available BPF program types:"
if command -v bpftool &> /dev/null; then
    bpftool feature 2>&1 | grep -A5 "Program types" | head -10 || echo "  (requires privileges)"
fi
echo ""

echo "=== Test Complete ==="
echo ""
echo "If running on Docker Desktop (Windows/macOS), eBPF may have limitations."
echo "For full eBPF support, consider running on native Linux."
