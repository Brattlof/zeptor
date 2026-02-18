package dev

import (
	"bytes"
	"context"
	"fmt"
	"io"
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
	config      *config.Config
	router      *router.Router
	hmr         *HMRSerer
	watcher     *Watcher
	proxyServer *http.Server
	childCmd    *exec.Cmd
	childMu     sync.Mutex
	rebuilding  bool
}

func NewDevServer(cfg *config.Config) (*DevServer, error) {
	rt, err := router.New(cfg.Routing.AppDir)
	if err != nil {
		return nil, fmt.Errorf("create router: %w", err)
	}

	return &DevServer{
		config: cfg,
		router: rt,
		hmr:    NewHMRSerer(),
	}, nil
}

func (d *DevServer) Start(ctx context.Context) error {
	if err := d.buildAndStartChild(ctx); err != nil {
		slog.Warn("Initial build failed", "error", err)
	}

	d.watcher, _ = NewWatcher([]string{
		d.config.Routing.AppDir,
	}, d.handleFileChange)

	if err := d.watcher.Start(ctx); err != nil {
		slog.Warn("File watcher failed", "error", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/__hmr", d.hmr.Handler())

	publicDir := d.config.Routing.PublicDir
	if _, err := os.Stat(publicDir); err == nil {
		r.Handle("/public/*", http.StripPrefix("/public/",
			noCacheFileServer(http.Dir(publicDir))))
	}

	r.HandleFunc("/*", d.proxyHandler)

	d.proxyServer = &http.Server{
		Addr:         d.config.Addr(),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	slog.Info("Dev proxy starting",
		"addr", d.config.Addr(),
		"routes", len(d.router.Routes()),
	)

	go func() {
		if err := d.proxyServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Proxy server error", "error", err)
		}
	}()

	return nil
}

func (d *DevServer) buildAndStartChild(ctx context.Context) error {
	d.childMu.Lock()
	defer d.childMu.Unlock()

	if d.childCmd != nil && d.childCmd.Process != nil {
		d.childCmd.Process.Signal(syscall.SIGTERM)
		time.Sleep(500 * time.Millisecond)
		if d.childCmd.Process != nil {
			d.childCmd.Process.Kill()
		}
		d.childCmd.Wait()
	}

	slog.Info("Building example...")
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", ".zeptor/server.exe", ".")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	slog.Info("Starting child server on :3001")
	d.childCmd = exec.CommandContext(context.Background(), "./.zeptor/server.exe")
	d.childCmd.Stdout = os.Stdout
	d.childCmd.Stderr = os.Stderr
	d.childCmd.Env = append(os.Environ(), "PORT=3001")

	if err := d.childCmd.Start(); err != nil {
		return fmt.Errorf("start child: %w", err)
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

func (d *DevServer) proxyHandler(w http.ResponseWriter, r *http.Request) {
	d.childMu.Lock()
	childCmd := d.childCmd
	d.childMu.Unlock()

	if childCmd == nil || childCmd.Process == nil {
		http.Error(w, "Server starting...", http.StatusServiceUnavailable)
		return
	}

	targetURL := fmt.Sprintf("http://localhost:3001%s", r.URL.Path)
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, "Proxy error", http.StatusInternalServerError)
		return
	}

	proxyReq.Header = r.Header.Clone()

	resp, err := http.DefaultClient.Do(proxyReq)
	if err != nil {
		http.Error(w, "Server unavailable", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	var body []byte
	if strings.Contains(contentType, "text/html") {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "Read error", http.StatusInternalServerError)
			return
		}
		body = d.injectHMR(bodyBytes)
	} else {
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "Read error", http.StatusInternalServerError)
			return
		}
	}

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func (d *DevServer) injectHMR(body []byte) []byte {
	idx := bytes.LastIndex(body, []byte("</body>"))
	if idx == -1 {
		return body
	}

	var result []byte
	result = append(result, body[:idx]...)
	result = append(result, []byte(hmrClientScript)...)
	result = append(result, body[idx:]...)
	return result
}

func (d *DevServer) handleFileChange(path string) {
	ext := filepath.Ext(path)
	d.childMu.Lock()
	rebuilding := d.rebuilding
	d.childMu.Unlock()

	if rebuilding {
		return
	}

	switch ext {
	case ".templ":
		slog.Info("Templ file changed, rebuilding...", "file", path)
		d.childMu.Lock()
		d.rebuilding = true
		d.childMu.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		if err := exec.CommandContext(ctx, "templ", "generate").Run(); err != nil {
			slog.Error("Templ generate failed", "error", err)
			d.childMu.Lock()
			d.rebuilding = false
			d.childMu.Unlock()
			return
		}

		if err := d.buildAndStartChild(ctx); err != nil {
			slog.Error("Rebuild failed", "error", err)
			d.childMu.Lock()
			d.rebuilding = false
			d.childMu.Unlock()
			return
		}

		d.childMu.Lock()
		d.rebuilding = false
		d.childMu.Unlock()

		d.hmr.Reload(path)

	case ".go":
		if strings.Contains(path, "_templ.go") {
			return
		}
		slog.Info("Go file changed, rebuilding...", "file", path)
		d.childMu.Lock()
		d.rebuilding = true
		d.childMu.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		if err := d.buildAndStartChild(ctx); err != nil {
			slog.Error("Rebuild failed", "error", err)
			d.childMu.Lock()
			d.rebuilding = false
			d.childMu.Unlock()
			return
		}

		d.childMu.Lock()
		d.rebuilding = false
		d.childMu.Unlock()

		d.hmr.Reload(path)
	}
}

func (d *DevServer) Shutdown(ctx context.Context) error {
	if d.watcher != nil {
		d.watcher.Close()
	}
	d.hmr.Close()

	d.childMu.Lock()
	if d.childCmd != nil && d.childCmd.Process != nil {
		d.childCmd.Process.Signal(syscall.SIGTERM)
	}
	d.childMu.Unlock()

	if d.proxyServer != nil {
		return d.proxyServer.Shutdown(ctx)
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
