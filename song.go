package song

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	//veni "github.com/Emperor42/veni"
	//vidi "github.com/Emperor42/vidi"
)

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
