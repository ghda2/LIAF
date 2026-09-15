package minifier

import (
	"path/filepath"
	"strings"
)

// Minify takes raw file content and minifies it if it is HTML, CSS, or JS.
func Minify(filename string, content []byte) []byte {
	ext := strings.ToLower(filepath.Ext(filename))
	str := string(content)

	switch ext {
	case ".html", ".htm":
		return []byte(MinifyHTML(str))
	case ".css":
		return []byte(MinifyCSS(str))
	case ".js", ".mjs":
		return []byte(MinifyJS(str))
	default:
		return content
	}
}
