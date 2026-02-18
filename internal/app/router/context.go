package router

import (
	"context"
	"net/http"
)

type contextKey string

const (
	RouteParamsKey contextKey = "routeParams"
	RouteKey       contextKey = "route"
)

type RouteParams struct {
	Params map[string]string
	Route  *Route
}

func ParamsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		params := &RouteParams{
			Params: make(map[string]string),
		}

		ctx = context.WithValue(ctx, RouteParamsKey, params)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func SetParams(r *http.Request, params map[string]string) {
	ctx := r.Context()
	if p, ok := ctx.Value(RouteParamsKey).(*RouteParams); ok {
		for k, v := range params {
			p.Params[k] = v
		}
	}
}

func GetParams(r *http.Request) map[string]string {
	ctx := r.Context()
	if p, ok := ctx.Value(RouteParamsKey).(*RouteParams); ok {
		return p.Params
	}
	return nil
}

func Param(r *http.Request, name string) string {
	params := GetParams(r)
	if params != nil {
		return params[name]
	}
	return ""
}

func SetRoute(r *http.Request, route *Route) {
	ctx := r.Context()
	if p, ok := ctx.Value(RouteParamsKey).(*RouteParams); ok {
		p.Route = route
	}
}

func GetRouteFromContext(r *http.Request) *Route {
	ctx := r.Context()
	if p, ok := ctx.Value(RouteParamsKey).(*RouteParams); ok {
		return p.Route
	}
	return nil
}
