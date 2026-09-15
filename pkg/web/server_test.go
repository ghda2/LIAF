package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
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

func TestTemplateLayoutAndIncludes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "liaf_template_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	baseHTML := `<!DOCTYPE html><html><body><!-- include "menu.html" --><main><!-- content --></main><!-- include "footer.html" --></body></html>`
	menuHTML := `<nav><a href="/">Home</a><a href="/pagina1">P1</a></nav>`
	footerHTML := `<footer><p>Rodapé LIAF</p></footer>`
	p1HTML := `<!-- layout "base.html" --><h1>Conteúdo P1</h1>`

	os.WriteFile(filepath.Join(tmpDir, "base.html"), []byte(baseHTML), 0644)
	os.WriteFile(filepath.Join(tmpDir, "menu.html"), []byte(menuHTML), 0644)
	os.WriteFile(filepath.Join(tmpDir, "footer.html"), []byte(footerHTML), 0644)
	os.WriteFile(filepath.Join(tmpDir, "pagina1.html"), []byte(p1HTML), 0644)

	server, err := NewServer(ServerOptions{
		Dir:  tmpDir,
		Port: ":8080",
	})
	if err != nil {
		t.Fatalf("Erro criando servidor: %v", err)
	}

	// Testa Clean URL "/pagina1"
	req := httptest.NewRequest(http.MethodGet, "/pagina1", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Esperava 200 OK para /pagina1, recebido %d", rr.Code)
	}

	body := rr.Body.String()
	// Confirma se o menu foi incluído
	if !strings.Contains(body, "Home") || !strings.Contains(body, "P1") {
		t.Errorf("Menu não encontrado no corpo renderizado: %s", body)
	}
	// Confirma se o conteúdo foi injetado
	if !strings.Contains(body, "Conteúdo P1") {
		t.Errorf("Conteúdo não encontrado no corpo renderizado: %s", body)
	}
	// Confirma se o footer foi incluído
	if !strings.Contains(body, "Rodapé LIAF") {
		t.Errorf("Footer não encontrado no corpo renderizado: %s", body)
	}
}

