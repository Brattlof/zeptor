package ratelimit

import (
	"net/http"
	"sync"
	"time"

	"github.com/brattlof/zeptor/pkg/plugin"
)

type RateLimitPlugin struct {
	requests map[string]*clientInfo
	mu       sync.RWMutex
	limit    int
	window   time.Duration
	cleanup  time.Duration
	stopChan chan struct{}
	enabled  bool
}

type clientInfo struct {
	count   int
	resetAt time.Time
}

func New() *RateLimitPlugin {
	return &RateLimitPlugin{
		requests: make(map[string]*clientInfo),
		limit:    100,
		window:   time.Minute,
		cleanup:  time.Minute * 5,
		stopChan: make(chan struct{}),
		enabled:  true,
	}
}

func (p *RateLimitPlugin) Name() string        { return "ratelimit" }
func (p *RateLimitPlugin) Version() string     { return "1.0.0" }
func (p *RateLimitPlugin) Description() string { return "IP-based rate limiting middleware" }

func (p *RateLimitPlugin) Init(ctx *plugin.PluginContext) error {
	if limit := ctx.ConfigInt("limit"); limit > 0 {
		p.limit = limit
	}
	if windowSec := ctx.ConfigInt("windowSeconds"); windowSec > 0 {
		p.window = time.Duration(windowSec) * time.Second
	}
	if cleanupSec := ctx.ConfigInt("cleanupSeconds"); cleanupSec > 0 {
		p.cleanup = time.Duration(cleanupSec) * time.Second
	}

	go p.cleanupLoop()
	return nil
}

func (p *RateLimitPlugin) Close() error {
	close(p.stopChan)
	return nil
}

func (p *RateLimitPlugin) Priority() int { return 50 }

func (p *RateLimitPlugin) OnMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !p.enabled {
				next.ServeHTTP(w, r)
				return
			}

			ip := getClientIP(r)
			if !p.allow(ip) {
				w.Header().Set("X-RateLimit-Limit", intToStr(p.limit))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("Rate limit exceeded\n"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (p *RateLimitPlugin) allow(ip string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()

	info, exists := p.requests[ip]
	if !exists || now.After(info.resetAt) {
		p.requests[ip] = &clientInfo{
			count:   1,
			resetAt: now.Add(p.window),
		}
		return true
	}

	if info.count >= p.limit {
		return false
	}

	info.count++
	return true
}

func (p *RateLimitPlugin) cleanupLoop() {
	ticker := time.NewTicker(p.cleanup)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.mu.Lock()
			now := time.Now()
			for ip, info := range p.requests {
				if now.After(info.resetAt) {
					delete(p.requests, ip)
				}
			}
			p.mu.Unlock()
		case <-p.stopChan:
			return
		}
	}
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := splitString(xff, ",")
		if len(parts) > 0 {
			return trimSpace(parts[0])
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	ip := r.RemoteAddr
	if idx := lastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

func Register(registry *plugin.Registry, config map[string]interface{}) error {
	p := New()
	ctx := plugin.NewPluginContext(nil, config, nil)
	if err := p.Init(ctx); err != nil {
		return err
	}
	return registry.Register(p)
}

func intToStr(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return intToStr(n/10) + string(rune('0'+n%10))
}

func splitString(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i:i+1] == sep {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func lastIndex(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
