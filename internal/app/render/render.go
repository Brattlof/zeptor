package render

import (
	"context"
	"io"
)

type RenderMode int

const (
	ModeSSR RenderMode = iota
	ModeSSG
	ModeISR
)

type Component interface {
	Render(ctx context.Context, w io.Writer) error
}

type Renderer struct {
	mode RenderMode
}

func NewRenderer(mode RenderMode) *Renderer {
	return &Renderer{mode: mode}
}

func (r *Renderer) Render(ctx context.Context, w io.Writer, component Component) error {
	return component.Render(ctx, w)
}

func (r *Renderer) Mode() RenderMode {
	return r.mode
}

func ParseRenderMode(s string) RenderMode {
	switch s {
	case "ssg":
		return ModeSSG
	case "isr":
		return ModeISR
	default:
		return ModeSSR
	}
}
