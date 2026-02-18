package router

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type RouteType int

const (
	RouteTypePage RouteType = iota
	RouteTypeAPI
	RouteTypeLayout
)

type Route struct {
	Pattern     string
	Handler     http.HandlerFunc
	Params      []string
	IsDynamic   bool
	Type        RouteType
	File        string
	Method      string
	Middlewares []func(http.Handler) http.Handler
	Children    []*Route
}

type Layout struct {
	Pattern string
	File    string
	Handler func(child http.Handler) http.Handler
}

type Router struct {
	routes  []*Route
	layouts []*Layout
	tree    *radixNode
	static  map[string]*Route
	dynamic []*Route
	appDir  string
}

var (
	dynamicSegment  = regexp.MustCompile(`\[([^\]]+)\]`)
	catchAllSegment = regexp.MustCompile(`\[\.\.\.([^\]]+)\]`)
)

func New(appDir string) (*Router, error) {
	r := &Router{
		static:  make(map[string]*Route),
		dynamic: make([]*Route, 0),
		routes:  make([]*Route, 0),
		layouts: make([]*Layout, 0),
		tree:    newRadixNode("", nodeStatic),
		appDir:  appDir,
	}

	absPath, err := filepath.Abs(appDir)
	if err != nil {
		return r, nil
	}

	if _, err := os.Stat(absPath); err != nil {
		return r, nil
	}

	if err := r.discoverRoutes(absPath); err != nil {
		return nil, err
	}

	r.buildTree()

	return r, nil
}

func (r *Router) discoverRoutes(appDir string) error {
	return filepath.Walk(appDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if strings.HasPrefix(info.Name(), "_") || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		relPath := strings.TrimPrefix(path, appDir)
		relPath = strings.TrimPrefix(relPath, string(filepath.Separator))

		baseName := info.Name()
		dirPath := filepath.Dir(path)

		switch baseName {
		case "page.templ", "page.go":
			r.addPageRoute(relPath, path)
		case "layout.templ":
			r.addLayoutRoute(relPath, path)
		case "route.go":
			r.addAPIRoute(relPath, path)
		}

		_ = dirPath

		return nil
	})
}

func (r *Router) addPageRoute(relPath, fullPath string) {
	relPath = filepath.ToSlash(relPath)
	pattern := strings.TrimSuffix(relPath, "page.templ")
	pattern = strings.TrimSuffix(pattern, "page.go")
	pattern = strings.TrimSuffix(pattern, "/")
	pattern = "/" + strings.TrimPrefix(pattern, "/")

	params := []string{}
	isDynamic := dynamicSegment.MatchString(pattern) || catchAllSegment.MatchString(pattern)

	if isDynamic {
		matches := dynamicSegment.FindAllStringSubmatch(pattern, -1)
		for _, m := range matches {
			params = append(params, m[1])
		}
		matches = catchAllSegment.FindAllStringSubmatch(pattern, -1)
		for _, m := range matches {
			params = append(params, "..."+m[1])
		}
		pattern = normalizePattern(pattern)
	}

	if pattern == "/" {
		pattern = "/"
	} else {
		pattern = strings.TrimSuffix(pattern, "/")
	}

	route := &Route{
		Pattern:   pattern,
		Params:    params,
		IsDynamic: isDynamic,
		Type:      RouteTypePage,
		File:      fullPath,
		Method:    "GET",
	}

	if isDynamic {
		r.dynamic = append(r.dynamic, route)
	} else {
		r.static[pattern] = route
	}
	r.routes = append(r.routes, route)
}

func (r *Router) addLayoutRoute(relPath, fullPath string) {
	relPath = filepath.ToSlash(relPath)
	pattern := strings.TrimSuffix(relPath, "layout.templ")
	pattern = strings.TrimSuffix(pattern, "/")
	pattern = "/" + strings.TrimPrefix(pattern, "/")

	if pattern == "" {
		pattern = "/"
	}

	layout := &Layout{
		Pattern: pattern,
		File:    fullPath,
	}

	r.layouts = append(r.layouts, layout)
}

func (r *Router) addAPIRoute(relPath, fullPath string) {
	relPath = filepath.ToSlash(relPath)
	pattern := strings.TrimSuffix(relPath, "route.go")
	pattern = strings.TrimSuffix(pattern, "/")
	pattern = "/" + pattern
	if pattern == "/" {
		pattern = "/api"
	}

	params := []string{}
	isDynamic := dynamicSegment.MatchString(pattern) || catchAllSegment.MatchString(pattern)

	if isDynamic {
		matches := dynamicSegment.FindAllStringSubmatch(pattern, -1)
		for _, m := range matches {
			params = append(params, m[1])
		}
		pattern = normalizePattern(pattern)
	}

	pattern = strings.TrimSuffix(pattern, "/")

	route := &Route{
		Pattern:   pattern,
		Params:    params,
		IsDynamic: isDynamic,
		Type:      RouteTypeAPI,
		File:      fullPath,
		Method:    "*",
	}

	if isDynamic {
		r.dynamic = append(r.dynamic, route)
	} else {
		r.static[pattern] = route
	}
	r.routes = append(r.routes, route)
}

