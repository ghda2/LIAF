package content

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePost(t *testing.T) {
	raw := `---
title: "LIAF v1.0 e IA First"
date: 2026-10-17
author: "Gabriel Santos"
tags: [ia, compilers, web]
summary: "Como a LIAF atinge deploy instantâneo."
image: "cover.png"
---

# Introdução

A LIAF é uma linguagem moderna.

Veja esta imagem: ![Grafico](chart.png)
`
	post, err := Parse(raw)
	if err != nil {
		t.Fatalf("erro ao fazer parse: %v", err)
	}

	if post.Title != "LIAF v1.0 e IA First" {
		t.Errorf("titulo incorreto: %s", post.Title)
	}
	if post.Slug != "liaf-v1-0-e-ia-first" {
		t.Errorf("slug gerado incorreto: %s", post.Slug)
	}
	if len(post.Tags) != 3 || post.Tags[0] != "ia" || post.Tags[1] != "compilers" || post.Tags[2] != "web" {
		t.Errorf("tags incorretas: %+v", post.Tags)
	}
	if post.Image != "cover.png" {
		t.Errorf("imagem incorreta: %s", post.Image)
	}
	if !strings.Contains(post.ContentHTML, "<h1>Introdução</h1>") {
		t.Errorf("HTML nao gerou h1: %s", post.ContentHTML)
	}
	if post.ReadingTimeMin < 1 {
		t.Errorf("tempo de leitura invalido: %d", post.ReadingTimeMin)
	}
}

func TestValidatePostMissingTitle(t *testing.T) {
	raw := `---
slug: meu-artigo
---
Corpo sem titulo
`
	post, err := Parse(raw)
	if err != nil {
		t.Fatalf("erro no parse: %v", err)
	}

	diags := Validate(post, "")
	if len(diags) == 0 || diags[0].Code != "E_CONTENT_MISSING_TITLE" {
		t.Fatalf("esperava E_CONTENT_MISSING_TITLE, obteve: %+v", diags)
	}
	if diags[0].FixHint == "" {
		t.Errorf("dica de correcao deve estar preenchida para IAs")
	}
}

func TestValidatePostImages(t *testing.T) {
	tmpDir := t.TempDir()
	// Cria uma imagem valida
	if err := os.WriteFile(filepath.Join(tmpDir, "existente.png"), []byte("png data"), 0600); err != nil {
		t.Fatal(err)
	}

	raw := `---
title: "Post com Imagens"
image: existente.png
---
Aqui esta uma imagem quebrada: ![Falha](nao_existe.png)
`
	post, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	diags := Validate(post, tmpDir)
	if len(diags) != 1 || diags[0].Code != "E_CONTENT_EMBED_IMAGE_NOT_FOUND" {
		t.Fatalf("esperava erro na imagem inexistente, obteve: %+v", diags)
	}

	images := ExtractLocalImages(post)
	if len(images) != 2 {
		t.Fatalf("esperava 2 imagens extraidas, obteve: %+v", images)
	}
}
