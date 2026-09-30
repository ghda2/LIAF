package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestAutomaticSEOGeneration(t *testing.T) {
	mockFS := fstest.MapFS{
		"index.html": {
			Data: []byte("<h1>Home</h1>"),
		},
		"sobre.html": {
			Data: []byte("<h1>Sobre</h1>"),
		},
		"blog/artigo.html": {
			Data: []byte("<h1>Artigo</h1>"),
		},
	}

	SetEmbeddedFS(mockFS)
	defer SetEmbeddedFS(nil)

	server, err := NewServer(ServerOptions{
		Dir:  "non_existent_folder",
		Port: ":8080",
	})
	if err != nil {
		t.Fatalf("erro ao criar servidor: %v", err)
	}

	// 1. Testa /sitemap.xml gerado automaticamente
	reqSitemap := httptest.NewRequest("GET", "/sitemap.xml", nil)
	wSitemap := httptest.NewRecorder()
	server.ServeHTTP(wSitemap, reqSitemap)
	if wSitemap.Code != http.StatusOK {
		t.Fatalf("status sitemap.xml = %d, esperado 200", wSitemap.Code)
	}
	sitemapBody := wSitemap.Body.String()
	for _, expectedRoute := range []string{"<loc>/</loc>", "<loc>/sobre</loc>", "<loc>/blog/artigo</loc>"} {
		if !strings.Contains(sitemapBody, expectedRoute) {
			t.Errorf("sitemap.xml não contém a rota esperada: %s\nCorpo:\n%s", expectedRoute, sitemapBody)
		}
	}

	// 2. Testa /robots.txt gerado automaticamente
	reqRobots := httptest.NewRequest("GET", "/robots.txt", nil)
	wRobots := httptest.NewRecorder()
	server.ServeHTTP(wRobots, reqRobots)
	if wRobots.Code != http.StatusOK {
		t.Fatalf("status robots.txt = %d, esperado 200", wRobots.Code)
	}
	robotsBody := wRobots.Body.String()
	if !strings.Contains(robotsBody, "Sitemap: /sitemap.xml") {
		t.Errorf("robots.txt não aponta para sitemap.xml:\n%s", robotsBody)
	}
}
