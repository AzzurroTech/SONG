package SONG

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"post_api"
	"strings"

	"vidi"

	"veni"

	"connect_api"
	"delete_api"
	"get_api"
	"head_api"
	"options_api"
	"patch_api"
	"put_api"
	"trace_api"
)

type Server struct {
	router     *http.ServeMux
	targetDir  string
	portStr    string
	ve         *veni.VeniContext
	vd         *vidi.VidiContext
	vi         string
	fileServer http.Handler
}

func InitServer(dir, ps, veniName, vidiName, viciName string, dm vidi.VidiInterface) *Server {
	var s Server
	s.router = http.NewServeMux()
	s.targetDir = dir
	s.portStr = ps
	var veniInstance veni.VeniContext
	veniInstance.ConnectAPI = connect_api.InitAPI()
	veniInstance.DeleteAPI = delete_api.InitAPI()
	veniInstance.GetAPI = get_api.InitAPI()
	veniInstance.HeadAPI = head_api.InitAPI()
	veniInstance.OptionsAPI = options_api.InitAPI()
	veniInstance.PatchAPI = patch_api.InitAPI()
	veniInstance.PutAPI = put_api.InitAPI()
	veniInstance.TraceAPI = trace_api.InitAPI()
	veniInstance.PostAPI = post_api.InitAPI()
	veniInstance.Name = veniName
	s.ve = &veniInstance
	s.vd = vidi.InitContext(vidiName, dm)
	s.vi = viciName
	ret := &s
	return ret
}

func (s *Server) Route(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.router.HandleFunc(pattern, handler)
	index := strings.Index(pattern, " ")
	if index != -1 {
		fmt.Printf("The first space is at index: %d\n", index)
		target := index + 1
		method := pattern[0:index]
		route := pattern[target:]
		switch strings.ToLower(method) {
		case "get":
			s.ve.GetAPI.AddRoute(route, handler)
		case "head":
			s.ve.HeadAPI.AddRoute(route, handler)
		case "options":
			s.ve.OptionsAPI.AddRoute(route, handler)
		case "trace":
			s.ve.TraceAPI.AddRoute(route, handler)
		case "put":
			s.ve.PutAPI.AddRoute(route, handler)
		case "delete":
			s.ve.DeleteAPI.AddRoute(route, handler)
		case "post":
			s.ve.PostAPI.AddRoute(route, handler)
		case "patch":
			s.ve.PatchAPI.AddRoute(route, handler)
		case "connect":
			s.ve.ConnectAPI.AddRoute(route, handler)
		}
	} else {
		fmt.Println("There is no space in the string...Auto creating POST and GET")
		s.ve.GetAPI.AddRoute(pattern, handler)
		s.ve.PostAPI.AddRoute(pattern, handler)
	}
}

func (s *Server) serveDirectory(route, tarString string) {
	// Define the directory to serve files from, as the parent directory (to include this component as a server)
	fs := http.Dir(tarString)
	// Create a custom file server handler
	s.fileServer = http.FileServer(fs)
	// Create a handler function to handle requests
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Attempt to open the file to serve
		_, err := fs.Open(filepath.Clean(r.URL.Path))
		if os.IsNotExist(err) {
			// If the file does not exist, throw 404
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// If the file exists, serve it using the default file server
		s.fileServer.ServeHTTP(w, r)
	}
	// Handle all requests with the custom handler, by adding in at this point
	s.Route(route, handler)
}

func getFirstPathComponent(urlPath string) (string, error) {
	parsedURL, err := url.Parse(urlPath)
	if err != nil {
		return "", err
	}

	path := parsedURL.Path
	pathComponents := strings.Split(path, "/")

	// Handle cases with leading/trailing slashes or empty paths
	for _, component := range pathComponents {
		if component != "" {
			return component, nil
		}
	}

	return "", nil // Return empty string if no component is found
}

func (s *Server) Serve() {

	// Define the directory to serve files from, as the parent directory (to include this component as a server)
	fs := http.Dir(s.targetDir)

	// Create a custom file server handler
	s.fileServer = http.FileServer(fs)

	// Create a handler function to handle requests
	handler := func(w http.ResponseWriter, r *http.Request) {
		base, err := getFirstPathComponent(r.URL.Path)
		if err != nil {
			base = ""
		}
		fmt.Println(base)
		switch strings.ToLower(base) {
		case s.vi:
			fmt.Println("VICI CALLED")
			// Define the directory to serve files from, as the parent directory (to include this component as a server)
			tfs := http.Dir("./SONG")
			fmt.Println("tfs is ")
			fmt.Println(tfs)
			fmt.Println(filepath.Clean(r.URL.Path))
			// Create a custom file server handler
			tmpFileServer := http.FileServer(tfs)
			// Attempt to open the file to serve
			_, err := tfs.Open(filepath.Clean(r.URL.Path))
			if os.IsNotExist(err) {
				// If the file does not exist, throw 404
				w.WriteHeader(http.StatusNotFound)
				return
			}
			// If the file exists, serve it using the default file server
			tmpFileServer.ServeHTTP(w, r)
		default:
			if s.vd.Comply(r) {
				s.vd.ProcessRequest(w, r)
			} else {
				if s.ve.Comply(r) {
					//Veni compliant, attempting to load as VENI
					s.ve.Process(w, r)
				} else {
					// Attempt to open the file to serve
					_, err := fs.Open(filepath.Clean(r.URL.Path))
					if os.IsNotExist(err) {
						// If the file does not exist, throw 404
						w.WriteHeader(http.StatusNotFound)
						return
					}
					// If the file exists, serve it using the default file server
					s.fileServer.ServeHTTP(w, r)
				}
			}
		}
	}

	// Handle all requests with the custom handler, by adding in at this point
	s.Route("/", handler)
	s.Route("GET /song/", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "SONG/README.md") })
	s.Route("GET /song/veni/", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "SONG/veni/README.md") })
	s.Route("GET /song/vidi/", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "SONG/vidi/README.md") })
	s.Route("GET /song/vici/", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "SONG/vici/README.md") })
	// Start the server
	log.Println("Server listening on port " + s.portStr)
	log.Fatal(http.ListenAndServe(s.portStr, s.router))
}
