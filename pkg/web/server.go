package web

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

type ServerOptions struct {
	Dir      string
	Domain   string
	Port     string
	AutoTLS  bool
}

type Server struct {
	cache   *AssetCache
	options ServerOptions
}

func NewServer(opts ServerOptions) (*Server, error) {
	cache := NewAssetCache()
	if err := cache.LoadDirectory(opts.Dir); err != nil {
		return nil, fmt.Errorf("falha carregando diretorio '%s': %w", opts.Dir, err)
	}

	return &Server{
		cache:   cache,
		options: opts,
	}, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	asset, found := s.cache.Get(path)
	if !found {
		// Fallback para index.html se for SPA (Single Page Application)
		asset, found = s.cache.Get("/index.html")
		if !found {
			http.NotFound(w, r)
			return
		}
	}

	// 1. ETag & Cache Validation (304 Not Modified)
	if match := r.Header.Get("If-None-Match"); match != "" && match == asset.ETag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("ETag", asset.ETag)
	w.Header().Set("Content-Type", asset.ContentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
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

	fmt.Printf("⚡ [LIAF Web Engine] %d arquivos carregados, minificados e comprimidos em RAM\n", s.cache.Count())

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
