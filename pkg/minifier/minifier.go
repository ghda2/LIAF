package minifier

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
)

// Minify takes raw file content and minifies it if it is HTML, CSS, JS, or JSON.
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
