package basicauth

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"

	"github.com/brattlof/zeptor/pkg/plugin"
)

type BasicAuthPlugin struct {
	users   map[string]string
	paths   []string
	realm   string
	enabled bool
}

func New() *BasicAuthPlugin {
	return &BasicAuthPlugin{
		users:   make(map[string]string),
		paths:   []string{},
		realm:   "Restricted",
		enabled: true,
	}
}

func (p *BasicAuthPlugin) Name() string        { return "basicauth" }
func (p *BasicAuthPlugin) Version() string     { return "1.0.0" }
func (p *BasicAuthPlugin) Description() string { return "HTTP Basic Authentication middleware" }

func (p *BasicAuthPlugin) Init(ctx *plugin.PluginContext) error {
	for _, user := range ctx.ConfigStringSlice("users") {
		parts := strings.SplitN(user, ":", 2)
		if len(parts) == 2 {
			p.users[parts[0]] = parts[1]
		}
	}

	paths := ctx.ConfigStringSlice("paths")
	if len(paths) > 0 {
		p.paths = paths
	}

	if realm := ctx.ConfigString("realm"); realm != "" {
		p.realm = realm
	}

	return nil
}

func (p *BasicAuthPlugin) Close() error {
	return nil
}

func (p *BasicAuthPlugin) Priority() int { return 100 }

func (p *BasicAuthPlugin) OnMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !p.enabled {
				next.ServeHTTP(w, r)
				return
			}

			if !p.shouldProtect(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			user, pass, ok := r.BasicAuth()
			if !ok {
				p.unauthorized(w)
				return
			}

			expectedPass, exists := p.users[user]
			if !exists {
				p.unauthorized(w)
				return
			}

			if subtle.ConstantTimeCompare([]byte(pass), []byte(expectedPass)) != 1 {
				p.unauthorized(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (p *BasicAuthPlugin) shouldProtect(path string) bool {
	if len(p.paths) == 0 {
		return true
	}
	for _, prefix := range p.paths {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func (p *BasicAuthPlugin) unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, p.realm))
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("Unauthorized\n"))
}

func Register(registry *plugin.Registry, config map[string]interface{}) error {
	p := New()
	ctx := plugin.NewPluginContext(nil, config, nil)
	if err := p.Init(ctx); err != nil {
		return err
	}
	return registry.Register(p)
}
