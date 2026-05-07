package server

import (
	"encoding/json"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// requestLoggingMiddleware logs all incoming requests to stdout
func (s *Server) requestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom ResponseWriter to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     200, // default status code
		}

		// Process the request
		next.ServeHTTP(wrapped, r)

		// Calculate duration
		duration := time.Since(start)

		// Get the real client IP address
		clientIP := getRealClientIP(r)

		s.logger.KV("client_ip", clientIP).
			KV("method", r.Method).
			KV("path", r.URL.Path).
			KV("status", wrapped.statusCode).
			KV("duration", duration.String()).
			KV("query", r.URL.RawQuery).
			Info("http_request")
	})
}

// getRealClientIP extracts the real client IP from common proxy headers
func getRealClientIP(r *http.Request) string {
	for _, header := range []string{
		"X-Forwarded-For",
		"X-Real-IP",
		"CF-Connecting-IP",
		"True-Client-IP",
		"X-Client-IP",
	} {
		if value := firstValidIP(r.Header.Get(header)); value != "" {
			return value
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && net.ParseIP(host) != nil {
		return host
	}
	if net.ParseIP(r.RemoteAddr) != nil {
		return r.RemoteAddr
	}
	return ""
}

func firstValidIP(value string) string {
	for _, candidate := range strings.Split(value, ",") {
		candidate = strings.TrimSpace(candidate)
		if net.ParseIP(candidate) != nil {
			return candidate
		}
	}
	return ""
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

type rateLimitClient struct {
	tokens float64
	last   time.Time
}

type rateLimiter struct {
	mu      sync.Mutex
	rate    float64
	burst   float64
	clients map[string]*rateLimitClient
}

func newRateLimiter(rate float64, burst int) *rateLimiter {
	if rate <= 0 {
		rate = 20
	}
	if burst <= 0 {
		burst = int(rate * 3)
	}
	return &rateLimiter{
		rate:    rate,
		burst:   float64(burst),
		clients: make(map[string]*rateLimitClient),
	}
}

func (rl *rateLimiter) allow(clientID string) (bool, time.Duration) {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	client := rl.clients[clientID]
	if client == nil {
		client = &rateLimitClient{
			tokens: rl.burst,
			last:   now,
		}
		rl.clients[clientID] = client
	}

	elapsed := now.Sub(client.last).Seconds()
	if elapsed > 0 {
		client.tokens = math.Min(rl.burst, client.tokens+(elapsed*rl.rate))
		client.last = now
	}

	if client.tokens >= 1 {
		client.tokens -= 1
		return true, 0
	}

	needed := 1 - client.tokens
	retrySeconds := needed / rl.rate
	if retrySeconds < 0 {
		retrySeconds = 0
	}
	return false, time.Duration(math.Ceil(retrySeconds * float64(time.Second)))
}

func normalizeClientIP(ip string) string {
	if host, _, err := net.SplitHostPort(ip); err == nil {
		return host
	}
	return ip
}

func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.rateLimiter == nil {
			next.ServeHTTP(w, r)
			return
		}

		clientIP := normalizeClientIP(getRealClientIP(r))
		allowed, retryAfter := s.rateLimiter.allow(clientIP)
		if allowed {
			next.ServeHTTP(w, r)
			return
		}

		if retryAfter < time.Second {
			retryAfter = time.Second
		}
		w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error":       "rate_limited",
				"retry_after": int(retryAfter.Seconds()),
			})
			return
		}

		http.Error(w, "Rate limit exceeded. Try again shortly.", http.StatusTooManyRequests)
	})
}

// securityHeadersMiddleware adds essential security headers to all responses
func (s *Server) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		csp := "default-src 'self'; " +
			"base-uri 'self'; " +
			"object-src 'none'; " +
			"script-src 'self'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"font-src 'self'; " +
			"connect-src 'self'; " +
			"frame-ancestors 'none'; " +
			"form-action 'self'"
		w.Header().Set("Content-Security-Policy", csp)
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "()")
		w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		w.Header().Set("Origin-Agent-Cluster", "?1")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}

		next.ServeHTTP(w, r)
	})
}
