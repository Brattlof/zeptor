package plugin

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestLoader_DiscoverPlugins(t *testing.T) {
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "plugins")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)
	loader := NewLoader(registry, pluginDir, logger)

	t.Run("empty directory", func(t *testing.T) {
		plugins, err := loader.DiscoverPlugins()
		if err != nil {
			t.Errorf("DiscoverPlugins() error = %v", err)
		}
		if len(plugins) != 0 {
			t.Errorf("DiscoverPlugins() = %v, want empty", plugins)
		}
	})

	t.Run("with .so files", func(t *testing.T) {
		dummyFiles := []string{"test1.so", "test2.so", "readme.txt"}
		for _, f := range dummyFiles {
			path := filepath.Join(pluginDir, f)
			if err := os.WriteFile(path, []byte{}, 0644); err != nil {
				t.Fatal(err)
			}
		}

		plugins, err := loader.DiscoverPlugins()
		if err != nil {
			t.Errorf("DiscoverPlugins() error = %v", err)
		}
		if len(plugins) != 2 {
			t.Errorf("DiscoverPlugins() found %d plugins, want 2", len(plugins))
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		loader := NewLoader(registry, "/nonexistent/path", logger)
		plugins, err := loader.DiscoverPlugins()
		if err != nil {
			t.Errorf("DiscoverPlugins() error = %v", err)
		}
		if plugins != nil {
			t.Errorf("DiscoverPlugins() = %v, want nil", plugins)
		}
	})
}

func TestLoader_LoadedPlugins(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)
	loader := NewLoader(registry, "./plugins", logger)

	loaded := loader.LoadedPlugins()
	if len(loaded) != 0 {
		t.Errorf("LoadedPlugins() = %v, want empty", loaded)
	}
}

func TestLoader_LoadFromConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)
	loader := NewLoader(registry, "/nonexistent", logger)

	ctx := context.Background()
	configs := map[string]PluginOptions{
		"test-plugin": {"key": "value"},
	}

	err := loader.LoadFromConfig(ctx, []string{"test-plugin"}, configs)
	if err == nil {
		t.Error("LoadFromConfig() should fail for nonexistent plugin")
	}
}

func TestLoader_UnloadPlugin(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)
	loader := NewLoader(registry, "./plugins", logger)

	t.Run("unload non-existent plugin", func(t *testing.T) {
		err := loader.UnloadPlugin("nonexistent")
		if err == nil {
			t.Error("UnloadPlugin() should fail for non-existent plugin")
		}
	})
}

func TestPluginOptions(t *testing.T) {
	opts := PluginOptions{
		"string":  "value",
		"int":     42,
		"bool":    true,
		"strings": []string{"a", "b"},
	}

	if opts["string"] != "value" {
		t.Errorf("opts[string] = %v, want value", opts["string"])
	}
	if opts["int"] != 42 {
		t.Errorf("opts[int] = %v, want 42", opts["int"])
	}
	if !opts["bool"].(bool) {
		t.Error("opts[bool] should be true")
	}
}
