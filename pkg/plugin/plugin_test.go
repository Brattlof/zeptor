package plugin

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"testing"
)

type mockPlugin struct {
	name        string
	version     string
	description string
	initCalled  bool
	closeCalled bool
	initError   error
	closeError  error
}

func (m *mockPlugin) Name() string        { return m.name }
func (m *mockPlugin) Version() string     { return m.version }
func (m *mockPlugin) Description() string { return m.description }
func (m *mockPlugin) Init(ctx *PluginContext) error {
	m.initCalled = true
	return m.initError
}
func (m *mockPlugin) Close() error {
	m.closeCalled = true
	return m.closeError
}

type mockPluginWithHooks struct {
	mockPlugin
	priority       int
	configHook     bool
	routerHook     bool
	middlewareHook bool
	requestHook    bool
	responseHook   bool
	buildHook      bool
	devHook        bool
}

func (m *mockPluginWithHooks) Priority() int { return m.priority }

func (m *mockPluginWithHooks) OnConfigLoad(config map[string]interface{}) error {
	return nil
}

func (m *mockPluginWithHooks) OnRouterInit(router Router) error {
	return nil
}

func (m *mockPluginWithHooks) OnMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler { return next }
}

func (m *mockPluginWithHooks) OnRequest(r *http.Request) error { return nil }

func (m *mockPluginWithHooks) OnResponse(w http.ResponseWriter, r *http.Request, status int) {
}

func (m *mockPluginWithHooks) OnBuildPre() error  { return nil }
func (m *mockPluginWithHooks) OnBuildPost() error { return nil }

func (m *mockPluginWithHooks) OnDevStart() error             { return nil }
func (m *mockPluginWithHooks) OnDevReload(path string) error { return nil }
func (m *mockPluginWithHooks) OnDevStop() error              { return nil }

func TestPluginContext(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := NewPluginContext(context.Background(), map[string]interface{}{
		"string":  "value",
		"int":     42,
		"bool":    true,
		"strings": []string{"a", "b", "c"},
	}, logger)

	t.Run("ConfigString", func(t *testing.T) {
		if got := ctx.ConfigString("string"); got != "value" {
			t.Errorf("ConfigString = %q, want %q", got, "value")
		}
		if got := ctx.ConfigString("missing"); got != "" {
			t.Errorf("ConfigString(missing) = %q, want empty", got)
		}
	})

	t.Run("ConfigInt", func(t *testing.T) {
		if got := ctx.ConfigInt("int"); got != 42 {
			t.Errorf("ConfigInt = %d, want 42", got)
		}
		if got := ctx.ConfigInt("missing"); got != 0 {
			t.Errorf("ConfigInt(missing) = %d, want 0", got)
		}
	})

	t.Run("ConfigBool", func(t *testing.T) {
		if got := ctx.ConfigBool("bool"); !got {
			t.Errorf("ConfigBool = %v, want true", got)
		}
		if got := ctx.ConfigBool("missing"); got {
			t.Errorf("ConfigBool(missing) = %v, want false", got)
		}
	})

	t.Run("ConfigStringSlice", func(t *testing.T) {
		got := ctx.ConfigStringSlice("strings")
		if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
			t.Errorf("ConfigStringSlice = %v, want [a b c]", got)
		}
	})

	t.Run("Store", func(t *testing.T) {
		ctx.Set("key", "stored")
		if got, ok := ctx.GetString("key"); !ok || got != "stored" {
			t.Errorf("GetString = %q, %v, want stored, true", got, ok)
		}
	})
}

func TestRegistry(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	t.Run("Register", func(t *testing.T) {
		p := &mockPlugin{name: "test", version: "1.0.0", description: "test plugin"}
		if err := registry.Register(p); err != nil {
			t.Errorf("Register() error = %v", err)
		}
		if registry.Count() != 1 {
			t.Errorf("Count() = %d, want 1", registry.Count())
		}
	})

	t.Run("RegisterDuplicate", func(t *testing.T) {
		p := &mockPlugin{name: "test", version: "2.0.0"}
		if err := registry.Register(p); err == nil {
			t.Error("Register() should fail for duplicate")
		}
	})

	t.Run("Get", func(t *testing.T) {
		p, ok := registry.Get("test")
		if !ok {
			t.Error("Get() not found")
		}
		if p.Name() != "test" {
			t.Errorf("Get() = %q, want test", p.Name())
		}
	})

	t.Run("Names", func(t *testing.T) {
		names := registry.Names()
		if len(names) != 1 || names[0] != "test" {
			t.Errorf("Names() = %v, want [test]", names)
		}
	})

	t.Run("Unregister", func(t *testing.T) {
		if err := registry.Unregister("test"); err != nil {
			t.Errorf("Unregister() error = %v", err)
		}
		if registry.Count() != 0 {
			t.Errorf("Count() = %d, want 0", registry.Count())
		}
	})
}

func TestRegistryHooks(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	p1 := &mockPluginWithHooks{
		mockPlugin: mockPlugin{name: "plugin1", version: "1.0.0"},
		priority:   10,
		configHook: true,
	}
	p2 := &mockPluginWithHooks{
		mockPlugin: mockPlugin{name: "plugin2", version: "1.0.0"},
		priority:   5,
		configHook: true,
	}

	registry.Register(p1)
	registry.Register(p2)

	hooks := registry.GetHooks(HookConfig)
	if len(hooks) != 2 {
		t.Errorf("GetHooks() = %d hooks, want 2", len(hooks))
	}

	h1 := hooks[0].(Hook)
	h2 := hooks[1].(Hook)
	if h1.Priority() > h2.Priority() {
		t.Errorf("Hooks not sorted by priority: %d > %d", h1.Priority(), h2.Priority())
	}
}

func TestRegistryInfo(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	p := &mockPluginWithHooks{
		mockPlugin:     mockPlugin{name: "test", version: "1.0.0", description: "test plugin"},
		priority:       1,
		middlewareHook: true,
		requestHook:    true,
	}
	registry.Register(p)
	registry.SetConfig("test", map[string]interface{}{"key": "value"})

	info, ok := registry.Info("test")
	if !ok {
		t.Fatal("Info() not found")
	}

	if info.Name != "test" {
		t.Errorf("Name = %q, want test", info.Name)
	}
	if info.Version != "1.0.0" {
		t.Errorf("Version = %q, want 1.0.0", info.Version)
	}
	if info.Config["key"] != "value" {
		t.Errorf("Config[key] = %v, want value", info.Config["key"])
	}

	hasMiddleware := false
	hasRequest := false
	for _, h := range info.Hooks {
		if h == HookMiddleware {
			hasMiddleware = true
		}
		if h == HookRequest {
			hasRequest = true
		}
	}
	if !hasMiddleware {
		t.Error("Hooks missing HookMiddleware")
	}
	if !hasRequest {
		t.Error("Hooks missing HookRequest")
	}
}
