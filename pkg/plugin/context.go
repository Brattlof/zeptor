package plugin

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
)

type PluginContext struct {
	Config     map[string]interface{}
	Logger     *slog.Logger
	HTTPClient *http.Client
	Context    context.Context
	mu         sync.RWMutex
	store      map[string]interface{}
}

func NewPluginContext(ctx context.Context, config map[string]interface{}, logger *slog.Logger) *PluginContext {
	return &PluginContext{
		Config:     config,
		Logger:     logger,
		HTTPClient: http.DefaultClient,
		Context:    ctx,
		store:      make(map[string]interface{}),
	}
}

func (p *PluginContext) Set(key string, value interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.store[key] = value
}

func (p *PluginContext) Get(key string) (interface{}, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	val, ok := p.store[key]
	return val, ok
}

func (p *PluginContext) GetString(key string) (string, bool) {
	val, ok := p.Get(key)
	if !ok {
		return "", false
	}
	s, ok := val.(string)
	return s, ok
}

func (p *PluginContext) GetInt(key string) (int, bool) {
	val, ok := p.Get(key)
	if !ok {
		return 0, false
	}
	switch v := val.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

func (p *PluginContext) GetBool(key string) (bool, bool) {
	val, ok := p.Get(key)
	if !ok {
		return false, false
	}
	b, ok := val.(bool)
	return b, ok
}

func (p *PluginContext) GetStringSlice(key string) ([]string, bool) {
	val, ok := p.Get(key)
	if !ok {
		return nil, false
	}
	switch v := val.(type) {
	case []string:
		return v, true
	case []interface{}:
		result := make([]string, len(v))
		for i, item := range v {
			if s, ok := item.(string); ok {
				result[i] = s
			}
		}
		return result, true
	default:
		return nil, false
	}
}

func (p *PluginContext) ConfigString(key string) string {
	if p.Config == nil {
		return ""
	}
	val, ok := p.Config[key]
	if !ok {
		return ""
	}
	s, ok := val.(string)
	if !ok {
		return ""
	}
	return s
}

func (p *PluginContext) ConfigInt(key string) int {
	if p.Config == nil {
		return 0
	}
	val, ok := p.Config[key]
	if !ok {
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func (p *PluginContext) ConfigBool(key string) bool {
	if p.Config == nil {
		return false
	}
	val, ok := p.Config[key]
	if !ok {
		return false
	}
	b, ok := val.(bool)
	return b && ok
}

func (p *PluginContext) ConfigStringSlice(key string) []string {
	if p.Config == nil {
		return nil
	}
	val, ok := p.Config[key]
	if !ok {
		return nil
	}
	switch v := val.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}
