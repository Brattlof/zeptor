package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	goplugin "plugin"
	"sync"
)

type PluginOptions map[string]interface{}

type Loader struct {
	registry  *Registry
	pluginDir string
	logger    *slog.Logger
	mu        sync.RWMutex
	loaded    map[string]string
}

func NewLoader(registry *Registry, pluginDir string, logger *slog.Logger) *Loader {
	return &Loader{
		registry:  registry,
		pluginDir: pluginDir,
		logger:    logger,
		loaded:    make(map[string]string),
	}
}

func (l *Loader) LoadFromConfig(ctx context.Context, enabled []string, configs map[string]PluginOptions) error {
	for _, name := range enabled {
		config := configs[name]
		if config == nil {
			config = make(PluginOptions)
		}

		if err := l.LoadPlugin(ctx, name, config); err != nil {
			return fmt.Errorf("load plugin %s: %w", name, err)
		}
	}
	return nil
}

func (l *Loader) LoadPlugin(ctx context.Context, name string, config PluginOptions) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.loaded[name]; exists {
		return fmt.Errorf("plugin %s already loaded", name)
	}

	pluginPath := filepath.Join(l.pluginDir, name+".so")

	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return fmt.Errorf("plugin file not found: %s", pluginPath)
	}

	p, err := goplugin.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("open plugin: %w", err)
	}

	sym, err := p.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("lookup Plugin symbol: %w", err)
	}

	pluginInstance, ok := sym.(Plugin)
	if !ok {
		return fmt.Errorf("plugin does not implement Plugin interface")
	}

	pluginCtx := NewPluginContext(ctx, config, l.logger)

	if err := pluginInstance.Init(pluginCtx); err != nil {
		return fmt.Errorf("init plugin: %w", err)
	}

	if err := l.registry.Register(pluginInstance); err != nil {
		pluginInstance.Close()
		return fmt.Errorf("register plugin: %w", err)
	}

	l.registry.SetConfig(name, config)
	l.loaded[name] = pluginPath

	l.logger.Info("plugin loaded", "name", name, "version", pluginInstance.Version())
	return nil
}

func (l *Loader) UnloadPlugin(name string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.loaded[name]; !exists {
		return fmt.Errorf("plugin %s not loaded", name)
	}

	if err := l.registry.Unregister(name); err != nil {
		return err
	}

	delete(l.loaded, name)
	l.logger.Info("plugin unloaded", "name", name)
	return nil
}

func (l *Loader) LoadedPlugins() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	names := make([]string, 0, len(l.loaded))
	for name := range l.loaded {
		names = append(names, name)
	}
	return names
}

func (l *Loader) DiscoverPlugins() ([]string, error) {
	if _, err := os.Stat(l.pluginDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(l.pluginDir)
	if err != nil {
		return nil, fmt.Errorf("read plugin dir: %w", err)
	}

	var plugins []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".so" {
			name := entry.Name()[:len(entry.Name())-3]
			plugins = append(plugins, name)
		}
	}

	return plugins, nil
}

func (l *Loader) Close() error {
	return l.registry.CloseAll()
}