func normalizePattern(pattern string) string {
	pattern = dynamicSegment.ReplaceAllString(pattern, "{$1}")
	pattern = catchAllSegment.ReplaceAllString(pattern, "{$1}")
	return pattern
}

func (r *Router) buildTree() {
	for _, route := range r.routes {
		r.tree.insert(route.Pattern, route)
	}
}

func (r *Router) Lookup(path string) (*Route, map[string]string) {
	if route, ok := r.static[path]; ok {
		return route, nil
	}

	return r.tree.lookup(path)
}

func (r *Router) Routes() []*Route {
	sort.Slice(r.routes, func(i, j int) bool {
		return r.routes[i].Pattern < r.routes[j].Pattern
	})
	return r.routes
}

func (r *Router) Layouts() []*Layout {
	return r.layouts
}

func (r *Router) StaticRoutes() map[string]*Route {
	return r.static
}

func (r *Router) DynamicRoutes() []*Route {
	return r.dynamic
}

func (r *Router) Mount(chiRouter interface {
	Method(method, pattern string, handler http.Handler)
}) {
	for _, route := range r.static {
		handler := r.createHandler(route)
		if route.Type == RouteTypeAPI {
			chiRouter.Method("GET", route.Pattern, handler)
			chiRouter.Method("POST", route.Pattern, handler)
			chiRouter.Method("PUT", route.Pattern, handler)
			chiRouter.Method("DELETE", route.Pattern, handler)
			chiRouter.Method("PATCH", route.Pattern, handler)
		} else {
			chiRouter.Method("GET", route.Pattern, handler)
		}
	}

	for _, route := range r.dynamic {
		handler := r.createHandler(route)
		if route.Type == RouteTypeAPI {
			chiRouter.Method("GET", route.Pattern, handler)
			chiRouter.Method("POST", route.Pattern, handler)
			chiRouter.Method("PUT", route.Pattern, handler)
			chiRouter.Method("DELETE", route.Pattern, handler)
			chiRouter.Method("PATCH", route.Pattern, handler)
		} else {
			chiRouter.Method("GET", route.Pattern, handler)
		}
	}
}

func (r *Router) createHandler(route *Route) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		ctx = context.WithValue(ctx, routeKey{}, route)
		*req = *req.WithContext(ctx)

		if route.Handler != nil {
			route.Handler(w, req)
			return
		}

		r.defaultHandler(w, req, route)
	})
}

func (r *Router) defaultHandler(w http.ResponseWriter, req *http.Request, route *Route) {
	switch route.Type {
	case RouteTypePage:
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Zeptor - %s</title>
<script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-900 text-white min-h-screen">
<nav class="bg-gray-800 p-4"><div class="container mx-auto flex gap-4">
<a href="/" class="hover:text-blue-400">Home</a>
<a href="/about" class="hover:text-blue-400">About</a>
</div></nav>
<main class="container mx-auto p-8">
<h1 class="text-3xl font-bold mb-4">%s</h1>
<p class="text-gray-400 mb-2">Pattern: <code class="bg-gray-800 px-2 py-1 rounded">%s</code></p>
<p class="text-gray-400 mb-2">File: <code class="bg-gray-800 px-2 py-1 rounded">%s</code></p>
<p class="text-yellow-400 mt-4">Handler not yet implemented</p>
</main>
</body>
</html>`, route.Pattern, route.Pattern, route.Pattern, route.File)

	case RouteTypeAPI:
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"error":"API handler not implemented","route":"%s","file":"%s"}`, route.Pattern, route.File)
	}
}

type routeKey struct{}

func GetRoute(ctx context.Context) *Route {
	if route, ok := ctx.Value(routeKey{}).(*Route); ok {
		return route
	}
	return nil
}

func GetParam(ctx context.Context, name string) string {
	if req, ok := ctx.Value(http.Request{}).(*http.Request); ok {
		if chiCtx, ok := req.Context().Value(chiRouteCtxKey{}).(map[string]string); ok {
			return chiCtx[name]
		}
	}
	return ""
}

type chiRouteCtxKey struct{}
