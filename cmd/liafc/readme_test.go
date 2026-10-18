package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadmeExamplesCheck passa pelo checker cada bloco ```liaf do README que seja um programa
// completo (começa com `module`). Trechos soltos, sem `module`, ficam de fora: dependem de
// declarações que o README não mostra.
func TestReadmeExamplesCheck(t *testing.T) {
	checkDocExamples(t, "README.md", false)
}

// TestAIGuideExamplesCheck faz o mesmo no guia de agentes, mais estrito: lá todo bloco
// ```liaf tem de ser um programa completo, porque é o que o agente copia.
func TestAIGuideExamplesCheck(t *testing.T) {
	checkDocExamples(t, filepath.Join("docs", "linguagem", "AI_GUIDE.md"), true)
}

func checkDocExamples(t *testing.T, doc string, allComplete bool) {
	data, err := os.ReadFile(filepath.Join("..", "..", doc))
	if err != nil {
		t.Fatal(err)
	}

	blocks := liafBlocks(string(data))
	checked := 0
	for _, b := range blocks {
		if !strings.HasPrefix(strings.TrimSpace(b.code), "module ") {
			if allComplete {
				t.Errorf("%s, bloco da linha %d: não é um programa completo (falta `module`)", doc, b.line)
			}
			continue
		}
		checked++
		t.Run(fmt.Sprintf("linha_%d", b.line), func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "exemplo.liaf")
			if err := os.WriteFile(file, []byte(b.code), 0600); err != nil {
				t.Fatal(err)
			}
			_, diags, _ := parseAndCheck(file)
			for _, d := range diags {
				t.Errorf("%s, bloco da linha %d: %s", doc, b.line, d.String())
			}
		})
	}
	if checked == 0 {
		t.Fatalf("nenhum programa completo encontrado nos blocos ```liaf de %s", doc)
	}
}

type liafBlock struct {
	line int // linha da cerca de abertura no README
	code string
}

func liafBlocks(text string) []liafBlock {
	var blocks []liafBlock
	var cur *liafBlock
	var body []string
	for i, l := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(l)
		switch {
		case cur == nil && trimmed == "```liaf":
			cur = &liafBlock{line: i + 1}
			body = nil
		case cur != nil && trimmed == "```":
			cur.code = strings.Join(body, "\n") + "\n"
			blocks = append(blocks, *cur)
			cur = nil
		case cur != nil:
			body = append(body, l)
		}
	}
	return blocks
}
