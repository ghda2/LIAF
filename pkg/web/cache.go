package web

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/fs"
	"mime"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"liaf/pkg/markdown"
	"liaf/pkg/minifier"
)

var defaultMimes = map[string]string{
	".html":  "text/html; charset=utf-8",
	".htm":   "text/html; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".js":    "application/javascript; charset=utf-8",
	".mjs":   "application/javascript; charset=utf-8",
	".json":  "application/json; charset=utf-8",
	".xml":   "application/xml; charset=utf-8",
	".svg":   "image/svg+xml",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".webp":  "image/webp",
	".avif":  "image/avif",
	".ico":   "image/x-icon",
	".wasm":  "application/wasm",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ttf":   "font/ttf",
	".otf":   "font/otf",
	".txt":   "text/plain; charset=utf-8",
	".pdf":   "application/pdf",
	".mp4":   "video/mp4",
	".webm":  "video/webm",
	".mp3":   "audio/mpeg",
}

func detectContentType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	if m, ok := defaultMimes[ext]; ok {
		return m
	}
	if m := mime.TypeByExtension(ext); m != "" {
		return m
	}
	return "application/octet-stream"
}

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

var (
	embeddedMu sync.RWMutex
	embeddedFS fs.FS
)

// SetEmbeddedFS registra um sistema de arquivos virtual embutido (ex: embed.FS) para Single-Binary deploy.
func SetEmbeddedFS(fileSys fs.FS) {
	embeddedMu.Lock()
	defer embeddedMu.Unlock()
	embeddedFS = fileSys
}

func GetEmbeddedFS() fs.FS {
	embeddedMu.RLock()
	defer embeddedMu.RUnlock()
	return embeddedFS
}

func NewAssetCache() *AssetCache {
	return &AssetCache{
		assets: make(map[string]*CachedAsset),
	}
}

