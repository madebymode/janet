package server

import (
	"fmt"
	"net/http"
	"strings"
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