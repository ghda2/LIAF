package web

import (
	"crypto/sha1"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"liaf/pkg/minifier"
)

type CachedAsset struct {
	Path        string
	ContentType string
	Content     []byte
	GzipContent []byte
	ETag        string
	Size        int
}

type AssetCache struct {
	mu     sync.RWMutex
	assets map[string]*CachedAsset
}

func NewAssetCache() *AssetCache {
	return &AssetCache{
		assets: make(map[string]*CachedAsset),
	}
}

func (ac *AssetCache) LoadDirectory(dirPath string) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		rel, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}

		relPath := "/" + strings.ReplaceAll(rel, "\\", "/")

		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Minify
		minified := minifier.Minify(path, raw)

		// Pre-gzip
		gzipped, _ := CompressGzip(minified)

		// MIME type
		ext := filepath.Ext(path)
		mimeType := mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		// ETag (SHA-1 hash of minified content)
		etag := fmt.Sprintf("\"%x\"", sha1.Sum(minified))

		asset := &CachedAsset{
			Path:        relPath,
			ContentType: mimeType,
			Content:     minified,
			GzipContent: gzipped,
			ETag:        etag,
			Size:        len(minified),
		}

		ac.assets[relPath] = asset

		// If it's index.html, also register root "/" and directory paths
		if relPath == "/index.html" {
			ac.assets["/"] = asset
		} else if strings.HasSuffix(relPath, "/index.html") {
			dirRoute := strings.TrimSuffix(relPath, "index.html")
			ac.assets[dirRoute] = asset
			ac.assets[strings.TrimSuffix(dirRoute, "/")] = asset
		}

		return nil
	})
}

func (ac *AssetCache) Get(route string) (*CachedAsset, bool) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	asset, ok := ac.assets[route]
	return asset, ok
}

func (ac *AssetCache) Count() int {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return len(ac.assets)
}
