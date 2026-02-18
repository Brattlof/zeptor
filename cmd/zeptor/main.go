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

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(renderHomePage()))
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

	rt.Mount(r)

	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(render404Page()))
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

func renderHomePage() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Zeptor - eBPF-accelerated Go Framework</title>
	<script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-900 text-white min-h-screen">
	<nav class="bg-gray-800 border-b border-gray-700">
		<div class="container mx-auto px-4 py-3 flex items-center justify-between">
			<span class="text-xl font-bold text-blue-400">⚡ Zeptor</span>
			<div class="flex gap-4">
				<a href="/" class="hover:text-blue-400">Home</a>
				<a href="/about" class="hover:text-blue-400">About</a>
				<a href="/api/routes" class="hover:text-blue-400">Routes</a>
				<a href="/health" class="hover:text-blue-400">Health</a>
			</div>
		</div>
	</nav>

	<main class="container mx-auto px-4 py-12">
		<div class="text-center mb-12">
			<h1 class="text-5xl font-bold mb-4">Welcome to Zeptor</h1>
			<p class="text-xl text-gray-400">A Next.js-like framework for Go with eBPF acceleration</p>
		</div>

		<div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-12">
			<div class="bg-gray-800 p-6 rounded-lg border border-gray-700">
				<h3 class="text-xl font-semibold mb-3 text-blue-400">⚡ Ultra Fast</h3>
				<p class="text-gray-400">Sub-microsecond routing with eBPF XDP kernel-level packet processing</p>
			</div>
			<div class="bg-gray-800 p-6 rounded-lg border border-gray-700">
				<h3 class="text-xl font-semibold mb-3 text-green-400">📁 File-based Routing</h3>
				<p class="text-gray-400">Next.js-style routing conventions with automatic route discovery</p>
			</div>
			<div class="bg-gray-800 p-6 rounded-lg border border-gray-700">
				<h3 class="text-xl font-semibold mb-3 text-purple-400">🎨 Type-safe Templates</h3>
				<p class="text-gray-400">Full type safety with a-h/templ components and Go</p>
			</div>
		</div>

		<div class="bg-gray-800 p-6 rounded-lg border border-gray-700">
			<h3 class="text-lg font-semibold mb-4">Quick Links</h3>
			<ul class="space-y-2 text-gray-400">
				<li><code class="bg-gray-900 px-2 py-1 rounded">GET /</code> - This page</li>
				<li><code class="bg-gray-900 px-2 py-1 rounded">GET /about</code> - About page</li>
				<li><code class="bg-gray-900 px-2 py-1 rounded">GET /{slug}</code> - Dynamic route example</li>
				<li><code class="bg-gray-900 px-2 py-1 rounded">GET /api/users</code> - API endpoint</li>
				<li><code class="bg-gray-900 px-2 py-1 rounded">GET /api/routes</code> - List all routes</li>
				<li><code class="bg-gray-900 px-2 py-1 rounded">GET /health</code> - Health check</li>
			</ul>
		</div>
	</main>

	<footer class="bg-gray-800 border-t border-gray-700 mt-12 py-4 text-center text-gray-500">
		Powered by Zeptor + eBPF | Version: ` + version + `
	</footer>
</body>
</html>`
}

func render404Page() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>404 - Not Found</title>
	<script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-900 text-white min-h-screen flex items-center justify-center">
	<div class="text-center">
		<h1 class="text-8xl font-bold text-gray-700 mb-4">404</h1>
		<p class="text-xl text-gray-400 mb-8">Page not found</p>
		<a href="/" class="bg-blue-600 hover:bg-blue-700 px-6 py-3 rounded-lg text-white font-semibold">
			Go Home
		</a>
	</div>
</body>
</html>`
}
