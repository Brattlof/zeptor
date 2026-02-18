package dev

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

type Builder struct {
	appDir  string
	outDir  string
	mu      sync.Mutex
	running bool
}

func NewBuilder(appDir, outDir string) *Builder {
	return &Builder{
		appDir: appDir,
		outDir: outDir,
	}
}

func (b *Builder) GenerateTempl(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	cmd := exec.CommandContext(ctx, "templ", "generate")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("templ generate: %w", err)
	}

	return nil
}

func (b *Builder) GenerateEBPF(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	cmd := exec.CommandContext(ctx, "go", "generate", "./internal/ebpf/...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go generate ebpf: %w", err)
	}

	return nil
}

func (b *Builder) BuildSSG(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := os.MkdirAll(b.outDir, 0755); err != nil {
		return fmt.Errorf("create out dir: %w", err)
	}

	slog.Info("SSG build not yet implemented")

	return nil
}

func (b *Builder) BuildBinary(ctx context.Context, output string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	cmd := exec.CommandContext(ctx, "go", "build", "-o", output, "./cmd/zeptor")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build: %w", err)
	}

	return nil
}

func (b *Builder) Rebuild(ctx context.Context, changedFile string) error {
	ext := filepath.Ext(changedFile)

	switch ext {
	case ".templ":
		slog.Info("Rebuilding templ components")
		return b.GenerateTempl(ctx)
	case ".go":
		slog.Info("Go file changed - rebuild may be needed")
		return nil
	default:
		return nil
	}
}

func (b *Builder) IsRunning() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.running
}
