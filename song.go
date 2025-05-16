package song

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/AzzurroTech/SONG/connect"
	//veni "github.com/Emperor42/veni"
	//vidi "github.com/Emperor42/vidi"
)

func test() {
	test := connect.InitAPI()
	test.AddRoute("test", nil)
	test.RemoveRoute("test")
}

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

func Serve() {
	/*
		http.Handle("/call", &veni.VeniContext{Name: "veni"})
		http.Handle("/data", &vidi.VidiContext{Name: "vidi"})

		fmt.Printf("Starting server at port 3000\n")
		if err := http.ListenAndServe(":3000", nil); err != nil {
			log.Fatal(err)
		}
	*/
	// Define the directory to serve files from, as the parent directory (to include this componet as a server)
	fs := http.Dir("..")

	// Create a custom file server handler
	fileServer := http.FileServer(fs)

	// Create a handler function to handle requests
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Attempt to open the file
		_, err := fs.Open(filepath.Clean(r.URL.Path))
		if os.IsNotExist(err) {
			// If the file does not exist, throw 404
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// If the file exists, serve it using the default file server
		fileServer.ServeHTTP(w, r)
	}

	// Handle all requests with the custom handler
	http.HandleFunc("/", handler)

	// Start the server
	log.Println("Server listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
