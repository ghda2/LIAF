package web

import (
	"net/http"
	"testing"
)

func TestRateLimiter(t *testing.T) {
	key := "test_ip_1"
	maxReqs := int64(3)
	windowSec := int64(10)

	// 1, 2 e 3 devem passar
	if !RateLimit(key, maxReqs, windowSec) {
		t.Error("1a requisicao deveria passar")
	}
	if !RateLimit(key, maxReqs, windowSec) {
		t.Error("2a requisicao deveria passar")
	}
	if !RateLimit(key, maxReqs, windowSec) {
		t.Error("3a requisicao deveria passar")
	}

	// 4a deve ser bloqueada
	if RateLimit(key, maxReqs, windowSec) {
		t.Error("4a requisicao deveria ser bloqueada pelo rate limit")
	}

	// Outra chave não deve ser afetada
	if !RateLimit("test_ip_2", maxReqs, windowSec) {
		t.Error("requisicao para outra chave deveria passar")
	}
}

func TestClientIP(t *testing.T) {
	req := Request{
		Header: http.Header{
			"X-Forwarded-For": []string{"203.0.113.195, 70.41.3.18"},
		},
	}
	if ip := ClientIP(req); ip != "203.0.113.195" {
		t.Errorf("ClientIP esperado 203.0.113.195, obteve %s", ip)
	}

	req2 := Request{
		Header: http.Header{
			"X-Real-Ip": []string{"198.51.100.1"},
		},
	}
	if ip := ClientIP(req2); ip != "198.51.100.1" {
		t.Errorf("ClientIP esperado 198.51.100.1, obteve %s", ip)
	}
}
