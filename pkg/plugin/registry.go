package plugin

import (
	"fmt"
	"log/slog"
	"sort"
	"sync"
)

type Registry struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	configs map[string]map[string]interface{}
	logger  *slog.Logger
}

func NewRegistry(logger *slog.Logger) *Registry {
	return &Registry{
		plugins: make(map[string]Plugin),
		configs: make(map[string]map[string]interface{}),
		logger:  logger,
	}
}

func (r *Registry) Register(p Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := p.Name()
	if name == "" {
		return fmt.Errorf("plugin has empty name")
	}

	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	r.plugins[name] = p
	r.logger.Debug("plugin registered", "name", name, "version", p.Version())
	return nil
}

func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, exists := r.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	if err := p.Close(); err != nil {
		r.logger.Warn("plugin close error", "name", name, "error", err)
	}

	delete(r.plugins, name)
	delete(r.configs, name)
	r.logger.Debug("plugin unregistered", "name", name)
	return nil
}

func (r *Registry) Get(name string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plugins[name]
	return p, ok
}

func (r *Registry) SetConfig(name string, config map[string]interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.configs[name] = config
}

func (r *Registry) GetConfig(name string) map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.configs[name]
}

func (r *Registry) All() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r *Registry) Info(name string) (*Info, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.plugins[name]
	if !ok {
		return nil, false
	}

	info := &Info{
		Name:        p.Name(),
		Version:     p.Version(),
		Description: p.Description(),
		Enabled:     true,
		Config:      r.configs[name],
		Hooks:       r.detectHooks(p),
	}
	return info, true
}

func (r *Registry) AllInfo() []*Info {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]*Info, 0, len(r.plugins))
	for name, p := range r.plugins {
		info := &Info{
			Name:        p.Name(),
			Version:     p.Version(),
			Description: p.Description(),
			Enabled:     true,
			Config:      r.configs[name],
			Hooks:       r.detectHooks(p),
		}
		infos = append(infos, info)
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})
	return infos
}

func (r *Registry) detectHooks(p Plugin) []HookType {
	hooks := []HookType{}
	if _, ok := p.(ConfigHook); ok {
		hooks = append(hooks, HookConfig)
	}
	if _, ok := p.(RouterHook); ok {
		hooks = append(hooks, HookRouter)
	}
	if _, ok := p.(MiddlewareHook); ok {
		hooks = append(hooks, HookMiddleware)
	}
	if _, ok := p.(RequestHook); ok {
		hooks = append(hooks, HookRequest)
	}
	if _, ok := p.(ResponseHook); ok {
		hooks = append(hooks, HookResponse)
	}
	if _, ok := p.(BuildHook); ok {
		hooks = append(hooks, HookBuild)
	}
	if _, ok := p.(DevHook); ok {
		hooks = append(hooks, HookDev)
	}
	return hooks
}

func (r *Registry) GetHooks(hookType HookType) []interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var hooks []interface{}
	for _, p := range r.plugins {
		switch hookType {
		case HookConfig:
			if h, ok := p.(ConfigHook); ok {
				hooks = append(hooks, h)
			}
		case HookRouter:
			if h, ok := p.(RouterHook); ok {
				hooks = append(hooks, h)
			}
		case HookMiddleware:
			if h, ok := p.(MiddlewareHook); ok {
				hooks = append(hooks, h)
			}
		case HookRequest:
			if h, ok := p.(RequestHook); ok {
				hooks = append(hooks, h)
			}
		case HookResponse:
			if h, ok := p.(ResponseHook); ok {
				hooks = append(hooks, h)
			}
		case HookBuild:
			if h, ok := p.(BuildHook); ok {
				hooks = append(hooks, h)
			}
		case HookDev:
			if h, ok := p.(DevHook); ok {
				hooks = append(hooks, h)
			}
		}
	}

	sort.Slice(hooks, func(i, j int) bool {
		pi := hooks[i].(Hook).Priority()
		pj := hooks[j].(Hook).Priority()
		return pi < pj
	})

	return hooks
}

func (r *Registry) CloseAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error
	for name, p := range r.plugins {
		if err := p.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close plugin %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing plugins: %v", errs)
	}
	return nil
}

func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.plugins)
}
