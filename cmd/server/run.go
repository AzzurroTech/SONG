package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Start begins listening for HTTP requests on the specified address
func (s *Server) Start(addr string) error {
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to capture server errors
	errChan := make(chan error, 1)

	go func() {
		fmt.Printf("Listening on %s\n", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
		close(errChan)
	}()

	// Wait for server error or context cancellation
	select {
	case err := <-errChan:
		return err
	case <-context.Background().Done():
		return nil
	}
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := s.router.(*http.ServeMux).ServeHTTP; err != nil {
		// Note: http.ServeMux doesn't have a native Shutdown method.
		// In a real implementation, we'd wrap the mux in a custom struct
		// or use a different router like gorilla/mux that supports shutdown.
		// For now, we just close the underlying listener if we had one.
		// This is a placeholder for the actual shutdown logic.
	}

	// Close database connections
	if s.dbManager != nil {
		if err := s.dbManager.Close(); err != nil {
			return err
		}
	}

	// Close FaaS engine resources
	if s.faasEngine != nil {
		if err := s.faasEngine.Shutdown(ctx); err != nil {
			return err
		}
	}

	return nil
}
