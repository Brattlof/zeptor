package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouter_StaticRoutes(t *testing.T) {
	r, err := New("testdata/static")
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	tests := []struct {
		path      string
		wantFound bool
	}{
		{"/", true},
		{"/about", true},
		{"/nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			route, _ := r.Lookup(tt.path)
			if (route != nil) != tt.wantFound {
				t.Errorf("Lookup(%q) = %v, want found = %v", tt.path, route != nil, tt.wantFound)
			}
		})
	}
}

func TestRouter_DynamicRoutes(t *testing.T) {
	r := &Router{
		static:  make(map[string]*Route),
		dynamic: make([]*Route, 0),
		routes:  make([]*Route, 0),
		tree:    newRadixNode("", nodeStatic),
	}

	route := &Route{
		Pattern:   "/users/{id}",
		Params:    []string{"id"},
		IsDynamic: true,
		Type:      RouteTypePage,
	}
	r.tree.insert("/users/{id}", route)
	r.dynamic = append(r.dynamic, route)

	tests := []struct {
		path       string
		wantFound  bool
		wantParams map[string]string
	}{
		{"/users/123", true, map[string]string{"id": "123"}},
		{"/users/abc", true, map[string]string{"id": "abc"}},
		{"/users", false, nil},
		{"/posts/123", false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			route, params := r.Lookup(tt.path)
			if (route != nil) != tt.wantFound {
				t.Errorf("Lookup(%q) found = %v, want %v", tt.path, route != nil, tt.wantFound)
			}
			if tt.wantParams != nil {
				for key, wantVal := range tt.wantParams {
					if params[key] != wantVal {
						t.Errorf("params[%q] = %q, want %q", key, params[key], wantVal)
					}
				}
			}
		})
	}
}

func TestRouter_NestedDynamicRoutes(t *testing.T) {
	r := &Router{
		static:  make(map[string]*Route),
		dynamic: make([]*Route, 0),
		routes:  make([]*Route, 0),
		tree:    newRadixNode("", nodeStatic),
	}

	route1 := &Route{
		Pattern:   "/users/{userId}/posts/{postId}",
		Params:    []string{"userId", "postId"},
		IsDynamic: true,
		Type:      RouteTypePage,
	}
	r.tree.insert("/users/{userId}/posts/{postId}", route1)

	tests := []struct {
		path       string
		wantFound  bool
		wantParams map[string]string
	}{
		{"/users/1/posts/2", true, map[string]string{"userId": "1", "postId": "2"}},
		{"/users/john/posts/hello", true, map[string]string{"userId": "john", "postId": "hello"}},
		{"/users/1", false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			route, params := r.Lookup(tt.path)
			if (route != nil) != tt.wantFound {
				t.Errorf("Lookup(%q) found = %v, want %v", tt.path, route != nil, tt.wantFound)
			}
			if tt.wantParams != nil && params != nil {
				for key, wantVal := range tt.wantParams {
					if params[key] != wantVal {
						t.Errorf("params[%q] = %q, want %q", key, params[key], wantVal)
					}
				}
			}
		})
	}
}

func TestRadixTree_InsertAndLookup(t *testing.T) {
	root := newRadixNode("", nodeStatic)

	routes := []struct {
		pattern string
		route   *Route
	}{
		{"/", &Route{Pattern: "/"}},
		{"/about", &Route{Pattern: "/about"}},
		{"/about/us", &Route{Pattern: "/about/us"}},
		{"/users/{id}", &Route{Pattern: "/users/{id}", Params: []string{"id"}}},
		{"/users/{id}/profile", &Route{Pattern: "/users/{id}/profile"}},
	}

	for _, r := range routes {
		root.insert(r.pattern, r.route)
	}

	tests := []struct {
		path      string
		wantFound bool
	}{
		{"/", true},
		{"/about", true},
		{"/about/us", true},
		{"/users/123", true},
		{"/users/456/profile", true},
		{"/nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			route, _ := root.lookup(tt.path)
			if (route != nil) != tt.wantFound {
				t.Errorf("lookup(%q) = %v, want found = %v", tt.path, route != nil, tt.wantFound)
			}
		})
	}
}

func TestRouter_Mount(t *testing.T) {
	r := &Router{
		static:  make(map[string]*Route),
		dynamic: make([]*Route, 0),
		routes:  make([]*Route, 0),
		tree:    newRadixNode("", nodeStatic),
	}

	r.static["/"] = &Route{Pattern: "/", Type: RouteTypePage}
	r.static["/about"] = &Route{Pattern: "/about", Type: RouteTypePage}
	r.dynamic = append(r.dynamic, &Route{
		Pattern:   "/users/{id}",
		Params:    []string{"id"},
		IsDynamic: true,
		Type:      RouteTypePage,
	})

	for _, route := range r.static {
		r.routes = append(r.routes, route)
	}
	r.routes = append(r.routes, r.dynamic...)

	mux := http.NewServeMux()
	routesMounted := 0

	mockChi := &mockChiRouter{
		mux: mux,
		onMount: func(method, pattern string) {
			routesMounted++
		},
	}

	r.Mount(mockChi)

	if routesMounted < 2 {
		t.Errorf("Expected at least 2 routes mounted, got %d", routesMounted)
	}
}

type mockChiRouter struct {
	mux     *http.ServeMux
	onMount func(method, pattern string)
}

func (m *mockChiRouter) Method(method, pattern string, handler http.Handler) {
	if m.onMount != nil {
		m.onMount(method, pattern)
	}
	m.mux.Handle(pattern, handler)
}

func TestRouter_ParamsMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params := GetParams(r)
		if params == nil {
			t.Error("Expected params to be set")
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := ParamsMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}
