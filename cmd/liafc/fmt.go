package main

import (
	"fmt"
	"os"

	"liaf/pkg/ast"
	"liaf/pkg/lexer"
	"liaf/pkg/parser"
)

// runFmt imprime a forma canonica de um modulo LIAF. O formato canonico da v0.3
// (chamadas diretas, try/on-err, rotas declarativas) e produzido pelo printer em
// pkg/ast, entao formatar e reimprimir a AST, nao reescrever texto.
func runFmt(args []string) {
	write := false
	list := false
	dropComments := false
	var files []string

	for _, a := range args {
		switch a {
		case "-w", "--write":
			write = true
		case "-l", "--list":
			list = true
		case "--drop-comments":
			dropComments = true
		default:
			files = append(files, a)
		}
	}

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "Uso: liafc fmt <arquivo.liaf> [mais arquivos...] [-w] [-l]")
		os.Exit(1)
	}
	if write && list {
		fmt.Fprintln(os.Stderr, "liafc fmt: -w e -l sao mutuamente exclusivos")
		os.Exit(1)
	}

	changed := false
	failed := false

	for _, filePath := range files {
		original, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao ler arquivo %s: %v\n", filePath, err)
			failed = true
			continue
		}

		p := parser.New(lexer.New(string(original)), filePath)
		mod := p.ParseModule()
		if mod == nil || len(p.Diagnostics) > 0 {
			fmt.Fprintf(os.Stderr, "%s: nao formatado, o arquivo nao analisa\n", filePath)
			for _, d := range p.Diagnostics {
				fmt.Fprintf(os.Stderr, "  [%s] %s:%d:%d - %s\n", d.Code, d.File, d.Line, d.Col, d.Message)
			}
			failed = true
			continue
		}

		formatted := ast.Format(mod)
		if formatted == string(original) {
			continue
		}
		changed = true

		switch {
		case list:
			fmt.Println(filePath)
		case write:
			if hasComments(string(original)) && !dropComments {
				fmt.Fprintf(os.Stderr, "%s: nao gravado, o arquivo tem comentarios e o formatador ainda nao os preserva.\n", filePath)
				fmt.Fprintln(os.Stderr, "  Use 'liafc fmt <arquivo>' para ver a saida, ou '-w --drop-comments' para gravar mesmo assim.")
				failed = true
				continue
			}
			if err := os.WriteFile(filePath, []byte(formatted), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao gravar %s: %v\n", filePath, err)
				failed = true
				continue
			}
			fmt.Printf("formatado: %s\n", filePath)
		default:
			fmt.Print(formatted)
		}
	}

	if failed {
		os.Exit(1)
	}
	// -l existe para CI: sair diferente de zero quando algo esta fora do formato.
	if list && changed {
		os.Exit(1)
	}
}

// hasComments diz se o fonte tem comentarios ';'. O printer reimprime a AST e a
// AST nao guarda comentarios, entao formatar um arquivo comentado apaga todos.
// Enquanto o parser nao anexar comentarios aos nos, -w se recusa a fazer isso em
// silencio.
func hasComments(src string) bool {
	inString := false
	for i := 0; i < len(src); i++ {
		switch src[i] {
		case '\\':
			if inString {
				i++
			}
		case '"':
			inString = !inString
		case ';':
			if !inString {
				return true
			}
		}
	}
	return false
}
