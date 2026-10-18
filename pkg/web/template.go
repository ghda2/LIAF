package web

import (
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strings"
)

var (
	layoutRegex  = regexp.MustCompile(`<!--\s*layout\s*["']([^"']+)["']\s*-->`)
	includeRegex = regexp.MustCompile(`<!--\s*(?:liaf:)?include\s*["']([^"']+)["']\s*-->`)
	slotRegex    = regexp.MustCompile(`<!--\s*(?:content|slot)\s*-->`)
	forLoopRegex = regexp.MustCompile(`(?s)<!--\s*for\s+(\w+)\s+in\s+(\w+)\s*-->([\s\S]*?)<!--\s*endfor\s*-->`)
	varTagRegex  = regexp.MustCompile(`\{\{\s*([\w\.]+)\s*\}\}`)
)

func resolveIncludePath(fsys fs.FS, currentDir, target string) ([]byte, string, error) {
	cleanTarget := strings.TrimPrefix(path.Clean(target), "/")
	if cleanTarget == "" || cleanTarget == "." || strings.HasPrefix(cleanTarget, "../") {
		return nil, "", fmt.Errorf("caminho invalido: %s", target)
	}

	// 1. Tenta relativo à pasta atual do arquivo
	relPath := path.Clean(path.Join(currentDir, target))
	relClean := strings.TrimPrefix(relPath, "/")
	if !strings.HasPrefix(relClean, "../") {
		if bytes, err := fs.ReadFile(fsys, relClean); err == nil {
			return bytes, relClean, nil
		}
	}

	// 2. Tenta a partir da raiz do sistema de arquivos
	if bytes, err := fs.ReadFile(fsys, cleanTarget); err == nil {
		return bytes, cleanTarget, nil
	}

	return nil, "", fmt.Errorf("arquivo nao encontrado: %s", target)
}

func processHTML(fsys fs.FS, filePath string, raw []byte, depth int) []byte {
	if depth > 10 {
		return raw
	}

	content := string(raw)
	currentDir := path.Dir(filePath)
	if strings.HasPrefix(currentDir, "/") {
		currentDir = strings.TrimPrefix(currentDir, "/")
	}

	// 1. Process <!-- layout "base.html" -->
	if match := layoutRegex.FindStringSubmatch(content); len(match) > 1 {
		layoutFile := match[1]
		if layoutBytes, layoutPath, err := resolveIncludePath(fsys, currentDir, layoutFile); err == nil {
			bodyContent := layoutRegex.ReplaceAllString(content, "")
			layoutStr := string(processHTML(fsys, layoutPath, layoutBytes, depth+1))
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
		incBytes, incPath, err := resolveIncludePath(fsys, currentDir, incFile)
		if err != nil {
			return fmt.Sprintf("<!-- erro incluindo %s: %v -->", incFile, err)
		}
		return string(processHTML(fsys, incPath, incBytes, depth+1))
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
