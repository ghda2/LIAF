package markdown

import (
	"strings"
	"testing"
)

func TestParseFrontmatterAndContent(t *testing.T) {
	input := `---
title: "Título de Teste"
date: "2026-09-15"
layout: "base.html"
---
# Meu Artigo Incrível

Este é um parágrafo com **negrito**, *itálico* e ` + "`código`" + `.

- Item 1
- Item 2 com [link](https://liaf.dev)

> Uma citação relevante

` + "```go\nfunc main() {}\n```"

	doc := Parse(input)

	if doc.Frontmatter["title"] != "Título de Teste" {
		t.Fatalf("esperado title 'Título de Teste', obtido '%s'", doc.Frontmatter["title"])
	}
	if doc.Frontmatter["date"] != "2026-09-15" {
		t.Fatalf("esperado date '2026-09-15', obtido '%s'", doc.Frontmatter["date"])
	}
	if doc.Frontmatter["layout"] != "base.html" {
		t.Fatalf("esperado layout 'base.html', obtido '%s'", doc.Frontmatter["layout"])
	}

	if !strings.Contains(doc.ContentHTML, "<h1>Meu Artigo Incrível</h1>") {
		t.Errorf("HTML não contém h1: %s", doc.ContentHTML)
	}
	if !strings.Contains(doc.ContentHTML, "<strong>negrito</strong>") {
		t.Errorf("HTML não contém negrito: %s", doc.ContentHTML)
	}
	if !strings.Contains(doc.ContentHTML, "<em>itálico</em>") {
		t.Errorf("HTML não contém itálico: %s", doc.ContentHTML)
	}
	if !strings.Contains(doc.ContentHTML, "<code>código</code>") {
		t.Errorf("HTML não contém code: %s", doc.ContentHTML)
	}
	if !strings.Contains(doc.ContentHTML, "<li>Item 1</li>") {
		t.Errorf("HTML não contém li: %s", doc.ContentHTML)
	}
	if !strings.Contains(doc.ContentHTML, "<a href=\"https://liaf.dev\">link</a>") {
		t.Errorf("HTML não contém link: %s", doc.ContentHTML)
	}
	if !strings.Contains(doc.ContentHTML, "<blockquote>") {
		t.Errorf("HTML não contém blockquote: %s", doc.ContentHTML)
	}
	if !strings.Contains(doc.ContentHTML, "<pre><code class=\"language-go\">func main() {}</code></pre>") {
		t.Errorf("HTML não contém code block com classe: %s", doc.ContentHTML)
	}
}
