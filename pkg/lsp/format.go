package lsp

import (
	"strings"

	"liaf/pkg/ast"
	"liaf/pkg/lexer"
	"liaf/pkg/parser"
)

func computeFormatting(source string) []TextEdit {
	formatted, ok := formatDocument(source)
	if !ok || formatted == source {
		return nil
	}

	lines := strings.Split(source, "\n")
	lastLine := len(lines) - 1
	if lastLine < 0 {
		lastLine = 0
	}
	lastChar := 0
	if len(lines) > 0 {
		lastChar = len(lines[lastLine])
	}

	return []TextEdit{
		{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: lastLine, Character: lastChar},
			},
			NewText: formatted,
		},
	}
}

func formatDocument(source string) (string, bool) {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return source, true
	}

	// Se for S-expression tradicional:
	if strings.HasPrefix(trimmed, "(module") || strings.HasPrefix(trimmed, "(") {
		p := parser.New(lexer.New(source), "format.liaf")
		mod := p.ParseModule()
		if mod == nil || len(p.Diagnostics) > 0 {
			return source, false
		}
		return ast.Format(mod), true
	}

	// Se for sintaxe linear (v0.6):
	// Realiza formatação e indentação consistente de blocos lineares
	return formatLinear(source), true
}

func formatLinear(source string) string {
	lines := strings.Split(source, "\n")
	var formatted []string
	indent := 0

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			formatted = append(formatted, "")
			continue
		}

		// Palavras que fecham ou reduzem indentação antes da linha
		if strings.HasPrefix(line, "end") || strings.HasPrefix(line, "else") || strings.HasPrefix(line, "on-err") {
			if indent > 0 {
				indent--
			}
		}

		// Aplica a indentação atual (2 espaços)
		prefix := strings.Repeat("  ", indent)
		formatted = append(formatted, prefix+line)

		// Palavras que abrem bloco para a próxima linha
		if opensBlock(line) {
			indent++
		}
	}

	res := strings.Join(formatted, "\n")
	if !strings.HasSuffix(res, "\n") {
		res += "\n"
	}
	return res
}

func opensBlock(line string) bool {
	// Se a linha já terminar com end na mesma linha, não abre bloco
	if strings.HasSuffix(line, "end") {
		return false
	}

	blockStarters := []string{
		"fn ", "pub fn ", "struct ", "pub struct ", "enum ", "pub enum ",
		"if ", "else", "while ", "for ", "route ", "ws ", "middleware ",
		"with ", "test ", "try", "on-err", "match ",
	}

	for _, starter := range blockStarters {
		if strings.HasPrefix(line, starter) || line == strings.TrimSpace(starter) {
			return true
		}
	}
	return false
}
