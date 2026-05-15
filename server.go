package song

import (
	"fmt"
	"log"
	"net/http"
)

type Server struct {
	config   Config
	router   *Router
	executor *Executor
	specGen  *SpecGenerator
}

func NewServer(cfg Config) *Server {
	return &Server{
		config:   cfg,
		router:   NewRouter(cfg.HandlersDir),
		executor: NewExecutor(cfg.HandlersDir),
		specGen:  NewSpecGenerator(cfg.HandlersDir),
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)

	switch {
	case r.URL.Path == "/health":
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"healthy","service":"song"}`))
		return
	case r.URL.Path == "/api/spec":
		spec, err := s.specGen.Generate()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(spec)
		return
	}

	switch r.Method {
	case http.MethodGet, http.MethodHead:
		s.serveStatic(w, r)
	case http.MethodPost:
		s.handleAPIRequest(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) serveStatic(w http.ResponseWriter, r *http.Request) {
	fs := http.FileServer(http.Dir(s.config.PublicDir))
	fs.ServeHTTP(w, r)
}

func (s *Server) handleAPIRequest(w http.ResponseWriter, r *http.Request) {
	handlerName, params, err := s.router.Match(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	go func() {
		result, execErr := s.executor.Execute(handlerName, r, params)
		if execErr != nil {
			log.Printf("Handler %s failed: %v", handlerName, execErr)
			return
		}
		log.Printf("Handler %s completed: %s", handlerName, string(result))
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(fmt.Sprintf(`{"status":"processing","handler":"%s"}`, handlerName)))
}