func TestAutomaticAssetFingerprinting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "liaf_fingerprint_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	htmlContent := `<!DOCTYPE html><html><head><link rel="stylesheet" href="/style.css"></head><body><script src="/app.js"></script></body></html>`
	cssContent := `body { background: black; }`
	jsContent := `console.log("hello");`

	os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte(htmlContent), 0644)
	os.WriteFile(filepath.Join(tmpDir, "style.css"), []byte(cssContent), 0644)
	os.WriteFile(filepath.Join(tmpDir, "app.js"), []byte(jsContent), 0644)

	server, err := NewServer(ServerOptions{
		Dir:  tmpDir,
		Port: ":8080",
	})
	if err != nil {
		t.Fatalf("Erro criando servidor: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	body := rr.Body.String()
	if !strings.Contains(body, `href="/style.css?v=`) {
		t.Errorf("Esperava href com versionamento automático ?v=, recebido: %s", body)
	}
	if !strings.Contains(body, `src="/app.js?v=`) {
		t.Errorf("Esperava src com versionamento automático ?v=, recebido: %s", body)
	}
}

func TestSandboxedVFS(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "liaf_vfs_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Cria estrutura com subpasta, json, css na raiz e pagina dentro da subpasta
	subDir := filepath.Join(tmpDir, "blog")
	os.MkdirAll(subDir, 0755)

	os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte("<h1>Root</h1>"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "style.css"), []byte("body { margin: 0; }"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "data.json"), []byte("{\n  \"status\": \"ok\",\n  \"count\": 42\n}"), 0644)
	os.WriteFile(filepath.Join(subDir, "index.html"), []byte("<h1>Blog Home</h1>"), 0644)
	os.WriteFile(filepath.Join(subDir, "post.html"), []byte("<h1>Post 1</h1>"), 0644)

	server, err := NewServer(ServerOptions{
		Dir:  tmpDir,
		Port: ":8080",
	})
	if err != nil {
		t.Fatalf("Erro criando servidor: %v", err)
	}

	// 1. Testa JSON servido e minificado
	reqJSON := httptest.NewRequest(http.MethodGet, "/data.json", nil)
	rrJSON := httptest.NewRecorder()
	server.ServeHTTP(rrJSON, reqJSON)
	if rrJSON.Code != http.StatusOK {
		t.Errorf("Esperava 200 para /data.json, recebido %d", rrJSON.Code)
	}
	if !strings.Contains(rrJSON.Header().Get("Content-Type"), "application/json") {
		t.Errorf("Esperava MIME application/json, recebido %s", rrJSON.Header().Get("Content-Type"))
	}
	if strings.Contains(rrJSON.Body.String(), "\n") {
		t.Errorf("JSON deveria ter sido minificado sem quebras de linha: %s", rrJSON.Body.String())
	}

	// 2. Testa subpasta /blog/
	reqSub := httptest.NewRequest(http.MethodGet, "/blog/", nil)
	rrSub := httptest.NewRecorder()
	server.ServeHTTP(rrSub, reqSub)
	if !strings.Contains(rrSub.Body.String(), "Blog Home") {
		t.Errorf("Esperava Blog Home em /blog/, recebido %s", rrSub.Body.String())
	}

	// 3. Testa clean URL em subpasta /blog/post
	reqPost := httptest.NewRequest(http.MethodGet, "/blog/post", nil)
	rrPost := httptest.NewRecorder()
	server.ServeHTTP(rrPost, reqPost)
	if !strings.Contains(rrPost.Body.String(), "Post 1") {
		t.Errorf("Esperava Post 1 em /blog/post, recebido %s", rrPost.Body.String())
	}

	// 4. Testa fallback de asset relativo da subpasta para a raiz (/blog/style.css => /style.css)
	reqRelCSS := httptest.NewRequest(http.MethodGet, "/blog/style.css", nil)
	rrRelCSS := httptest.NewRecorder()
	server.ServeHTTP(rrRelCSS, reqRelCSS)
	if rrRelCSS.Code != http.StatusOK {
		t.Errorf("Esperava fallback do asset para raiz, recebido %d", rrRelCSS.Code)
	}

	// 5. Testa que arquivo ausente com extensão retorna 404 (sem cair no index.html)
	req404 := httptest.NewRequest(http.MethodGet, "/imagem-inexistente.png", nil)
	rr404 := httptest.NewRecorder()
	server.ServeHTTP(rr404, req404)
	if rr404.Code != http.StatusNotFound {
		t.Errorf("Esperava 404 para asset inexistente, recebido %d", rr404.Code)
	}

	// 6. Testa Jail/Sandbox contra Path Traversal
	reqJail := httptest.NewRequest(http.MethodGet, "/../../../../etc/passwd", nil)
	rrJail := httptest.NewRecorder()
	server.ServeHTTP(rrJail, reqJail)
	if rrJail.Code != http.StatusNotFound {
		t.Errorf("Esperava 404 para path traversal, recebido %d", rrJail.Code)
	}
}

