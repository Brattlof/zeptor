package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/brattlof/zeptor/internal/app/config"
	"github.com/brattlof/zeptor/internal/app/router"
)

type Server struct {
	config *config.Config
	router *router.Router
	mux    *chi.Mux
}

func New(cfg *config.Config, rt *router.Router) *Server {
	return &Server{
		config: cfg,
		router: rt,
		mux:    chi.NewRouter(),
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
