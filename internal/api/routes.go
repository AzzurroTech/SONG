package api

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/azzurrotech/song/internal/auth"
	"github.com/azzurrotech/song/internal/config"
	"github.com/azzurrotech/song/internal/core"
	"github.com/azzurrotech/song/internal/db"
	"github.com/azzurrotech/song/internal/faas"
	"github.com/azzurrotech/song/internal/templates"
	"github.com/gorilla/mux"
)

// Router wraps the mux router with additional context
type Router struct {
	*mux.Router
	config     *config.Config
	coreStore  *core.Store
	dbManager  *db.Manager
	authSvc    *auth.Service
	faasEngine *faas.Engine
	templateR  *templates.Renderer
}

// SetupRoutes creates and configures all HTTP routes
func SetupRoutes(
	cfg *config.Config,
	coreStore *core.Store,
	dbManager *db.Manager,
	authSvc *auth.Service,
	faasEngine *faas.Engine,
	templateR *templates.Renderer,
) *mux.Router {
	r := mux.NewRouter()

	// Create router wrapper
	router := &Router{
		Router:     r,
		config:     cfg,
		coreStore:  coreStore,
		dbManager:  dbManager,
		authSvc:    authSvc,
		faasEngine: faasEngine,
		templateR:  templateR,
	}

	// Apply global middleware
	r.Use(router.loggingMiddleware)
	r.Use(router.recoveryMiddleware)
	r.Use(protectCoreDir) // Block access to /core/*

	// Public routes (no auth required)
	r.HandleFunc("/health", router.healthHandler).Methods("GET")
	r.HandleFunc("/", router.indexHandler).Methods("GET")

	// Static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.StaticDir))))

	// Auth routes
	authGroup := r.PathPrefix("/auth").Subrouter()
	authGroup.HandleFunc("/login", router.loginHandler).Methods("GET", "POST")
	authGroup.HandleFunc("/logout", router.logoutHandler).Methods("POST")
	authGroup.HandleFunc("/register", router.registerHandler).Methods("GET", "POST")

	// OIDC callback (if enabled)
	if cfg.OIDCEnabled {
		authGroup.HandleFunc("/oidc/callback", router.oidcCallbackHandler).Methods("GET")
		authGroup.HandleFunc("/oidc/login", router.oidcLoginHandler).Methods("GET")
	}

	// Admin routes (require admin role)
	adminGroup := r.PathPrefix("/admin").Subrouter()
	adminGroup.Use(authSvc.RequireAuth())
	adminGroup.Use(authSvc.RequireRole("admin"))

	adminGroup.HandleFunc("/dashboard", router.adminDashboardHandler).Methods("GET")
	adminGroup.HandleFunc("/functions", router.listFunctionsHandler).Methods("GET")
	adminGroup.HandleFunc("/functions/create", router.createFunctionHandler).Methods("GET", "POST")
	adminGroup.HandleFunc("/functions/{id}", router.getFunctionHandler).Methods("GET")
	adminGroup.HandleFunc("/functions/{id}/deploy", router.deployFunctionHandler).Methods("POST")
	adminGroup.HandleFunc("/functions/{id}/delete", router.deleteFunctionHandler).Methods("POST")
	adminGroup.HandleFunc("/functions/{id}/logs", router.getFunctionLogsHandler).Methods("GET")
	adminGroup.HandleFunc("/users", router.listUsersHandler).Methods("GET")
	adminGroup.HandleFunc("/settings", router.settingsHandler).Methods("GET", "POST")

	// Developer routes (require developer or admin role)
	devGroup := r.PathPrefix("/dev").Subrouter()
	devGroup.Use(authSvc.RequireAuth())
	devGroup.Use(authSvc.RequireRole("developer", "admin"))

	devGroup.HandleFunc("/functions", router.devListFunctionsHandler).Methods("GET")
	devGroup.HandleFunc("/functions/upload", router.devUploadFunctionHandler).Methods("POST")
	devGroup.HandleFunc("/functions/{id}/test", router.devTestFunctionHandler).Methods("POST")

	// Function trigger endpoints (public or authenticated, depending on function config)
	// Format: /api/v1/functions/{function_name}
	apiGroup := r.PathPrefix("/api/v1/functions").Subrouter()
	apiGroup.Use(router.functionExecutionMiddleware)
	apiGroup.HandleFunc("/{functionName}", router.triggerFunctionHandler).Methods("GET", "POST", "PUT", "DELETE")

	// Form processing endpoint
	formGroup := r.PathPrefix("/form").Subrouter()
	formGroup.Use(router.formProcessingMiddleware)
	formGroup.HandleFunc("/process", router.processFormHandler).Methods("POST", "GET")

	// Health check for database
	r.HandleFunc("/health/db", router.dbHealthHandler).Methods("GET")

	// 404 handler
	r.NotFoundHandler = http.HandlerFunc(router.notFoundHandler)
	r.MethodNotAllowedHandler = http.HandlerFunc(router.methodNotAllowedHandler)

	return r
}

// loggingMiddleware logs all incoming requests
func (r *Router) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()

		// Generate request ID
		reqID := generateRequestID()
		req.Header.Set("X-Request-ID", reqID)

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, req)

		// Log request
		log.Printf("%s %s %d %v %s",
			req.Method,
			req.URL.Path,
			wrapped.statusCode,
			time.Since(start),
			req.RemoteAddr,
		)
	})
}

// recoveryMiddleware recovers from panics
func (r *Router) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, req)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// protectCoreDir blocks access to the core directory
func protectCoreDir(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, "/core/") {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, req)
	})
}

// generateRequestID creates a unique request identifier
func generateRequestID() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

// notFoundHandler handles 404 errors
func (r *Router) notFoundHandler(w http.ResponseWriter, req *http.Request) {
	http.Error(w, "Not Found", http.StatusNotFound)
}

// methodNotAllowedHandler handles 405 errors
func (r *Router) methodNotAllowedHandler(w http.ResponseWriter, req *http.Request) {
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}
