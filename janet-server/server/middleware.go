package server

import (
	"encoding/json"
	"fmt"
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

		// Log the request details to stdout
		fmt.Printf("[%s] %s %s %s - %d - %v - %s\n",
			start.Format("2006-01-02 15:04:05"),
			clientIP,
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			duration,
			r.URL.RawQuery,
		)
	})
}

// getRealClientIP extracts the real client IP from common proxy headers
func getRealClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (most common)
	if xForwardedFor := r.Header.Get("X-Forwarded-For"); xForwardedFor != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header (Nginx)
	if xRealIP := r.Header.Get("X-Real-IP"); xRealIP != "" {
		return xRealIP
	}

	// Check CF-Connecting-IP header (Cloudflare)
	if cfConnectingIP := r.Header.Get("CF-Connecting-IP"); cfConnectingIP != "" {
		return cfConnectingIP
	}

	// Check True-Client-IP header (Akamai, Cloudflare)
	if trueClientIP := r.Header.Get("True-Client-IP"); trueClientIP != "" {
		return trueClientIP
	}

	// Check X-Client-IP header
	if xClientIP := r.Header.Get("X-Client-IP"); xClientIP != "" {
		return xClientIP
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
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
		// Strict Transport Security - Force HTTPS for 1 year
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		// Content Security Policy - Restrict resource loading
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"font-src 'self'; " +
			"connect-src 'self'; " +
			"frame-ancestors 'none'"
		w.Header().Set("Content-Security-Policy", csp)

		// X-Frame-Options - Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// X-Content-Type-Options - Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// X-XSS-Protection - Enable XSS filtering (legacy browsers)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Referrer-Policy - Control referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions-Policy - Restrict browser features
		w.Header().Set("Permissions-Policy", "()")

		// X-Permitted-Cross-Domain-Policies - Block Adobe Flash/PDF cross-domain requests
		w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")

		next.ServeHTTP(w, r)
	})
}