func (ac *AssetCache) LoadDirectory(dirPath string) error {
	if efs := GetEmbeddedFS(); efs != nil {
		return ac.LoadFS(efs)
	}

	ac.mu.Lock()
	defer ac.mu.Unlock()

	type rawHTMLItem struct {
		path      string
		relPath   string
		processed []byte
		vars      map[string]string
	}

	type rawMDItem struct {
		path    string
		relPath string
		doc     *markdown.Document
	}

	var htmlItems []rawHTMLItem
	var mdItems []rawMDItem
	assetHashes := make(map[string]string)
	collections := make(map[string][]map[string]string)

	// Pass 1: Percorrer todos os arquivos, indexar dados, estáticos, HTML e Markdown
	err := filepath.Walk(dirPath, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		rel, err := filepath.Rel(dirPath, p)
		if err != nil {
			return err
		}

		relPath := "/" + strings.ReplaceAll(rel, "\\", "/")
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}

		ext := strings.ToLower(filepath.Ext(p))

		// 1.1 Coleções em JSON (ex: public/data/servicos.json)
		if strings.HasPrefix(relPath, "/data/") && ext == ".json" {
			colName := strings.TrimSuffix(filepath.Base(p), ext)
			var rawList []map[string]any
			if err := json.Unmarshal(raw, &rawList); err == nil {
				var items []map[string]string
				for _, obj := range rawList {
					item := make(map[string]string)
					for k, v := range obj {
						item[k] = fmt.Sprintf("%v", v)
					}
					items = append(items, item)
				}
				collections[colName] = items
			}
		}

		// 1.2 Arquivos Markdown (.md, .markdown)
		if ext == ".md" || ext == ".markdown" {
			doc := markdown.Parse(string(raw))
			cleanRoute := strings.TrimSuffix(relPath, ext)
			
			// Registrar na coleção de posts/artigos
			postItem := make(map[string]string)
			for k, v := range doc.Frontmatter {
				postItem[k] = v
			}
			postItem["url"] = cleanRoute
			if _, ok := postItem["title"]; !ok {
				postItem["title"] = path.Base(cleanRoute)
			}
			collections["posts"] = append(collections["posts"], postItem)

			mdItems = append(mdItems, rawMDItem{
				path:    p,
				relPath: relPath,
				doc:     doc,
			})
			return nil
		}

		// 1.3 Arquivos HTML
		if ext == ".html" || ext == ".htm" {
			processed := processHTML(dirPath, p, raw, 0)
			htmlItems = append(htmlItems, rawHTMLItem{
				path:      p,
				relPath:   relPath,
				processed: processed,
				vars:      make(map[string]string),
			})
			return nil
		}

		// 1.4 Assets Estáticos (CSS, JS, JSON, Imagens, etc.)
		minified := minifier.Minify(p, raw)
		gzipped, _ := CompressGzip(minified)
		mimeType := detectContentType(p)

		hash := fmt.Sprintf("%x", sha1.Sum(minified))
		etag := fmt.Sprintf("\"%s\"", hash)
		shortHash := hash
		if len(shortHash) > 8 {
			shortHash = shortHash[:8]
		}
		assetHashes[relPath] = shortHash

		ac.assets[relPath] = &CachedAsset{
			Path:        relPath,
			ContentType: mimeType,
			Content:     minified,
			GzipContent: gzipped,
			ETag:        etag,
			Size:        len(minified),
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Ordenar posts por data decrescente (se data existir)
	if posts, ok := collections["posts"]; ok && len(posts) > 1 {
		sort.SliceStable(posts, func(i, j int) bool {
			return posts[i]["date"] > posts[j]["date"]
		})
		collections["posts"] = posts
	}

	// Pass 2: Converter Markdown em HTML e aplicar layouts
	for _, mdItem := range mdItems {
		doc := mdItem.doc
		layoutFile := doc.Frontmatter["layout"]
		if layoutFile == "" {
			// Procura base.html como layout padrão se existir
			if _, err := os.Stat(filepath.Join(dirPath, "base.html")); err == nil {
				layoutFile = "base.html"
			}
		}

		renderedHTML := doc.ContentHTML
		if layoutFile != "" {
			layoutPath := resolveIncludePath(dirPath, filepath.Dir(mdItem.path), layoutFile)
			if layoutBytes, err := os.ReadFile(layoutPath); err == nil {
				// Processa includes do layout base
				processedLayout := string(processHTML(dirPath, layoutPath, layoutBytes, 0))
				if slotRegex.MatchString(processedLayout) {
					renderedHTML = slotRegex.ReplaceAllString(processedLayout, renderedHTML)
				} else {
					renderedHTML = processedLayout + "\n" + renderedHTML
				}
			}
		}

		// Substitui variáveis do frontmatter ({{ title }}, {{ page.title }}, etc.)
		renderedHTML = processVariables(renderedHTML, doc.Frontmatter)

		htmlRelPath := strings.TrimSuffix(mdItem.relPath, filepath.Ext(mdItem.relPath)) + ".html"
		htmlItems = append(htmlItems, rawHTMLItem{
			path:      mdItem.path,
			relPath:   htmlRelPath,
			processed: []byte(renderedHTML),
			vars:      doc.Frontmatter,
		})
	}

	// Pass 3: Processar Loops de Template, Interpolação de Coleções e Fingerprinting em todos os HTMLs
	for _, item := range htmlItems {
		// 3.1 Processar loops (<!-- for item in collection --> ... <!-- endfor -->)
		content := processTemplateLoops(string(item.processed), collections)

		// 3.2 Substituir variáveis da página se houver
		if len(item.vars) > 0 {
			content = processVariables(content, item.vars)
		}

		// 3.3 Asset Fingerprinting (?v=hash)
		fingerprinted := fingerprintHTML(path.Dir(item.relPath), []byte(content), assetHashes)

		// 3.4 Minificação e Compressão
		minified := minifier.Minify(item.path, fingerprinted)
		gzipped, _ := CompressGzip(minified)

		mimeType := detectContentType(item.path)
		etag := fmt.Sprintf("\"%x\"", sha1.Sum(minified))

		asset := &CachedAsset{
			Path:        item.relPath,
			ContentType: mimeType,
			Content:     minified,
			GzipContent: gzipped,
			ETag:        etag,
			Size:        len(minified),
		}

		ac.assets[item.relPath] = asset

		// Clean URLs
		cleanRoute := strings.TrimSuffix(item.relPath, ".html")
		if item.relPath == "/index.html" {
			ac.assets["/"] = asset
		} else if strings.HasSuffix(item.relPath, "/index.html") {
			dirRoute := strings.TrimSuffix(item.relPath, "index.html")
			ac.assets[dirRoute] = asset
			ac.assets[strings.TrimSuffix(dirRoute, "/")] = asset
		} else {
			ac.assets[cleanRoute] = asset
			ac.assets[cleanRoute+"/"] = asset
		}
	}

	return nil
}

var (
	layoutRegex  = regexp.MustCompile(`<!--\s*layout\s*["']([^"']+)["']\s*-->`)
	includeRegex = regexp.MustCompile(`<!--\s*(?:liaf:)?include\s*["']([^"']+)["']\s*-->`)
	slotRegex    = regexp.MustCompile(`<!--\s*(?:content|slot)\s*-->`)
	forLoopRegex = regexp.MustCompile(`(?s)<!--\s*for\s+(\w+)\s+in\s+(\w+)\s*-->([\s\S]*?)<!--\s*endfor\s*-->`)
	varTagRegex  = regexp.MustCompile(`\{\{\s*([\w\.]+)\s*\}\}`)
)

func processHTML(dirPath, filePath string, raw []byte, depth int) []byte {
	if depth > 10 {
		return raw
	}

	content := string(raw)

	// 1. Process <!-- layout "base.html" -->
	if match := layoutRegex.FindStringSubmatch(content); len(match) > 1 {
		layoutFile := match[1]
		layoutPath := resolveIncludePath(dirPath, filepath.Dir(filePath), layoutFile)
		if layoutBytes, err := os.ReadFile(layoutPath); err == nil {
			bodyContent := layoutRegex.ReplaceAllString(content, "")
			layoutStr := string(processHTML(dirPath, layoutPath, layoutBytes, depth+1))
			if slotRegex.MatchString(layoutStr) {
				content = slotRegex.ReplaceAllString(layoutStr, bodyContent)
			} else {
				content = layoutStr + "\n" + bodyContent
			}
		}
	}

	// 2. Process <!-- include "..." -->
	content = includeRegex.ReplaceAllStringFunc(content, func(m string) string {
		sub := includeRegex.FindStringSubmatch(m)
		if len(sub) < 2 {
			return ""
		}
		incFile := sub[1]
		incPath := resolveIncludePath(dirPath, filepath.Dir(filePath), incFile)
		incBytes, err := os.ReadFile(incPath)
		if err != nil {
			return fmt.Sprintf("<!-- erro incluindo %s: %v -->", incFile, err)
		}
		return string(processHTML(dirPath, incPath, incBytes, depth+1))
	})

	return []byte(content)
}

func processTemplateLoops(content string, collections map[string][]map[string]string) string {
	return forLoopRegex.ReplaceAllStringFunc(content, func(m string) string {
		matches := forLoopRegex.FindStringSubmatch(m)
		if len(matches) < 4 {
			return m
		}
		itemVar := matches[1]
		collectionName := matches[2]
		loopBody := matches[3]

		items, exists := collections[collectionName]
		if !exists || len(items) == 0 {
			return ""
		}

		var sb strings.Builder
		for _, item := range items {
			iteration := loopBody
			// Substitui {{ itemVar.chave }} e {{ chave }}
			iteration = varTagRegex.ReplaceAllStringFunc(iteration, func(tag string) string {
				tagMatch := varTagRegex.FindStringSubmatch(tag)
				if len(tagMatch) < 2 {
					return tag
				}
				prop := tagMatch[1]
				// Se tiver prefixo (ex: post.title)
				if strings.HasPrefix(prop, itemVar+".") {
					prop = strings.TrimPrefix(prop, itemVar+".")
				}
				if val, ok := item[prop]; ok {
					return val
				}
				return tag
			})
			sb.WriteString(iteration)
		}

		return sb.String()
	})
}

func processVariables(content string, vars map[string]string) string {
	return varTagRegex.ReplaceAllStringFunc(content, func(tag string) string {
		tagMatch := varTagRegex.FindStringSubmatch(tag)
		if len(tagMatch) < 2 {
			return tag
		}
		prop := tagMatch[1]
		if strings.HasPrefix(prop, "page.") {
			prop = strings.TrimPrefix(prop, "page.")
		}
		if val, ok := vars[prop]; ok {
			return val
		}
		return tag
	})
}

func resolveIncludePath(baseDir, currentDir, target string) string {
	relPath := filepath.Join(currentDir, target)
	if _, err := os.Stat(relPath); err == nil {
		return relPath
	}
	basePath := filepath.Join(baseDir, target)
	if _, err := os.Stat(basePath); err == nil {
		return basePath
	}
	return relPath
}

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

// LoadFS carrega todos os assets a partir de um sistema de arquivos virtual (ex: embed.FS).
func (ac *AssetCache) LoadFS(fsys fs.FS) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	type rawHTMLItem struct {
		path      string
		relPath   string
		processed []byte
		vars      map[string]string
	}

	type rawMDItem struct {
		path    string
		relPath string
		doc     *markdown.Document
	}

	var htmlItems []rawHTMLItem
	var mdItems []rawMDItem
	assetHashes := make(map[string]string)
	collections := make(map[string][]map[string]string)

	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		raw, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}

		cleanP := strings.TrimPrefix(p, ".")
		cleanP = strings.TrimPrefix(cleanP, "/")
		relPath := "/" + cleanP

		ext := strings.ToLower(path.Ext(relPath))

		// 1.1 Coleções em JSON (ex: data/servicos.json)
		if strings.HasPrefix(relPath, "/data/") && ext == ".json" {
			colName := strings.TrimSuffix(path.Base(relPath), ext)
			var rawList []map[string]any
			if err := json.Unmarshal(raw, &rawList); err == nil {
				var items []map[string]string
				for _, obj := range rawList {
					item := make(map[string]string)
					for k, v := range obj {
						item[k] = fmt.Sprintf("%v", v)
					}
					items = append(items, item)
				}
				collections[colName] = items
			}
		}

		// 1.2 Markdown
		if ext == ".md" || ext == ".markdown" {
			doc := markdown.Parse(string(raw))
			cleanRoute := strings.TrimSuffix(relPath, ext)

			postItem := make(map[string]string)
			for k, v := range doc.Frontmatter {
				postItem[k] = v
			}
			postItem["url"] = cleanRoute
			if _, ok := postItem["title"]; !ok {
				postItem["title"] = path.Base(cleanRoute)
			}
			collections["posts"] = append(collections["posts"], postItem)

			mdItems = append(mdItems, rawMDItem{
				path:    p,
				relPath: relPath,
				doc:     doc,
			})
			return nil
		}

		// 1.3 HTML
		if ext == ".html" || ext == ".htm" {
			processed := processHTMLFS(fsys, p, raw, 0)
			htmlItems = append(htmlItems, rawHTMLItem{
				path:      p,
				relPath:   relPath,
				processed: processed,
				vars:      make(map[string]string),
			})
			return nil
		}

		// 1.4 Assets Estáticos
		minified := minifier.Minify(p, raw)
		gzipped, _ := CompressGzip(minified)
		mimeType := detectContentType(p)

		hash := fmt.Sprintf("%x", sha1.Sum(minified))
		etag := fmt.Sprintf("\"%s\"", hash)
		shortHash := hash
		if len(shortHash) > 8 {
			shortHash = shortHash[:8]
		}
		assetHashes[relPath] = shortHash

		ac.assets[relPath] = &CachedAsset{
			Path:        relPath,
			ContentType: mimeType,
			Content:     minified,
			GzipContent: gzipped,
			ETag:        etag,
			Size:        len(minified),
		}
		return nil
	})
	if err != nil {
		return err
	}

	if posts, ok := collections["posts"]; ok && len(posts) > 1 {
		sort.SliceStable(posts, func(i, j int) bool {
			return posts[i]["date"] > posts[j]["date"]
		})
		collections["posts"] = posts
	}

	for _, mdItem := range mdItems {
		doc := mdItem.doc
		layoutFile := doc.Frontmatter["layout"]
		if layoutFile == "" {
			if _, err := fs.Stat(fsys, "base.html"); err == nil {
				layoutFile = "base.html"
			}
		}

		renderedHTML := doc.ContentHTML
		if layoutFile != "" {
			layoutPath := path.Clean(path.Join(path.Dir(mdItem.path), layoutFile))
			layoutBytes, err := fs.ReadFile(fsys, layoutPath)
			if err != nil {
				layoutBytes, err = fs.ReadFile(fsys, layoutFile)
			}
			if err == nil {
				processedLayout := string(processHTMLFS(fsys, layoutPath, layoutBytes, 0))
				if slotRegex.MatchString(processedLayout) {
					renderedHTML = slotRegex.ReplaceAllString(processedLayout, renderedHTML)
				} else {
					renderedHTML = processedLayout + "\n" + renderedHTML
				}
			}
		}

		renderedHTML = processVariables(renderedHTML, doc.Frontmatter)
		htmlRelPath := strings.TrimSuffix(mdItem.relPath, path.Ext(mdItem.relPath)) + ".html"
		htmlItems = append(htmlItems, rawHTMLItem{
			path:      mdItem.path,
			relPath:   htmlRelPath,
			processed: []byte(renderedHTML),
			vars:      doc.Frontmatter,
		})
	}

	for _, item := range htmlItems {
		content := processTemplateLoops(string(item.processed), collections)
		if len(item.vars) > 0 {
			content = processVariables(content, item.vars)
		}
		fingerprinted := fingerprintHTML(path.Dir(item.relPath), []byte(content), assetHashes)
		minified := minifier.Minify(item.path, fingerprinted)
		gzipped, _ := CompressGzip(minified)

		mimeType := detectContentType(item.path)
		etag := fmt.Sprintf("\"%x\"", sha1.Sum(minified))

		asset := &CachedAsset{
			Path:        item.relPath,
			ContentType: mimeType,
			Content:     minified,
			GzipContent: gzipped,
			ETag:        etag,
			Size:        len(minified),
		}
		ac.assets[item.relPath] = asset

		cleanRoute := strings.TrimSuffix(item.relPath, ".html")
		if item.relPath == "/index.html" {
			ac.assets["/"] = asset
		} else if strings.HasSuffix(item.relPath, "/index.html") {
			dirRoute := strings.TrimSuffix(item.relPath, "index.html")
			ac.assets[dirRoute] = asset
			ac.assets[strings.TrimSuffix(dirRoute, "/")] = asset
		} else {
			ac.assets[cleanRoute] = asset
			ac.assets[cleanRoute+"/"] = asset
		}
	}
	return nil
}

