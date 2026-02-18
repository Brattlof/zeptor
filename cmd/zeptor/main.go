package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/brattlof/zeptor/internal/app/config"
	"github.com/brattlof/zeptor/internal/app/router"
	"github.com/brattlof/zeptor/internal/ebpf"
	"github.com/brattlof/zeptor/internal/templates"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	logger := setupLogger(cfg)
	slog.SetDefault(logger)

	rt, err := router.New(cfg.Routing.AppDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating router: %v\n", err)
		os.Exit(1)
	}

	slog.Info("Discovered routes", "count", len(rt.Routes()))
	for _, route := range rt.Routes() {
		slog.Debug("Route", "pattern", route.Pattern, "type", route.Type, "file", route.File)
	}

	var ebpfLoader *ebpf.Loader
	if cfg.EBPF.Enabled {
		ebpfLoader, err = ebpf.NewLoader(true)
		if err != nil {
			slog.Warn("eBPF loader initialization failed", "error", err)
		} else {
			slog.Info("eBPF acceleration enabled")
		}
	}

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(router.ParamsMiddleware)

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			rctx := chi.RouteContext(req.Context())
			params := make(map[string]string)
			for i, key := range rctx.URLParams.Keys {
				params[key] = rctx.URLParams.Values[i]
			}
			router.SetParams(req, params)
			next.ServeHTTP(w, req)
		})
	})

	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := templates.HomePage().Render(req.Context(), w); err != nil {
			slog.Error("Failed to render home page", "error", err)
		}
	})

	r.Get("/about", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := templates.AboutPage().Render(req.Context(), w); err != nil {
			slog.Error("Failed to render about page", "error", err)
		}
	})

	r.Get("/{slug}", func(w http.ResponseWriter, req *http.Request) {
		slug := chi.URLParam(req, "slug")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := templates.DynamicPage(slug).Render(req.Context(), w); err != nil {
			slog.Error("Failed to render dynamic page", "error", err)
		}
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		stats := "{}"
		if ebpfLoader != nil {
			s := ebpfLoader.GetStats()
			stats = fmt.Sprintf(`,"ebpf":{"requests":%d,"hits":%d,"misses":%d}`,
				s.TotalRequests, s.CacheHits, s.CacheMisses)
		}
		fmt.Fprintf(w, `{"status":"ok","version":"%s"%s}`, version, stats)
	})

	r.Get("/api/routes", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"routes":[`)
		routes := rt.Routes()
		for i, route := range routes {
			if i > 0 {
				w.Write([]byte(","))
			}
			fmt.Fprintf(w, `{"pattern":"%s","type":"%d","file":"%s","dynamic":%v}`,
				route.Pattern, route.Type, route.File, route.IsDynamic)
		}
		w.Write([]byte(`]}`))
	})

	r.Get("/api/stats", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if ebpfLoader != nil {
			stats := ebpfLoader.GetStats()
			fmt.Fprintf(w, `{"ebpf":{"requests":%d,"hits":%d,"misses":%d}}`,
				stats.TotalRequests, stats.CacheHits, stats.CacheMisses)
		} else {
			w.Write([]byte(`{"ebpf":{"enabled":false}}`))
		}
	})

	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		if err := templates.Error404().Render(req.Context(), w); err != nil {
			slog.Error("Failed to render 404 page", "error", err)
		}
	})

	srv := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("Starting Zeptor server",
			"addr", cfg.Addr(),
			"version", version,
			"ebpf", cfg.EBPF.Enabled,
			"routes", len(rt.Routes()),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	if ebpfLoader != nil {
		ebpfLoader.Close()
	}

	slog.Info("Server exited")
}

func setupLogger(cfg *config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	switch cfg.Logging.Level {
	case "debug":
		opts.Level = slog.LevelDebug
	case "warn":
		opts.Level = slog.LevelWarn
	case "error":
		opts.Level = slog.LevelError
	}

	var handler slog.Handler
	if cfg.Logging.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
