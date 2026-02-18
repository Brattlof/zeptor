//go:build ignore

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

#define ETH_P_IP 0x0800
#define MAX_ROUTES 256
#define MAX_CACHE_ENTRIES 10000

struct route_key {
    __u32 prefix_len;
    __u32 dest_ip;
    __u16 dest_port;
    __u8 protocol;
};

struct route_value {
    __u8 action;
    __u32 backend_ip;
    __u16 backend_port;
    __u8 padding[5];
};

struct stats_value {
    __u64 packets_total;
    __u64 packets_passed;
    __u64 packets_dropped;
    __u64 cache_hits;
};

struct {
    __uint(type, BPF_MAP_TYPE_LPM_TRIE);
    __uint(key_size, sizeof(struct route_key));
    __uint(value_size, sizeof(struct route_value));
    __uint(max_entries, MAX_ROUTES);
    __uint(map_flags, BPF_F_NO_PREALLOC);
} route_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __type(key, __u32);
    __type(value, struct stats_value);
    __uint(max_entries, 1);
} stats SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __type(key, __u32);
    __type(value, __u32);
    __uint(max_entries, 1);
} config SEC(".maps");

static __always_inline int parse_eth(void *data, void *data_end,
                                     struct ethhdr **eth) {
    struct ethhdr *eth_hdr = data;
    
    if ((void *)(eth_hdr + 1) > data_end)
        return -1;
    
    *eth = eth_hdr;
    return 0;
}

static __always_inline int parse_ip(void *data, void *data_end,
                                    struct ethhdr *eth,
                                    struct iphdr **ip) {
    struct iphdr *ip_hdr = (void *)(eth + 1);
    
    if ((void *)(ip_hdr + 1) > data_end)
        return -1;
    
    if (ip_hdr->version != 4)
        return -1;
    
    *ip = ip_hdr;
    return 0;
}

static __always_inline int parse_tcp(void *data, void *data_end,
                                     struct iphdr *ip,
                                    struct tcphdr **tcp) {
    struct tcphdr *tcp_hdr = (void *)ip + (ip->ihl * 4);
    
    if ((void *)(tcp_hdr + 1) > data_end)
        return -1;
    
    *tcp = tcp_hdr;
    return 0;
}

SEC("xdp")
int xdp_router_prog(struct xdp_md *ctx) {
    void *data_end = (void *)(long)ctx->data_end;
    void *data = (void *)(long)ctx->data;
    
    struct stats_value *stats_val;
    __u32 stats_key = 0;
    
    stats_val = bpf_map_lookup_elem(&stats, &stats_key);
    if (!stats_val)
        return XDP_PASS;
    
    stats_val->packets_total++;
    
    struct ethhdr *eth;
    if (parse_eth(data, data_end, &eth) < 0)
        return XDP_PASS;
    
    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return XDP_PASS;
    
    struct iphdr *ip;
    if (parse_ip(data, data_end, eth, &ip) < 0)
        return XDP_PASS;
    
    if (ip->protocol != IPPROTO_TCP)
        return XDP_PASS;
    
    struct tcphdr *tcp;
    if (parse_tcp(data, data_end, ip, &tcp) < 0)
        return XDP_PASS;
    
    struct route_key key = {
        .prefix_len = 48,
        .dest_ip = ip->daddr,
        .dest_port = tcp->dest,
        .protocol = ip->protocol,
    };
    
    struct route_value *route = bpf_map_lookup_elem(&route_map, &key);
    if (route) {
        switch (route->action) {
        case 1:
            stats_val->packets_dropped++;
            return XDP_DROP;
        case 2:
            stats_val->packets_passed++;
            return XDP_TX;
        default:
            stats_val->packets_passed++;
            return XDP_PASS;
        }
    }
    
    stats_val->packets_passed++;
    return XDP_PASS;
}

char LICENSE[] SEC("license") = "GPL";
