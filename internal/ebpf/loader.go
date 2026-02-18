package ebpf

import (
	"context"
	"fmt"
	"net"
	"sync"
)

type CacheStats struct {
	TotalRequests uint64
	CacheHits     uint64
	CacheMisses   uint64
	Evictions     uint64
}

type Loader struct {
	mu      sync.RWMutex
	enabled bool
	stats   CacheStats
}

func NewLoader(enabled bool) (*Loader, error) {
	return &Loader{
		enabled: enabled,
	}, nil
}

func (l *Loader) Enabled() bool {
	return l.enabled
}

func (l *Loader) AttachXDP(ifaceName string) error {
	if !l.enabled {
		return nil
	}

	_, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return fmt.Errorf("interface %s not found: %w", ifaceName, err)
	}

	return fmt.Errorf("eBPF XDP attachment not yet implemented - requires Linux kernel 5.4+")
}

func (l *Loader) AttachTC(ifaceName string) error {
	if !l.enabled {
		return nil
	}

	return fmt.Errorf("eBPF TC attachment not yet implemented - requires Linux kernel 5.4+")
}

func (l *Loader) GetStats() CacheStats {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.stats
}

func (l *Loader) UpdateRoute(destIP net.IP, destPort uint16, action uint8) error {
	if !l.enabled {
		return nil
	}

	return fmt.Errorf("eBPF route update not yet implemented")
}

func (l *Loader) CacheGet(key string) ([]byte, bool) {
	if !l.enabled {
		return nil, false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.stats.TotalRequests++
	l.stats.CacheMisses++

	return nil, false
}

func (l *Loader) CacheSet(key string, value []byte, ttlSeconds int) error {
	if !l.enabled {
		return nil
	}

	return fmt.Errorf("eBPF cache not yet implemented")
}

func (l *Loader) CacheInvalidate(pattern string) error {
	if !l.enabled {
		return nil
	}

	return fmt.Errorf("eBPF cache invalidation not yet implemented")
}

func (l *Loader) Close() error {
	return nil
}

func (l *Loader) WaitForEvents(ctx context.Context) error {
	if !l.enabled {
		<-ctx.Done()
		return ctx.Err()
	}

	return fmt.Errorf("eBPF event monitoring not yet implemented")
}
