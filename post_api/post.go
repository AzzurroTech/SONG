package post_api

import (
	"net/http"
)

type API struct {
	routeMap map[string](func(http.ResponseWriter, *http.Request))
}

func InitAPI() *API {
	var a API
	a.routeMap = make(map[string](func(http.ResponseWriter, *http.Request)))
	return &a
}

func (a *API) AddRoute(routeName string, handle func(http.ResponseWriter, *http.Request)) {
	a.routeMap[routeName] = handle
	//Add with post lower case
	appendRouteName := routeName + "/post"
	a.routeMap[appendRouteName] = handle
}

func (a *API) RemoveRoute(routeName string) {
	delete(a.routeMap, routeName)
	//Add with post lower case
	appendRouteName := routeName + "/post"
	delete(a.routeMap, appendRouteName)
}

func (a *API) Process(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	routeFunc, hasRoute := a.routeMap[path]
	if hasRoute {
		routeFunc(w, r)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}
