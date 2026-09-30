package web

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"liaf/pkg/markdown"
	"liaf/pkg/minifier"
)

func loadPipeline(ac *AssetCache, fsys fs.FS, diskBaseDir string) error {
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
		if err != nil {
			return err
		}

		// Pula diretórios ocultos e raiz
		if d.IsDir() {
			if p != "." && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}

		// Ignora arquivos ocultos e symlinks
		if strings.HasPrefix(d.Name(), ".") || d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		cleanP := strings.TrimPrefix(p, ".")
		cleanP = strings.TrimPrefix(cleanP, "/")
		relPath := "/" + cleanP
		ext := strings.ToLower(path.Ext(relPath))

		// Tier 2 (Disco / Streaming): Mídias e arquivos grandes não ocupam memória RAM
		if diskBaseDir != "" {
			if info, err := d.Info(); err == nil {
				isHeavyMedia := heavyMediaExts[ext]
				isTooBig := info.Size() > ac.MaxRAMAssetSize &&
					ext != ".html" && ext != ".htm" && ext != ".css" && ext != ".js" &&
					ext != ".json" && ext != ".md" && ext != ".markdown" && ext != ".svg"

				if isHeavyMedia || isTooBig {
					mimeType := detectContentType(p)
					etag := fmt.Sprintf("\"disk-%x-%x\"", info.ModTime().UnixNano(), info.Size())
					modHex := fmt.Sprintf("%x", info.ModTime().Unix())
					if len(modHex) > 8 {
						modHex = modHex[:8]
					}
					assetHashes[relPath] = modHex

					ac.assets[relPath] = &CachedAsset{
						Path:        relPath,
						ContentType: mimeType,
						ETag:        etag,
						Size:        int(info.Size()),
						DiskPath:    filepath.Join(diskBaseDir, filepath.FromSlash(p)),
						IsDisk:      true,
						ModTime:     info.ModTime(),
					}
					return nil
				}
			}
		}

		raw, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}

		// 1.1 Coleções em JSON (ex: public/data/servicos.json)
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

		// 1.2 Arquivos Markdown (.md, .markdown)
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

		// 1.3 Arquivos HTML
		if ext == ".html" || ext == ".htm" {
			processed := processHTML(fsys, p, raw, 0)
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
			if _, err := fs.Stat(fsys, "base.html"); err == nil {
				layoutFile = "base.html"
			}
		}

		renderedHTML := doc.ContentHTML
		if layoutFile != "" {
			currentDir := path.Dir(mdItem.path)
			if layoutBytes, layoutPath, err := resolveIncludePath(fsys, currentDir, layoutFile); err == nil {
				processedLayout := string(processHTML(fsys, layoutPath, layoutBytes, 0))
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

	// Pass 3: Processar Loops de Template, Interpolação de Coleções e Fingerprinting em todos os HTMLs
	for _, item := range htmlItems {
		content := processTemplateLoops(string(item.processed), collections)
		if len(item.vars) > 0 {
			content = processVariables(content, item.vars)
		}
		fingerprinted := fingerprintHTML(path.Dir(item.relPath), []byte(content), assetHashes)
		minified := minifier.Minify(item.relPath, fingerprinted)
		gzipped, _ := CompressGzip(minified)

		mimeType := detectContentType(item.relPath)
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

	// Pass 4: Auto-geração de SEO (sitemap.xml e robots.txt) caso não fornecidos manualmente
	if len(htmlItems) > 0 {
		if _, hasSitemap := ac.assets["/sitemap.xml"]; !hasSitemap {
			var sb strings.Builder
			sb.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
			sb.WriteString("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")

			var routes []string
			seen := make(map[string]bool)
			for _, item := range htmlItems {
				route := strings.TrimSuffix(item.relPath, ".html")
				if item.relPath == "/index.html" {
					route = "/"
				} else if strings.HasSuffix(item.relPath, "/index.html") {
					route = strings.TrimSuffix(item.relPath, "index.html")
				}
				if !seen[route] {
					seen[route] = true
					routes = append(routes, route)
				}
			}
			sort.Strings(routes)

			for _, r := range routes {
				priority := "0.8"
				if r == "/" {
					priority = "1.0"
				}
				sb.WriteString(fmt.Sprintf("  <url>\n    <loc>%s</loc>\n    <changefreq>weekly</changefreq>\n    <priority>%s</priority>\n  </url>\n", r, priority))
			}
			sb.WriteString("</urlset>\n")

			sitemapBytes := []byte(sb.String())
			sitemapGzip, _ := CompressGzip(sitemapBytes)
			etag := fmt.Sprintf("\"%x\"", sha1.Sum(sitemapBytes))
			ac.assets["/sitemap.xml"] = &CachedAsset{
				Path:        "/sitemap.xml",
				ContentType: "application/xml; charset=utf-8",
				Content:     sitemapBytes,
				GzipContent: sitemapGzip,
				ETag:        etag,
				Size:        len(sitemapBytes),
			}
		}

		if _, hasRobots := ac.assets["/robots.txt"]; !hasRobots {
			robotsContent := []byte("User-agent: *\nAllow: /\nSitemap: /sitemap.xml\n")
			robotsGzip, _ := CompressGzip(robotsContent)
			etag := fmt.Sprintf("\"%x\"", sha1.Sum(robotsContent))
			ac.assets["/robots.txt"] = &CachedAsset{
				Path:        "/robots.txt",
				ContentType: "text/plain; charset=utf-8",
				Content:     robotsContent,
				GzipContent: robotsGzip,
				ETag:        etag,
				Size:        len(robotsContent),
			}
		}
	}

	return nil
}