func processHTMLFS(fsys fs.FS, filePath string, raw []byte, depth int) []byte {
	if depth > 10 {
		return raw
	}
	content := string(raw)

	if match := layoutRegex.FindStringSubmatch(content); len(match) > 1 {
		layoutFile := match[1]
		layoutPath := path.Clean(path.Join(path.Dir(filePath), layoutFile))
		layoutBytes, err := fs.ReadFile(fsys, layoutPath)
		if err != nil {
			layoutBytes, err = fs.ReadFile(fsys, layoutFile)
		}
		if err == nil {
			bodyContent := layoutRegex.ReplaceAllString(content, "")
			layoutStr := string(processHTMLFS(fsys, layoutPath, layoutBytes, depth+1))
			if slotRegex.MatchString(layoutStr) {
				content = slotRegex.ReplaceAllString(layoutStr, bodyContent)
			} else {
				content = layoutStr + "\n" + bodyContent
			}
		}
	}

	content = includeRegex.ReplaceAllStringFunc(content, func(m string) string {
		sub := includeRegex.FindStringSubmatch(m)
		if len(sub) < 2 {
			return ""
		}
		incFile := sub[1]
		incPath := path.Clean(path.Join(path.Dir(filePath), incFile))
		incBytes, err := fs.ReadFile(fsys, incPath)
		if err != nil {
			incBytes, err = fs.ReadFile(fsys, incFile)
		}
		if err != nil {
			return fmt.Sprintf("<!-- erro incluindo %s: %v -->", incFile, err)
		}
		return string(processHTMLFS(fsys, incPath, incBytes, depth+1))
	})

	return []byte(content)
}

