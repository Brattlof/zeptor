package basicauth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brattlof/zeptor/pkg/plugin"
)

func TestBasicAuthPlugin_Name(t *testing.T) {
	p := New()
	if p.Name() != "basicauth" {
		t.Errorf("Name() = %q, want basicauth", p.Name())
	}
}

func TestBasicAuthPlugin_Init(t *testing.T) {
	p := New()
	ctx := plugin.NewPluginContext(nil, map[string]interface{}{
		"users": []string{"admin:secret", "user:pass"},
		"paths": []string{"/admin", "/api"},
		"realm": "MyRealm",
	}, nil)

	if err := p.Init(ctx); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if len(p.users) != 2 {
		t.Errorf("users count = %d, want 2", len(p.users))
	}
	if p.users["admin"] != "secret" {
		t.Errorf("users[admin] = %q, want secret", p.users["admin"])
	}
	if p.realm != "MyRealm" {
		t.Errorf("realm = %q, want MyRealm", p.realm)
	}
}

func TestBasicAuthPlugin_Middleware(t *testing.T) {
	p := New()
	ctx := plugin.NewPluginContext(nil, map[string]interface{}{
		"users": []string{"admin:secret"},
	}, nil)
	p.Init(ctx)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	middleware := p.OnMiddleware()(handler)

	t.Run("no auth header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("valid credentials", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		req.SetBasicAuth("admin", "secret")
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("invalid credentials", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		req.SetBasicAuth("admin", "wrong")
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("unknown user", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		req.SetBasicAuth("unknown", "pass")
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}

func TestBasicAuthPlugin_PathFiltering(t *testing.T) {
	p := New()
	ctx := plugin.NewPluginContext(nil, map[string]interface{}{
		"users": []string{"admin:secret"},
		"paths": []string{"/admin"},
	}, nil)
	p.Init(ctx)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	middleware := p.OnMiddleware()(handler)

	t.Run("protected path requires auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("unprotected path allows access", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/public", nil)
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
		}
	})
}
