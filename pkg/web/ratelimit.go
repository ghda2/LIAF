package web

import (
	"strings"
	"sync"
	"time"
)

type rateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rateLimitEntry
}

type rateLimitEntry struct {
	count     int64
	resetTime time.Time
}

var globalRateLimiter = &rateLimiter{
	entries: make(map[string]*rateLimitEntry),
}

// RateLimit retorna true se a requisição está dentro do limite configurado.
func RateLimit(key string, maxReqs int64, windowSeconds int64) bool {
	if maxReqs <= 0 || windowSeconds <= 0 {
		return true
	}
	globalRateLimiter.mu.Lock()
	defer globalRateLimiter.mu.Unlock()

	now := time.Now()
	entry, exists := globalRateLimiter.entries[key]
	if !exists || now.After(entry.resetTime) {
		globalRateLimiter.entries[key] = &rateLimitEntry{
			count:     1,
			resetTime: now.Add(time.Duration(windowSeconds) * time.Second),
		}
		return true
	}

	if entry.count < maxReqs {
		entry.count++
		return true
	}

	return false
}

// ClientIP extrai o IP do cliente inspecionando X-Forwarded-For, X-Real-IP ou fallback.
func ClientIP(req Request) string {
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := req.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	return "127.0.0.1"
}