func TestSSGMarkdownAndCollections(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "liaf_ssg_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Layout base
	baseHTML := `<!DOCTYPE html>
<html>
<head><title>{{ title }} - Meu Site</title></head>
<body>
  <!-- content -->
</body>
</html>`
	os.WriteFile(filepath.Join(tmpDir, "base.html"), []byte(baseHTML), 0644)

	// Arquivo de dados JSON
	os.MkdirAll(filepath.Join(tmpDir, "data"), 0755)
	servicosJSON := `[
		{"nome": "Consultoria IA", "preco": "1000"},
		{"nome": "Compiladores Web", "preco": "2000"}
	]`
	os.WriteFile(filepath.Join(tmpDir, "data", "servicos.json"), []byte(servicosJSON), 0644)

	// Markdown com Frontmatter
	os.MkdirAll(filepath.Join(tmpDir, "blog"), 0755)
	postMD := `---
title: "Artigo Exclusivo LIAF"
date: "2026-09-15"
layout: "base.html"
---
# Introducao ao LIAF

Texto do post em **Markdown** com listas:
- Alta performance
- Baixo consumo de RAM
`
	os.WriteFile(filepath.Join(tmpDir, "blog", "artigo.md"), []byte(postMD), 0644)

	// Página index com loops sobre 'servicos' e 'posts'
	indexHTML := `<!-- layout "base.html" -->
<h1>Bem-vindo</h1>
<ul class="servicos">
<!-- for s in servicos -->
  <li>{{ s.nome }} por R$ {{ s.preco }}</li>
<!-- endfor -->
</ul>
<ul class="posts">
<!-- for p in posts -->
  <li><a href="{{ p.url }}">{{ p.title }}</a> - {{ p.date }}</li>
<!-- endfor -->
</ul>
`
	os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte(indexHTML), 0644)

	server, err := NewServer(ServerOptions{
		Dir:  tmpDir,
		Port: ":8080",
	})
	if err != nil {
		t.Fatalf("Erro criando servidor SSG: %v", err)
	}

	// 1. Testa renderização do Markdown em clean URL (/blog/artigo)
	reqMD := httptest.NewRequest(http.MethodGet, "/blog/artigo", nil)
	rrMD := httptest.NewRecorder()
	server.ServeHTTP(rrMD, reqMD)

	if rrMD.Code != http.StatusOK {
		t.Fatalf("Esperava 200 em /blog/artigo, recebido %d", rrMD.Code)
	}
	bodyMD := rrMD.Body.String()
	if !strings.Contains(bodyMD, "<h1>Introducao ao LIAF</h1>") {
		t.Errorf("Markdown não renderizou H1 esperado: %s", bodyMD)
	}
	if !strings.Contains(bodyMD, "<strong>Markdown</strong>") {
		t.Errorf("Markdown não renderizou negrito: %s", bodyMD)
	}
	if !strings.Contains(bodyMD, "<li>Alta performance</li>") {
		t.Errorf("Markdown não renderizou lista: %s", bodyMD)
	}
	if !strings.Contains(bodyMD, "<title>Artigo Exclusivo LIAF - Meu Site</title>") {
		t.Errorf("Variável do frontmatter no layout não foi substituída: %s", bodyMD)
	}

	// 2. Testa loops no index.html
	reqIndex := httptest.NewRequest(http.MethodGet, "/", nil)
	rrIndex := httptest.NewRecorder()
	server.ServeHTTP(rrIndex, reqIndex)

	if rrIndex.Code != http.StatusOK {
		t.Fatalf("Esperava 200 em /, recebido %d", rrIndex.Code)
	}
	bodyIndex := rrIndex.Body.String()
	if !strings.Contains(bodyIndex, "<li>Consultoria IA por R$ 1000</li>") {
		t.Errorf("Loop de dados JSON 'servicos' falhou: %s", bodyIndex)
	}
	if !strings.Contains(bodyIndex, "<li>Compiladores Web por R$ 2000</li>") {
		t.Errorf("Loop de dados JSON 'servicos' falhou no segundo item: %s", bodyIndex)
	}
	if !strings.Contains(bodyIndex, `<a href="/blog/artigo">Artigo Exclusivo LIAF</a> - 2026-09-15`) {
		t.Errorf("Loop de coleção 'posts' gerado a partir do Markdown falhou: %s", bodyIndex)
	}
}

func TestEmbeddedFS(t *testing.T) {
	mapFS := fstest.MapFS{
		"base.html": &fstest.MapFile{
			Data: []byte(`<!DOCTYPE html><html><body><!-- content --></body></html>`),
		},
		"index.html": &fstest.MapFile{
			Data: []byte(`<!-- layout "base.html" --><h1>Single Binary LIAF</h1>`),
		},
		"style.css": &fstest.MapFile{
			Data: []byte(`body { margin: 0; }`),
		},
	}

	SetEmbeddedFS(mapFS)
	defer SetEmbeddedFS(nil)

	server, err := NewServer(ServerOptions{
		Dir:  "non_existent_folder",
		Port: ":8080",
	})
	if err != nil {
		t.Fatalf("Erro criando servidor com VFS embutido: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Esperava 200 OK do VFS embutido, recebido %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "<h1>Single Binary LIAF</h1>") {
		t.Errorf("Conteudo embutido nao foi servido corretamente: %s", rr.Body.String())
	}
}


