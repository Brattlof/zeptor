package headers

import (
	"net/http"

	"github.com/brattlof/zeptor/pkg/plugin"
)

type HeadersPlugin struct {
	add      map[string]string
	remove   []string
	override map[string]string
	enabled  bool
}

func New() *HeadersPlugin {
	return &HeadersPlugin{
		add:      make(map[string]string),
		remove:   []string{},
		override: make(map[string]string),
		enabled:  true,
	}
}

func (p *HeadersPlugin) Name() string        { return "headers" }
func (p *HeadersPlugin) Version() string     { return "1.0.0" }
func (p *HeadersPlugin) Description() string { return "Add, remove, and override HTTP headers" }

func (p *HeadersPlugin) Init(ctx *plugin.PluginContext) error {
	if add := ctx.Config["add"]; add != nil {
		if m, ok := add.(map[string]interface{}); ok {
			for k, v := range m {
				if s, ok := v.(string); ok {
					p.add[k] = s
				}
			}
		}
	}

	if remove := ctx.ConfigStringSlice("remove"); len(remove) > 0 {
		p.remove = remove
	}

	if override := ctx.Config["override"]; override != nil {
		if m, ok := override.(map[string]interface{}); ok {
			for k, v := range m {
				if s, ok := v.(string); ok {
					p.override[k] = s
				}
			}
		}
	}

	return nil
}

func (p *HeadersPlugin) Close() error {
	return nil
}

func (p *HeadersPlugin) Priority() int { return 200 }

func (p *HeadersPlugin) OnMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !p.enabled {
				next.ServeHTTP(w, r)
				return
			}

			for _, header := range p.remove {
				w.Header().Del(header)
			}

			for key, value := range p.override {
				w.Header().Set(key, value)
			}

			for key, value := range p.add {
				if existing := w.Header().Get(key); existing == "" {
					w.Header().Set(key, value)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (p *HeadersPlugin) OnResponse(w http.ResponseWriter, r *http.Request, status int) {
	for _, header := range p.remove {
		w.Header().Del(header)
	}

	for key, value := range p.override {
		w.Header().Set(key, value)
	}
}

func Register(registry *plugin.Registry, config map[string]interface{}) error {
	p := New()
	ctx := plugin.NewPluginContext(nil, config, nil)
	if err := p.Init(ctx); err != nil {
		return err
	}
	return registry.Register(p)
}
