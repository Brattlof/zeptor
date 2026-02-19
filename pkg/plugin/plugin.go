package plugin

import (
	"net/http"
)

type Plugin interface {
	Name() string
	Version() string
	Description() string
	Init(ctx *PluginContext) error
	Close() error
}

type Hook interface {
	Priority() int
}

type ConfigHook interface {
	Hook
	OnConfigLoad(config map[string]interface{}) error
}

type RouterHook interface {
	Hook
	OnRouterInit(router Router) error
}

type MiddlewareHook interface {
	Hook
	OnMiddleware() func(http.Handler) http.Handler
}

type RequestHook interface {
	Hook
	OnRequest(r *http.Request) error
}

type ResponseHook interface {
	Hook
	OnResponse(w http.ResponseWriter, r *http.Request, status int)
}

type BuildHook interface {
	Hook
	OnBuildPre() error
	OnBuildPost() error
}

type DevHook interface {
	Hook
	OnDevStart() error
	OnDevReload(path string) error
	OnDevStop() error
}

type Router interface {
	Get(pattern string, handler http.HandlerFunc)
	Post(pattern string, handler http.HandlerFunc)
	Put(pattern string, handler http.HandlerFunc)
	Delete(pattern string, handler http.HandlerFunc)
	Use(middleware func(http.Handler) http.Handler)
	Mount(pattern string, handler http.Handler)
}

type HookType string

const (
	HookConfig     HookType = "config"
	HookRouter     HookType = "router"
	HookMiddleware HookType = "middleware"
	HookRequest    HookType = "request"
	HookResponse   HookType = "response"
	HookBuild      HookType = "build"
	HookDev        HookType = "dev"
)

type Info struct {
	Name        string
	Version     string
	Description string
	Enabled     bool
	Config      map[string]interface{}
	Hooks       []HookType
}
