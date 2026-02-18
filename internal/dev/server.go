package dev

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/brattlof/zeptor/internal/app/config"
	"github.com/brattlof/zeptor/internal/app/router"
)

type DevServer struct {
	config     *config.Config
	router     *router.Router
	hmr        *HMRSerer
	watcher    *Watcher
	builder    *Builder
	server     *http.Server
	restartMu  sync.Mutex
	restarting bool
	cancel     context.CancelFunc
}

func NewDevServer(cfg *config.Config) (*DevServer, error) {
	rt, err := router.New(cfg.Routing.AppDir)
	if err != nil {
		return nil, fmt.Errorf("create router: %w", err)
	}

	return &DevServer{
		config:  cfg,
		router:  rt,
		hmr:     NewHMRSerer(),
		builder: NewBuilder(cfg.Routing.AppDir, cfg.Build.OutDir),
	}, nil
}

func (d *DevServer) Start(ctx context.Context) error {
	d.watcher, _ = NewWatcher([]string{
		d.config.Routing.AppDir,
		"cmd",
		"internal",
	}, d.handleFileChange)

	if err := d.watcher.Start(ctx); err != nil {
		slog.Warn("File watcher failed", "error", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(InjectHMR)

	d.setupRoutes(r)

	d.server = &http.Server{
		Addr:         d.config.Addr(),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	slog.Info("Starting dev server",
		"addr", d.config.Addr(),
		"routes", len(d.router.Routes()),
	)

	go func() {
		if err := d.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", "error", err)
		}
	}()

	return nil
}

func (d *DevServer) setupRoutes(r chi.Router) {
	r.Get("/__hmr", d.hmr.Handler())

	publicDir := d.config.Routing.PublicDir
	if _, err := os.Stat(publicDir); err == nil {
		r.Handle("/public/*", http.StripPrefix("/public/",
			noCacheFileServer(http.Dir(publicDir))))
	}

	r.Get("/", d.renderHome)
	r.Get("/about", d.renderAbout)
	r.Get("/{slug}", d.renderDynamic)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","mode":"dev"}`)
	})

	r.Get("/api/routes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		routes := d.router.Routes()
		fmt.Fprintf(w, `{"routes":[`)
		for i, route := range routes {
			if i > 0 {
				w.Write([]byte(","))
			}
			fmt.Fprintf(w, `{"pattern":"%s","file":"%s"}`, route.Pattern, route.File)
		}
		w.Write([]byte(`]}`))
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`<!DOCTYPE html><html><body><h1>404 Not Found</h1></body></html>`))
	})
}

func (d *DevServer) renderHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(d.renderPage("Home | Zeptor", "Welcome to Zeptor (Dev Mode)")))
}

func (d *DevServer) renderAbout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(d.renderPage("About | Zeptor", "About Zeptor (Dev Mode)")))
}

func (d *DevServer) renderDynamic(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(d.renderPage("Dynamic | Zeptor", "Slug: "+slug)))
}

func (d *DevServer) renderPage(title, heading string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>%s</title>
	<script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-900 text-white min-h-screen">
	<nav class="bg-gray-800 border-b border-gray-700">
		<div class="container mx-auto px-4 py-3 flex items-center justify-between">
			<a href="/" class="text-xl font-bold text-blue-400">Zeptor (Dev)</a>
			<div class="flex gap-4">
				<a href="/" class="hover:text-blue-400">Home</a>
				<a href="/about" class="hover:text-blue-400">About</a>
			</div>
		</div>
	</nav>
	<main class="container mx-auto px-4 py-12">
		<h1 class="text-4xl font-bold mb-6">%s</h1>
		<p class="text-gray-400">Development mode - HMR enabled</p>
	</main>
</body>
</html>`, title, heading)
}

func (d *DevServer) handleFileChange(path string) {
	ext := filepath.Ext(path)

	switch ext {
	case ".templ":
		slog.Info("Templ file changed, regenerating...", "file", path)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := d.builder.GenerateTempl(ctx); err != nil {
			slog.Error("Failed to regenerate templ", "error", err)
			return
		}

		d.hmr.Reload(path)

	case ".go":
		if strings.Contains(path, "_templ.go") {
			return
		}
		slog.Info("Go file changed, rebuild needed", "file", path)
		d.hmr.Reload(path)
	}
}

func (d *DevServer) Shutdown(ctx context.Context) error {
	if d.watcher != nil {
		d.watcher.Close()
	}
	d.hmr.Close()
	if d.server != nil {
		return d.server.Shutdown(ctx)
	}
	return nil
}

func noCacheFileServer(root http.FileSystem) http.Handler {
	fs := http.FileServer(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		fs.ServeHTTP(w, r)
	})
}

func RunDev(cfg *config.Config) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dev, err := NewDevServer(cfg)
	if err != nil {
		return fmt.Errorf("create dev server: %w", err)
	}

	if err := dev.Start(ctx); err != nil {
		return fmt.Errorf("start dev server: %w", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down dev server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	return dev.Shutdown(shutdownCtx)
}

func BuildAndRun(cfg *config.Config) error {
	ctx := context.Background()

	slog.Info("Generating templ components...")
	if err := exec.CommandContext(ctx, "templ", "generate").Run(); err != nil {
		return fmt.Errorf("templ generate: %w", err)
	}

	slog.Info("Building binary...")
	cmd := exec.CommandContext(ctx, "go", "build", "-o", "bin/zeptor.exe", "./cmd/zeptor")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build: %w", err)
	}

	slog.Info("Starting server...")
	serverCmd := exec.CommandContext(ctx, "./bin/zeptor.exe")
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr
	serverCmd.Stdin = os.Stdin

	if err := serverCmd.Start(); err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down...")
	return serverCmd.Process.Signal(syscall.SIGTERM)
}
