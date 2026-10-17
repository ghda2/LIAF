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
	data, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}

	blocks := liafBlocks(string(data))
	checked := 0
	for _, b := range blocks {
		if !strings.HasPrefix(strings.TrimSpace(b.code), "module ") {
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
				t.Errorf("README.md, bloco da linha %d: %s", b.line, d.String())
			}
		})
	}
	if checked == 0 {
		t.Fatal("nenhum programa completo encontrado nos blocos ```liaf do README")
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
