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

type markdownRenderer struct {
	buf            bytes.Buffer
	inCodeBlock    bool
	codeBlockLang  string
	codeBlockLines []string
	inList         bool
	inBlockquote   bool
	paraLines      []string
}

func (r *markdownRenderer) flushPara() {
	if len(r.paraLines) > 0 {
		text := strings.Join(r.paraLines, " ")
		text = processInlines(text)
		r.buf.WriteString(fmt.Sprintf("<p>%s</p>\n", text))
		r.paraLines = nil
	}
}

func (r *markdownRenderer) flushList() {
	if r.inList {
		r.buf.WriteString("</ul>\n")
		r.inList = false
	}
}

func (r *markdownRenderer) flushBlockquote() {
	if r.inBlockquote {
		r.buf.WriteString("</blockquote>\n")
		r.inBlockquote = false
	}
}

func (r *markdownRenderer) flushAll() {
	r.flushPara()
	r.flushList()
	r.flushBlockquote()
}

func (r *markdownRenderer) handleCodeFence(line string) {
	r.flushAll()
	if !r.inCodeBlock {
		r.inCodeBlock = true
		r.codeBlockLang = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "```"))
		r.codeBlockLines = nil
	} else {
		r.inCodeBlock = false
		codeContent := html.EscapeString(strings.Join(r.codeBlockLines, "\n"))
		if r.codeBlockLang != "" {
			r.buf.WriteString(fmt.Sprintf("<pre><code class=\"language-%s\">%s</code></pre>\n", r.codeBlockLang, codeContent))
		} else {
			r.buf.WriteString(fmt.Sprintf("<pre><code>%s</code></pre>\n", codeContent))
		}
	}
}

func (r *markdownRenderer) processLine(line string) {
	if strings.HasPrefix(strings.TrimSpace(line), "```") {
		r.handleCodeFence(line)
		return
	}
	if r.inCodeBlock {
		r.codeBlockLines = append(r.codeBlockLines, line)
		return
	}

	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		r.flushAll()
		return
	}

	if hMatch := headerRegex.FindStringSubmatch(trimmed); len(hMatch) == 3 {
		r.flushAll()
		level := len(hMatch[1])
		title := processInlines(hMatch[2])
		r.buf.WriteString(fmt.Sprintf("<h%d>%s</h%d>\n", level, title, level))
		return
	}

	if trimmed == "---" || trimmed == "***" || trimmed == "___" {
		r.flushAll()
		r.buf.WriteString("<hr />\n")
		return
	}

	if strings.HasPrefix(trimmed, ">") {
		r.flushPara()
		r.flushList()
		quoteContent := strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))
		if !r.inBlockquote {
			r.buf.WriteString("<blockquote>\n")
			r.inBlockquote = true
		}
		r.buf.WriteString(fmt.Sprintf("<p>%s</p>\n", processInlines(quoteContent)))
		return
	}
	r.flushBlockquote()

	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		r.flushPara()
		itemContent := strings.TrimSpace(trimmed[2:])
		if !r.inList {
			r.buf.WriteString("<ul>\n")
			r.inList = true
		}
		r.buf.WriteString(fmt.Sprintf("<li>%s</li>\n", processInlines(itemContent)))
		return
	}
	r.flushList()

	r.paraLines = append(r.paraLines, trimmed)
}

// RenderHTML converte Markdown padrão para HTML semanticamente limpo.
func RenderHTML(md string) string {
	r := &markdownRenderer{}
	for _, rawLine := range strings.Split(md, "\n") {
		r.processLine(strings.TrimRight(rawLine, "\r"))
	}
	r.flushAll()
	return strings.TrimSpace(r.buf.String())
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
