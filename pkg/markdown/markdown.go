package markdown

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"strings"
)

// Document representa um arquivo Markdown processado com Frontmatter e HTML.
type Document struct {
	Frontmatter map[string]string
	ContentHTML string
	RawBody     string
}

var (
	frontmatterRegex = regexp.MustCompile(`(?s)^---\r?\n(.*?)\r?\n---\r?\n(.*)$`)
	headerRegex      = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	boldRegex        = regexp.MustCompile(`\*\*(.+?)\*\*`)
	italicRegex      = regexp.MustCompile(`\*([^*]+?)\*`)
	inlineCodeRegex  = regexp.MustCompile("`([^`]+)`")
	linkRegex        = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	imageRegex       = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
)

// Parse processa uma string Markdown com frontmatter opcional e converte em HTML.
func Parse(input string) *Document {
	doc := &Document{
		Frontmatter: make(map[string]string),
	}

	body := input
	if matches := frontmatterRegex.FindStringSubmatch(input); len(matches) == 3 {
		yamlPart := matches[1]
		body = matches[2]
		parseFrontmatter(yamlPart, doc.Frontmatter)
	}

	doc.RawBody = body
	doc.ContentHTML = RenderHTML(body)
	return doc
}

func parseFrontmatter(raw string, target map[string]string) {
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			// Remove aspas simples ou duplas
			val = strings.Trim(val, `"'`)
			target[key] = val
		}
	}
}

// RenderHTML converte Markdown padrão para HTML semanticamente limpo.
func RenderHTML(md string) string {
	var buf bytes.Buffer
	lines := strings.Split(md, "\n")

	inCodeBlock := false
	codeBlockLang := ""
	var codeBlockLines []string

	inList := false
	inBlockquote := false
	var paraLines []string

	flushPara := func() {
		if len(paraLines) > 0 {
			text := strings.Join(paraLines, " ")
			text = processInlines(text)
			buf.WriteString(fmt.Sprintf("<p>%s</p>\n", text))
			paraLines = nil
		}
	}

	flushList := func() {
		if inList {
			buf.WriteString("</ul>\n")
			inList = false
		}
	}

	flushBlockquote := func() {
		if inBlockquote {
			buf.WriteString("</blockquote>\n")
			inBlockquote = false
		}
	}

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")

		// Blocos de código ```
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			flushPara()
			flushList()
			flushBlockquote()

			if !inCodeBlock {
				inCodeBlock = true
				codeBlockLang = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "```"))
				codeBlockLines = nil
			} else {
				inCodeBlock = false
				codeContent := html.EscapeString(strings.Join(codeBlockLines, "\n"))
				if codeBlockLang != "" {
					buf.WriteString(fmt.Sprintf("<pre><code class=\"language-%s\">%s</code></pre>\n", codeBlockLang, codeContent))
				} else {
					buf.WriteString(fmt.Sprintf("<pre><code>%s</code></pre>\n", codeContent))
				}
			}
			continue
		}

		if inCodeBlock {
			codeBlockLines = append(codeBlockLines, line)
			continue
		}

		trimmed := strings.TrimSpace(line)

		// Linha em branco
		if trimmed == "" {
			flushPara()
			flushList()
			flushBlockquote()
			continue
		}

		// Headings (# Título)
		if hMatch := headerRegex.FindStringSubmatch(trimmed); len(hMatch) == 3 {
			flushPara()
			flushList()
			flushBlockquote()
			level := len(hMatch[1])
			title := processInlines(hMatch[2])
			buf.WriteString(fmt.Sprintf("<h%d>%s</h%d>\n", level, title, level))
			continue
		}

		// Horizontal Rule (--- ou ***)
		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			flushPara()
			flushList()
			flushBlockquote()
			buf.WriteString("<hr />\n")
			continue
		}

		// Blockquotes (> citação)
		if strings.HasPrefix(trimmed, ">") {
			flushPara()
			flushList()
			quoteContent := strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))
			if !inBlockquote {
				buf.WriteString("<blockquote>\n")
				inBlockquote = true
			}
			buf.WriteString(fmt.Sprintf("<p>%s</p>\n", processInlines(quoteContent)))
			continue
		} else {
			flushBlockquote()
		}

		// Unordered List (- item ou * item)
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			flushPara()
			itemContent := strings.TrimSpace(trimmed[2:])
			if !inList {
				buf.WriteString("<ul>\n")
				inList = true
			}
			buf.WriteString(fmt.Sprintf("<li>%s</li>\n", processInlines(itemContent)))
			continue
		} else {
			flushList()
		}

		// Parágrafos normais
		paraLines = append(paraLines, trimmed)
	}

	flushPara()
	flushList()
	flushBlockquote()

	return strings.TrimSpace(buf.String())
}

func processInlines(text string) string {
	// Imagens ![alt](url)
	text = imageRegex.ReplaceAllString(text, `<img src="$2" alt="$1" />`)
	// Links [texto](url)
	text = linkRegex.ReplaceAllString(text, `<a href="$2">$1</a>`)
	// Bold **texto**
	text = boldRegex.ReplaceAllString(text, `<strong>$1</strong>`)
	// Italic *texto*
	text = italicRegex.ReplaceAllString(text, `<em>$1</em>`)
	// Inline Code `codigo`
	text = inlineCodeRegex.ReplaceAllStringFunc(text, func(m string) string {
		sub := inlineCodeRegex.FindStringSubmatch(m)
		if len(sub) > 1 {
			return fmt.Sprintf("<code>%s</code>", html.EscapeString(sub[1]))
		}
		return m
	})
	return text
}
