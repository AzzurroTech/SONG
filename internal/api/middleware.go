package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/azzurrotech/song/internal/auth"
	"github.com/azzurrotech/song/internal/core"
	"github.com/gorilla/mux"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	UserContextKey      ContextKey = "user"
	RequestIDContextKey ContextKey = "request_id"
	StartTimeContextKey ContextKey = "start_time"
)

// AuthMiddleware wraps the auth service to provide HTTP middleware
type AuthMiddleware struct {
	authSvc *auth.Service
}

// NewAuthMiddleware creates a new auth middleware instance
func NewAuthMiddleware(authSvc *auth.Service) *AuthMiddleware {
	return &AuthMiddleware{authSvc: authSvc}
}

// RequireAuth checks if the user is authenticated
func (am *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		token := auth.ExtractToken(req.Header.Get("Authorization"))
		if token == "" {
			http.Error(w, "Unauthorized: Missing token", http.StatusUnauthorized)
			return
		}

		user, err := am.authSvc.ValidateToken(token)
		if err != nil {
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		// Attach user to context
		ctx := context.WithValue(req.Context(), UserContextKey, user)
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

// RequireRole checks if the user has one of the required roles
func (am *AuthMiddleware) RequireRole(roles ...string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			user, ok := req.Context().Value(UserContextKey).(*auth.User)
			if !ok {
				http.Error(w, "Forbidden: User not found in context", http.StatusForbidden)
				return
			}

			hasRole := false
			for _, role := range roles {
				if user.Role == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				http.Error(w, fmt.Sprintf("Forbidden: Role %s required", strings.Join(roles, " or ")), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, req)
		})
	}
}

// LoggingMiddleware logs request details
func LoggingMiddleware(logger *core.LogManager) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			start := time.Now()

			// Generate request ID if not present
			requestID := req.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = fmt.Sprintf("req-%d", time.Now().UnixNano())
			}

			// Add to headers and context
			req.Header.Set("X-Request-ID", requestID)
			ctx := context.WithValue(req.Context(), RequestIDContextKey, requestID)
			ctx = context.WithValue(ctx, StartTimeContextKey, start)

			// Wrap response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, req.WithContext(ctx))

			// Log the request
			duration := time.Since(start)
			if logger != nil {
				logger.Info("HTTP Request", map[string]interface{}{
					"request_id":   requestID,
					"method":       req.Method,
					"path":         req.URL.Path,
					"status":       wrapped.statusCode,
					"duration_ms":  duration.Milliseconds(),
					"remote_addr":  req.RemoteAddr,
					"user_agent":   req.UserAgent(),
					"content_type": req.Header.Get("Content-Type"),
				})
			}
		})
	}
}

// RecoveryMiddleware recovers from panics and returns a 500 error
func RecoveryMiddleware(logger *core.LogManager) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					requestID := req.Header.Get("X-Request-ID")
					if logger != nil {
						logger.Critical("Panic recovered", map[string]interface{}{
							"request_id": requestID,
							"error":      fmt.Sprintf("%v", err),
							"path":       req.URL.Path,
						})
					}

					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, req)
		})
	}
}

// RateLimitMiddleware limits the number of requests per minute
// Note: This is a simple in-memory implementation. For production, use Redis.
type RateLimitMiddleware struct {
	limits map[string]int
	window time.Duration
	mu     sync.Mutex
}

// NewRateLimitMiddleware creates a new rate limiter
func NewRateLimitMiddleware(requestsPerMinute int) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limits: make(map[string]int),
		window: time.Minute,
	}
}

// Middleware returns the rate limiting middleware
func (rl *RateLimitMiddleware) Middleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ip := getIPAddress(req)

			rl.mu.Lock()
			count, exists := rl.limits[ip]
			if !exists {
				rl.limits[ip] = 1
				rl.mu.Unlock()
				next.ServeHTTP(w, req)
				return
			}

			// Reset window if expired (simplified logic)
			// In production, use a sliding window or token bucket algorithm
			rl.mu.Unlock()

			if count >= requestsPerMinute {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			rl.mu.Lock()
			rl.limits[ip] = count + 1
			rl.mu.Unlock()

			next.ServeHTTP(w, req)
		})
	}
}

// CORSMiddleware adds Cross-Origin Resource Sharing headers
func CORSMiddleware(allowedOrigins []string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			origin := req.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, o := range allowedOrigins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			// Handle preflight
			if req.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, req)
		})
	}
}

// SecurityHeadersMiddleware adds security-related HTTP headers
func SecurityHeadersMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Remove server header to prevent fingerprinting
			w.Header().Del("Server")

			next.ServeHTTP(w, req)
		})
	}
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

// getIPAddress extracts the client IP address
func getIPAddress(req *http.Request) string {
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	if xri := req.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return strings.Split(req.RemoteAddr, ":")[0]
}
