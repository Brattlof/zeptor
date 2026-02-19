package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/brattlof/zeptor/internal/app/config"
	"github.com/brattlof/zeptor/internal/app/router"
	"github.com/brattlof/zeptor/pkg/plugin"
)

type Server struct {
	config   *config.Config
	router   *router.Router
	mux      *chi.Mux
	registry *plugin.Registry
	logger   *slog.Logger
}

func New(cfg *config.Config, rt *router.Router, registry *plugin.Registry, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{
		config:   cfg,
		router:   rt,
		mux:      chi.NewRouter(),
		registry: registry,
		logger:   logger,
	}
}

func (s *Server) SetupMiddlewares() {
	s.mux.Use(middleware.RequestID)
	s.mux.Use(middleware.RealIP)
	s.mux.Use(middleware.Logger)
	s.mux.Use(middleware.Recoverer)
	s.mux.Use(middleware.Timeout(60 * time.Second))

	if s.config.EBPF.Enabled {
		s.mux.Use(s.eBPFMiddleware)
	}

	if s.registry != nil {
		hooks := s.registry.GetHooks(plugin.HookMiddleware)
		for _, h := range hooks {
			if mh, ok := h.(plugin.MiddlewareHook); ok {
				s.mux.Use(mh.OnMiddleware())
			}
		}

		s.mux.Use(s.pluginRequestHook)
		s.mux.Use(s.pluginResponseHook)
	}

	s.callRouterHooks()
}

func (s *Server) pluginRequestHook(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.registry == nil {
			next.ServeHTTP(w, r)
			return
		}

		hooks := s.registry.GetHooks(plugin.HookRequest)
		for _, h := range hooks {
			if rh, ok := h.(plugin.RequestHook); ok {
				if err := rh.OnRequest(r); err != nil {
					s.logger.Warn("plugin request hook error", "error", err)
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) pluginResponseHook(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.registry == nil {
			next.ServeHTTP(w, r)
			return
		}

		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		hooks := s.registry.GetHooks(plugin.HookResponse)
		for _, h := range hooks {
			if rh, ok := h.(plugin.ResponseHook); ok {
				rh.OnResponse(wrapped, r, wrapped.status)
			}
		}
	})
}

func (s *Server) callRouterHooks() {
	if s.registry == nil {
		return
	}

	hooks := s.registry.GetHooks(plugin.HookRouter)
	for _, h := range hooks {
		if rh, ok := h.(plugin.RouterHook); ok {
			if err := rh.OnRouterInit(&routerAdapter{mux: s.mux}); err != nil {
				s.logger.Error("plugin router hook error", "error", err)
			}
		}
	}
}

func (s *Server) eBPFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func (s *Server) SetupRoutes() {
	for pattern, route := range s.router.StaticRoutes() {
		s.mux.Get(pattern, s.wrapHandler(route))
	}

	for _, route := range s.router.DynamicRoutes() {
		s.mux.Get(route.Pattern, s.wrapHandler(route))
	}

	s.mux.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	s.mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	})
}

func (s *Server) wrapHandler(route *router.Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if route.Handler != nil {
			route.Handler(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Zeptor</title></head>
<body>
<h1>Page: ` + route.Pattern + `</h1>
<p>File: ` + route.File + `</p>
<p><em>Handler not yet implemented</em></p>
</body>
</html>`))
	}
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) Mount(pattern string, handler http.Handler) {
	s.mux.Mount(pattern, handler)
}

func (s *Server) Use(fn func(http.Handler) http.Handler) {
	s.mux.Use(fn)
}

func (s *Server) Get(pattern string, handler http.HandlerFunc) {
	s.mux.Get(pattern, handler)
}

func (s *Server) Post(pattern string, handler http.HandlerFunc) {
	s.mux.Post(pattern, handler)
}

func (s *Server) Put(pattern string, handler http.HandlerFunc) {
	s.mux.Put(pattern, handler)
}

func (s *Server) Delete(pattern string, handler http.HandlerFunc) {
	s.mux.Delete(pattern, handler)
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

type routerAdapter struct {
	mux *chi.Mux
}

func (r *routerAdapter) Get(pattern string, handler http.HandlerFunc) {
	r.mux.Get(pattern, handler)
}

func (r *routerAdapter) Post(pattern string, handler http.HandlerFunc) {
	r.mux.Post(pattern, handler)
}

func (r *routerAdapter) Put(pattern string, handler http.HandlerFunc) {
	r.mux.Put(pattern, handler)
}

func (r *routerAdapter) Delete(pattern string, handler http.HandlerFunc) {
	r.mux.Delete(pattern, handler)
}

func (r *routerAdapter) Use(middleware func(http.Handler) http.Handler) {
	r.mux.Use(middleware)
}

func (r *routerAdapter) Mount(pattern string, handler http.Handler) {
	r.mux.Mount(pattern, handler)
}
