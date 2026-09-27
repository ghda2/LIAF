package minifier

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Minify takes raw file content and minifies it if it is HTML, CSS, or JSON.
// JS minification is disabled by default to avoid breaking modern scripts and bundles.
// Minification can be completely disabled by setting LIAF_MINIFY=false (or LIAF_NO_MINIFY=1).
func Minify(filename string, content []byte) []byte {
	// 1. Respeita opção explícita de ambiente para desligar minificação (ex: LIAF_MINIFY=false)
	minifyEnv := strings.ToLower(strings.TrimSpace(os.Getenv("LIAF_MINIFY")))
	noMinifyEnv := strings.ToLower(strings.TrimSpace(os.Getenv("LIAF_NO_MINIFY")))
	if minifyEnv == "false" || minifyEnv == "0" || minifyEnv == "off" || minifyEnv == "no" ||
		noMinifyEnv == "1" || noMinifyEnv == "true" {
		return content
	}

	ext := strings.ToLower(filepath.Ext(filename))
	base := strings.ToLower(filepath.Base(filename))
	normalizedPath := filepath.ToSlash(strings.ToLower(filename))

	// 2. Nunca aplica minificação em bundles compilados (Vite/Webpack), .min.* ou diretórios de assets
	if strings.Contains(base, ".min.") ||
		strings.Contains(normalizedPath, "/assets/") ||
		strings.HasPrefix(normalizedPath, "assets/") ||
		strings.Contains(base, "-") {
		return content
	}

	str := string(content)

	switch ext {
	case ".html", ".htm":
		return []byte(MinifyHTML(str))
	case ".css":
		return []byte(MinifyCSS(str))
	case ".js", ".mjs":
		// Minificação de JS desativada para manter scripts e bundles íntegros
		return content
	case ".json":
		var buf bytes.Buffer
		if err := json.Compact(&buf, content); err == nil {
			return buf.Bytes()
		}
		return content
	default:
		return content
	}
}
