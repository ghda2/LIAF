package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLICommandPostCheckValid(t *testing.T) {
	tmpDir := t.TempDir()
	postFile := filepath.Join(tmpDir, "artigo.md")
	content := `---
title: "Meu Post de Teste"
slug: "meu-post-de-teste"
date: "2026-10-17"
tags: [ia, liaf]
summary: "Resumo do post"
---

# Titulo

Corpo do artigo em markdown.
`
	if err := os.WriteFile(postFile, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "run", "./cmd/liafc", "post", postFile, "--check", "--json")
	cmd.Dir = root
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err = cmd.Run()
	if err != nil {
		t.Fatalf("erro ao executar liafc post: %v\nstdout: %s", err, stdout.String())
	}

	var resp PostJSONResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		t.Fatalf("resposta json invalida: %v\n%s", err, stdout.String())
	}

	if !resp.Valid || resp.Post == nil || resp.Post.Title != "Meu Post de Teste" {
		t.Fatalf("esperava post valido, obteve: %+v", resp)
	}
}

func TestCLICommandPostCheckInvalidAutoHealing(t *testing.T) {
	tmpDir := t.TempDir()
	postFile := filepath.Join(tmpDir, "sem_titulo.md")
	content := `---
slug: "artigo-sem-titulo"
---

Corpo sem titulo
`
	if err := os.WriteFile(postFile, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "run", "./cmd/liafc", "post", postFile, "--check", "--json")
	cmd.Dir = root
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	_ = cmd.Run()

	var resp PostJSONResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		t.Fatalf("resposta json invalida: %v\n%s", err, stdout.String())
	}

	if resp.Valid {
		t.Fatalf("esperava erro de validacao, mas marcou como valido")
	}
	if len(resp.Errors) == 0 {
		t.Fatalf("esperava erros de validacao")
	}
	if resp.Errors[0].Code != "E_CONTENT_MISSING_TITLE" {
		t.Errorf("codigo esperado E_CONTENT_MISSING_TITLE, obteve %s", resp.Errors[0].Code)
	}
	if !strings.Contains(resp.Errors[0].FixHint, "title") {
		t.Errorf("dica de auto-cura esperada no FixHint: %s", resp.Errors[0].FixHint)
	}
}
