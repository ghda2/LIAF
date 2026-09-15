package web

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/crypto/acme/autocert"
)

type TLSConfig struct {
	Domain   string
	CacheDir string
}

// SetupAutoTLS configures automatic Let's Encrypt certificates for the given domain.
func SetupAutoTLS(domain string) (*autocert.Manager, *tls.Config) {
	cacheDir := filepath.Join(os.TempDir(), "liaf_certs")
	_ = os.MkdirAll(cacheDir, 0700)

	manager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(domain),
		Cache:      autocert.DirCache(cacheDir),
	}

	tlsConfig := manager.TLSConfig()
	return manager, tlsConfig
}

// StartHTTPRedirectServer listens on port 80 and redirects all traffic to HTTPS.
func StartHTTPRedirectServer(manager *autocert.Manager) {
	go func() {
		handler := manager.HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			target := "https://" + r.Host + r.URL.RequestURI()
			http.Redirect(w, r, target, http.StatusMovedPermanently)
		}))

		fmt.Println("[LIAF TLS] Servidor HTTP de redirecionamento ativo na porta 80")
		if err := http.ListenAndServe(":80", handler); err != nil {
			fmt.Printf("[LIAF TLS] Aviso na porta 80: %v\n", err)
		}
	}()
}
