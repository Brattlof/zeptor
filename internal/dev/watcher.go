package dev

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	watcher  *fsnotify.Watcher
	dirs     []string
	onChange func(path string)
	mu       sync.Mutex
	debounce map[string]time.Time
	running  bool
}

func NewWatcher(dirs []string, onChange func(path string)) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create watcher: %w", err)
	}

	return &Watcher{
		watcher:  w,
		dirs:     dirs,
		onChange: onChange,
		debounce: make(map[string]time.Time),
	}, nil
}

func (w *Watcher) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = true
	w.mu.Unlock()

	for _, dir := range w.dirs {
		if err := w.addDir(dir); err != nil {
			slog.Warn("Failed to watch directory", "dir", dir, "error", err)
		}
	}

	go w.processEvents(ctx)

	return nil
}

func (w *Watcher) addDir(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return w.watcher.Add(path)
		}
		return nil
	})
}

func (w *Watcher) processEvents(ctx context.Context) {
	debounceDuration := 500 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			if event.Op&fsnotify.Write == 0 && event.Op&fsnotify.Create == 0 {
				continue
			}

			ext := strings.ToLower(filepath.Ext(event.Name))
			if ext != ".go" && ext != ".templ" && ext != ".yaml" && ext != ".yml" {
				continue
			}

			w.mu.Lock()
			lastChange, exists := w.debounce[event.Name]
			if exists && time.Since(lastChange) < debounceDuration {
				w.mu.Unlock()
				continue
			}
			w.debounce[event.Name] = time.Now()
			w.mu.Unlock()

			slog.Info("File changed", "path", event.Name, "op", event.Op.String())

			if w.onChange != nil {
				go w.onChange(event.Name)
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("Watcher error", "error", err)
		}
	}
}

func (w *Watcher) Close() error {
	if w.watcher != nil {
		return w.watcher.Close()
	}
	return nil
}
