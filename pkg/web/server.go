package web

import (
	"fmt"
	"net/http"
	"path"
	"strings"
	"sync/atomic"
	"time"
)

type ServerOptions struct {
	Dir     string
	Domain  string
	Port    string
	AutoTLS bool
}

type Server struct {
	cache   atomic.Pointer[AssetCache]
	options ServerOptions
}

func NewServer(opts ServerOptions) (*Server, error) {
	s := &Server{
		options: opts,
	}

	if err := s.Reload(); err != nil {
		return nil, fmt.Errorf("falha carregando diretorio '%s': %w", opts.Dir, err)
	}

	return s, nil
}

// Reload recarrega todos os arquivos da pasta em RAM atomicamente sem downtime.
func (s *Server) Reload() error {
	newCache := NewAssetCache()
	if err := newCache.LoadDirectory(s.options.Dir); err != nil {
		return err
	}
	s.cache.Store(newCache)
	fmt.Printf("🔄 [LIAF VFS] Cache recarregado atomicamente em RAM: %d arquivos\n", newCache.Count())
	return nil
}

func (s *Server) CurrentCache() *AssetCache {
	return s.cache.Load()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Endpoint de Hot-Reload seguro em memória
	if r.URL.Path == "/_liaf/reload" {
		start := time.Now()
		if err := s.Reload(); err != nil {
			http.Error(w, fmt.Sprintf(`{"status":"error","message":%q}`, err.Error()), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","message":"VFS recarregado em RAM","files":%d,"duration_ms":%d}`, s.CurrentCache().Count(), time.Since(start).Milliseconds())
		return
	}

	cache := s.CurrentCache()

	// Bloqueia qualquer tentativa de Directory/Path Traversal
	if strings.Contains(r.URL.Path, "..") {
		http.NotFound(w, r)
		return
	}

	// Jail / Sandbox: sanitiza caminho para sempre ficar restrito à raiz virtual "/"
	cleanPath := path.Clean("/" + r.URL.Path)

	// 1. Busca exata do asset no VFS (ex: /style.css, /img/logo.png, /api/data.json)
	asset, found := cache.Get(cleanPath)

	// 2. Se for diretório (ex: "/" ou "/blog/"), busca index.html naquele diretório
	if !found {
		if cleanPath == "/" {
			asset, found = cache.Get("/index.html")
		} else {
			asset, found = cache.Get(cleanPath + "/index.html")
		}
	}

	// 3. URLs limpas para HTML (ex: /sobre => /sobre.html)
	if !found {
		asset, found = cache.Get(cleanPath + ".html")
	}

	// 4. Fallback de assets relativos compartilhados:
	// Se uma subpasta pediu "./style.css" (recebido como /sub/style.css), mas o CSS está na raiz /style.css
	if !found && path.Dir(cleanPath) != "/" {
		rootFallback := "/" + path.Base(cleanPath)
		asset, found = cache.Get(rootFallback)
	}

	// 5. Fallback para SPA: apenas se a rota NÃO tiver extensão de arquivo
	if !found {
		ext := path.Ext(cleanPath)
		if ext == "" {
			asset, found = cache.Get("/index.html")
		}
	}

	// Se ainda não encontrou (ex: imagem ou script inexistente), retorna 404
	if !found {
		http.NotFound(w, r)
		return
	}

	// 1. ETag & Cache Validation (304 Not Modified)
	if match := r.Header.Get("If-None-Match"); match != "" && match == asset.ETag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("ETag", asset.ETag)
	w.Header().Set("Content-Type", asset.ContentType)

	// Controle inteligente de cache:
	// HTML nunca fica em cache obsoleto (sempre traz os novos hashes de CSS/JS)
	if strings.HasPrefix(asset.ContentType, "text/html") {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	} else if r.URL.Query().Get("v") != "" {
		// Assets com hash na URL (?v=hash) são imutáveis e ultrarrápidos
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	}

	w.Header().Set("X-Powered-By", "LIAF Web Engine")

	// 2. Gzip Compression Check
	acceptsGzip := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
	if acceptsGzip && len(asset.GzipContent) > 0 {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		w.WriteHeader(http.StatusOK)
		w.Write(asset.GzipContent)
		return
	}

	// 3. Fallback to uncompressed
	w.WriteHeader(http.StatusOK)
	w.Write(asset.Content)
}

func (s *Server) Start() error {
	port := s.options.Port
	if port == "" {
		port = ":8080"
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	fmt.Printf("⚡ [LIAF Web Engine] %d arquivos carregados, minificados e comprimidos em RAM\n", s.CurrentCache().Count())
	fmt.Printf("🔄 [LIAF VFS] Hot-reload disponível em POST/GET /_liaf/reload\n")

	// Modo Auto-TLS (Caddy Style)
	if s.options.AutoTLS && s.options.Domain != "" && s.options.Domain != "localhost" {
		manager, tlsConfig := SetupAutoTLS(s.options.Domain)
		StartHTTPRedirectServer(manager)

		server := &http.Server{
			Addr:         ":443",
			Handler:      s,
			TLSConfig:    tlsConfig,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		}

		fmt.Printf("🔒 [LIAF Web Engine] Rodando com HTTPS/Auto-TLS em https://%s:443\n", s.options.Domain)
		return server.ListenAndServeTLS("", "")
	}

	// Modo HTTP Local / Standard
	fmt.Printf("🚀 [LIAF Web Engine] Rodando em http://localhost%s\n", port)
	return http.ListenAndServe(port, s)
}

// ServeSite is the high-level entrypoint exposed to LIAF programs.
func ServeSite(dir string, domain string, port string, autoTLS bool) {
	opts := ServerOptions{
		Dir:     dir,
		Domain:  domain,
		Port:    port,
		AutoTLS: autoTLS,
	}

	srv, err := NewServer(opts)
	if err != nil {
		fmt.Printf("Erro iniciando servidor: %v\n", err)
		return
	}

	if err := srv.Start(); err != nil {
		fmt.Printf("Erro de execucao do servidor: %v\n", err)
	}
}
