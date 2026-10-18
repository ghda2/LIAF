package lsp

import (
	"strings"

	"liaf/pkg/checker"
	"liaf/pkg/lexer"
	"liaf/pkg/parser"
)

// computeDiagnostics analisa o código fonte com lexer, parser e checker,
// convertendo os diagnósticos para o formato esperado pelo LSP (0-indexed).
func computeDiagnostics(source, uri string) []Diagnostic {
	var lspDiags []Diagnostic

	lex := lexer.New(source)
	p := parser.New(lex, uri)
	mod := p.ParseModule()

	// Coleta erros de sintaxe / parse
	for _, d := range p.Diagnostics {
		lspDiags = append(lspDiags, formatDiagnostic(d.Code, d.Message, d.Line, d.Col, source))
	}

	// Se o módulo pôde ser montado, executa o type checker
	if mod != nil {
		_, checkerErrors := checker.Check(mod, uri)
		for _, d := range checkerErrors {
			lspDiags = append(lspDiags, formatDiagnostic(d.Code, d.Message, d.Line, d.Col, source))
		}
	}

	return lspDiags
}

func formatDiagnostic(code, message string, line1, col1 int, source string) Diagnostic {
	// LIAF usa 1-indexed para linha e coluna.
	// LSP usa 0-indexed para linha e coluna.
	line := line1 - 1
	if line < 0 {
		line = 0
	}
	col := col1 - 1
	if col < 0 {
		col = 0
	}

	endCol := col + 1
	lines := strings.Split(source, "\n")
	if line < len(lines) {
		lineStr := strings.TrimRight(lines[line], "\r")
		if col < len(lineStr) {
			end := col
			for end < len(lineStr) && isIdentChar(lineStr[end]) {
				end++
			}
			if end > col {
				endCol = end
			}
		} else {
			endCol = len(lineStr)
		}
	}

	return Diagnostic{
		Range: Range{
			Start: Position{Line: line, Character: col},
			End:   Position{Line: line, Character: endCol},
		},
		Severity: SeverityError,
		Code:     code,
		Source:   "liafc",
		Message:  message,
	}
}

func isIdentChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' || b == '-'
}
