package web

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

var assetRefRegex = regexp.MustCompile(`(href|src)=["']([^"':\s?#]+\.(?:css|js|png|jpg|jpeg|svg|webp|ico|gif|woff2?))(?:\?[^"'\s]*)?["']`)

func fingerprintHTML(htmlRelDir string, raw []byte, hashes map[string]string) []byte {
	content := string(raw)

	replaced := assetRefRegex.ReplaceAllStringFunc(content, func(m string) string {
		sub := assetRefRegex.FindStringSubmatch(m)
		if len(sub) < 3 {
			return m
		}
		attr := sub[1]
		targetPath := sub[2]

		// 1. Tenta relativo à pasta do HTML
		relKey := targetPath
		if !strings.HasPrefix(relKey, "/") {
			relKey = path.Clean(path.Join(htmlRelDir, targetPath))
		}
		if hash, ok := hashes[relKey]; ok {
			return fmt.Sprintf(`%s="%s?v=%s"`, attr, targetPath, hash)
		}

		// 2. Tenta a partir da raiz (ex: /style.css)
		rootKey := targetPath
		if !strings.HasPrefix(rootKey, "/") {
			rootKey = "/" + rootKey
		}
		if hash, ok := hashes[rootKey]; ok {
			return fmt.Sprintf(`%s="%s?v=%s"`, attr, targetPath, hash)
		}

		return m
	})

	return []byte(replaced)
}
