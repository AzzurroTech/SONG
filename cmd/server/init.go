package main

import (
	"net/http"

	"github.com/azzurrotech/song/internal/api"
	"github.com/azzurrotech/song/internal/auth"
	"github.com/azzurrotech/song/internal/config"
	"github.com/azzurrotech/song/internal/core"
	"github.com/azzurrotech/song/internal/db"
	"github.com/azzurrotech/song/internal/faas"
	"github.com/azzurrotech/song/internal/templates"
)

// Server represents the SONG HTTP server
type Server struct {
	config     *config.Config
	coreStore  *core.Store
	dbManager  *db.Manager
	router     *http.ServeMux
	authSvc    *auth.Service
	faasEngine *faas.Engine
	templateR  *templates.Renderer
}

// NewServer creates and configures a new Server instance
func NewServer(cfg *config.Config, coreStore *core.Store, dbManager *db.Manager) (*Server, error) {
	s := &Server{
		config:    cfg,
		coreStore: coreStore,
		dbManager: dbManager,
	}

	// Initialize authentication service
	authSvc, err := auth.NewService(cfg, coreStore)
	if err != nil {
		return nil, err
	}
	s.authSvc = authSvc

	// Initialize FaaS engine
	faasEngine, err := faas.NewEngine(cfg, coreStore, dbManager)
	if err != nil {
		return nil, err
	}
	s.faasEngine = faasEngine

	// Initialize template renderer
	templateRenderer, err := templates.NewRenderer(cfg.TemplateDir)
	if err != nil {
		return nil, err
	}
	s.templateR = templateRenderer

	// Initialize router and register routes
	s.router = api.SetupRoutes(cfg, coreStore, dbManager, authSvc, faasEngine, templateRenderer)

	return s, nil
}
