//go:build ignore

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>
#include <string.h>

#define ETH_P_IP 0x0800
#define HTTP_GET 0x47455420
#define HTTP_POST 0x504f5354
#define MAX_URL_LEN 192
#define MAX_RESP_LEN 3072
#define CACHE_TTL_NS 60000000000ULL

struct cache_key {
    __u64 hash;
    __u16 method;
    __u16 port;
    __u32 padding;
};

struct cache_value {
    __u64 timestamp;
    __u32 status;
    __u16 content_len;
    __u8 content_type;
    __u8 body[MAX_RESP_LEN];
};

struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, 10000);
    __type(key, struct cache_key);
    __type(value, struct cache_value);
    __uint(map_flags, BPF_F_NO_COMMON_LRU);
} http_cache SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __type(key, __u32);
    __type(value, __u64);
    __uint(max_entries, 4);
} stats SEC(".maps");

#define FNV_OFFSET 0xcbf29ce484222325ULL
#define FNV_PRIME 0x100000001b3ULL

static __always_inline __u64 fnv1a_hash(const void *data, __u32 len) {
    __u64 hash = FNV_OFFSET;
    const __u8 *bytes = data;
    
    #pragma unroll
    for (int i = 0; i < 256 && i < len; i++) {
        if (i >= len)
            break;
        hash ^= bytes[i];
        hash *= FNV_PRIME;
    }
    return hash;
}

static __always_inline int parse_http_start(void *data, void *data_end,
                                            struct tcphdr *tcp,
                                            void **http_start,
                                            __u32 *method) {
    void *start = (void *)tcp + (tcp->doff * 4);
    
    if (start + 4 > data_end)
        return -1;
    
    *http_start = start;
    __u32 *m = (__u32 *)start;
    *method = *m;
    
    return 0;
}

SEC("tc")
int tc_http_cache(struct __sk_buff *skb) {
    void *data_end = (void *)(long)skb->data_end;
    void *data = (void *)(long)skb->data;
    
    __u32 stats_key = 0;
    __u64 *count = bpf_map_lookup_elem(&stats, &stats_key);
    if (count)
        (*count)++;
    
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return TC_ACT_OK;
    
    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return TC_ACT_OK;
    
    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        return TC_ACT_OK;
    
    if (ip->protocol != IPPROTO_TCP)
        return TC_ACT_OK;
    
    struct tcphdr *tcp = (void *)ip + (ip->ihl * 4);
    if ((void *)(tcp + 1) > data_end)
        return TC_ACT_OK;
    
    __u16 dport = bpf_ntohs(tcp->dest);
    if (dport != 80 && dport != 8080 && dport != 3000)
        return TC_ACT_OK;
    
    void *http_start;
    __u32 method_raw;
    if (parse_http_start(data, data_end, tcp, &http_start, &method_raw) < 0)
        return TC_ACT_OK;
    
    __u16 method = 0;
    if (method_raw == HTTP_GET)
        method = 1;
    else if (method_raw == HTTP_POST)
        method = 2;
    
    if (method != 1)
        return TC_ACT_OK;
    
    char *url_start = (char *)http_start + 4;
    int url_len = 0;
    
    #pragma unroll
    for (int i = 0; i < MAX_URL_LEN; i++) {
        if ((void *)(url_start + i) >= data_end)
            break;
        char c = url_start[i];
        if (c == ' ' || c == '\r' || c == '\n')
            break;
        url_len++;
    }
    
    if (url_len == 0)
        return TC_ACT_OK;
    
    struct cache_key key = {
        .method = method,
        .port = dport,
        .hash = fnv1a_hash(url_start, url_len),
    };
    
    struct cache_value *cached = bpf_map_lookup_elem(&http_cache, &key);
    if (cached) {
        __u64 now = bpf_ktime_get_ns();
        if (now - cached->timestamp < CACHE_TTL_NS) {
            __u32 hit_key = 1;
            __u64 *hits = bpf_map_lookup_elem(&stats, &hit_key);
            if (hits)
                (*hits)++;
            return TC_ACT_OK;
        }
    }
    
    return TC_ACT_OK;
}

char LICENSE[] SEC("license") = "GPL";
