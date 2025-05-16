package options

import (
	"net/http"
)

type API struct {
	routeMap map[string](func(http.ResponseWriter, *http.Request) http.ResponseWriter)
}

func InitAPI() *API {
	var a API
	a.routeMap = make(map[string](func(http.ResponseWriter, *http.Request) http.ResponseWriter))
	return &a
}

func (a *API) AddRoute(routeName string, handle func(http.ResponseWriter, *http.Request) http.ResponseWriter) {
	a.routeMap[routeName] = handle
}

func (a *API) RemoveRoute(routeName string) {
	delete(a.routeMap, routeName)
}

func (a *API) Process(w http.ResponseWriter, r *http.Request) http.ResponseWriter {
	path := r.URL.Path
	routeFunc, hasRoute := a.routeMap[path]
	if hasRoute {
		w = routeFunc(w, r)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
	return w
}
