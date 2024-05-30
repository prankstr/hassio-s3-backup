package router

import "net/http"

// Router defines an interface to specify a group of routes to add to the server.
type Router interface {
	RegisterRoutes(mux *http.ServeMux)
}

// BaseRouter is a struct that implements the Router interface.
type BaseRouter struct {
	routes []Route
}

// Route defines an individual API route in the server.
type Route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
}

// Routes returns the list of routes to add to the server.
func (br *BaseRouter) Routes() []Route {
	return br.routes
}

// RegisterRoutes registers all routes in the BaseRouter to the given ServeMux.
func (br *BaseRouter) RegisterRoutes(mux *http.ServeMux) {
	for _, route := range br.routes {
		mux.HandleFunc(route.Path, route.Handler)
	}
}

// AddRoute adds a new route to the BaseRouter.
func (br *BaseRouter) AddRoute(method, path string, handler http.HandlerFunc) {
	br.routes = append(br.routes, Route{method, path, handler})
}
