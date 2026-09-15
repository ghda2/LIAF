package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestServerCacheAndServing(t *testing.T) {
	// Cria pasta temporaria de teste com html e css
	tmpDir, err := os.MkdirTemp("", "liaf_site_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	htmlContent := `
		<!DOCTYPE html>
		<!-- Comentario -->
		<html>
			<body>
				<h1>Ola Teste</h1>
			</body>
		</html>
	`
	cssContent := `
		/* Estilo */
		h1 { color: blue ; }
	`

	os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte(htmlContent), 0644)
	os.WriteFile(filepath.Join(tmpDir, "style.css"), []byte(cssContent), 0644)

	server, err := NewServer(ServerOptions{
		Dir:  tmpDir,
		Port: ":8080",
	})
	if err != nil {
		t.Fatalf("Erro criando servidor: %v", err)
	}

	// 1. Testa requisicao de "/"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Esperava 200 OK, recebido %d", rr.Code)
	}

	etag := rr.Header().Get("ETag")
	if etag == "" {
		t.Errorf("Esperava header ETag gerado")
	}

	// 2. Testa 304 Not Modified com ETag
	req304 := httptest.NewRequest(http.MethodGet, "/", nil)
	req304.Header.Set("If-None-Match", etag)
	rr304 := httptest.NewRecorder()
	server.ServeHTTP(rr304, req304)

	if rr304.Code != http.StatusNotModified {
		t.Errorf("Esperava 304 Not Modified, recebido %d", rr304.Code)
	}

	// 3. Testa compressao Gzip
	reqGzip := httptest.NewRequest(http.MethodGet, "/style.css", nil)
	reqGzip.Header.Set("Accept-Encoding", "gzip")
	rrGzip := httptest.NewRecorder()
	server.ServeHTTP(rrGzip, reqGzip)

	if rrGzip.Code != http.StatusOK {
		t.Errorf("Esperava 200 OK, recebido %d", rrGzip.Code)
	}
	if rrGzip.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("Esperava header Content-Encoding: gzip")
	}
}
