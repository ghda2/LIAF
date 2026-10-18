package lsp

import (
	"fmt"
	"strings"

	"liaf/pkg/ast"
	"liaf/pkg/builtins"
	"liaf/pkg/lexer"
	"liaf/pkg/parser"
)

func computeHover(source string, pos Position) *Hover {
	word := getWordAtPosition(source, pos)
	if word == "" {
		return nil
	}

	// 1. Verifica se é um builtin da linguagem
	b := builtins.Lookup(word)
	if b != nil {
		paramsStr := strings.Join(b.Params, ", ")
		retStr := b.Return
		if retStr == "" {
			retStr = "void"
		}
		var effStr string
		if len(b.Effects) > 0 {
			effStr = fmt.Sprintf("\n\n**Efeitos obrigatórios:** `%s`", strings.Join(b.Effects, "`, `"))
		}

		md := fmt.Sprintf("```liaf\n(%s %s) -> %s\n```\n**Builtin LIAF** (`%s`)%s",
			b.Name, paramsStr, retStr, b.Category, effStr)

		return &Hover{
			Contents: MarkupContent{
				Kind:  "markdown",
				Value: md,
			},
		}
	}

	// 2. Verifica se é uma palavra-chave
	for _, kw := range liafKeywords {
		if kw == word {
			return &Hover{
				Contents: MarkupContent{
					Kind:  "markdown",
					Value: fmt.Sprintf("**Palavra-chave LIAF**: `%s`", kw),
				},
			}
		}
	}

	// 3. Procura declaração correspondente no próprio documento
	lex := lexer.New(source)
	p := parser.New(lex, "temp.liaf")
	mod := p.ParseModule()
	if mod != nil {
		for _, decl := range mod.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Name == word {
					if d.IsMiddleware {
						md := fmt.Sprintf("```liaf\nmiddleware %s(req)\n```", d.Name)
						return &Hover{
							Contents: MarkupContent{
								Kind:  "markdown",
								Value: md,
							},
						}
					}
					var params []string
					for _, param := range d.Params {
						params = append(params, fmt.Sprintf("%s: %s", param.Name, param.Type.String()))
					}
					ret := "void"
					if d.ReturnType != nil {
						ret = d.ReturnType.String()
					}
					var eff string
					if len(d.Effects) > 0 {
						eff = fmt.Sprintf("\n\n**Efeitos:** `%s`", strings.Join(d.Effects, "`, `"))
					}
					md := fmt.Sprintf("```liaf\nfn %s(%s) %s\n```%s", d.Name, strings.Join(params, ", "), ret, eff)
					return &Hover{
						Contents: MarkupContent{
							Kind:  "markdown",
							Value: md,
						},
					}
				}
			case *ast.StructDecl:
				if d.Name == word {
					var fields []string
					for _, f := range d.Fields {
						fields = append(fields, fmt.Sprintf("  %s %s", f.Name, f.Type.String()))
					}
					md := fmt.Sprintf("```liaf\nstruct %s\n%s\nend\n```", d.Name, strings.Join(fields, "\n"))
					return &Hover{
						Contents: MarkupContent{
							Kind:  "markdown",
							Value: md,
						},
					}
				}
			case *ast.EnumDecl:
				if d.Name == word {
					var variants []string
					for _, v := range d.Variants {
						variants = append(variants, fmt.Sprintf("  %s", v.Name))
					}
					md := fmt.Sprintf("```liaf\nenum %s\n%s\nend\n```", d.Name, strings.Join(variants, "\n"))
					return &Hover{
						Contents: MarkupContent{
							Kind:  "markdown",
							Value: md,
						},
					}
				}
			}
		}
	}

	return nil
}

func getWordAtPosition(source string, pos Position) string {
	lines := strings.Split(source, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return ""
	}
	line := strings.TrimRight(lines[pos.Line], "\r")
	if pos.Character < 0 || pos.Character > len(line) {
		return ""
	}

	start := pos.Character
	if start >= len(line) && start > 0 {
		start = len(line) - 1
	}

	// Se estiver em cima de espaço ou separador, tenta olhar 1 caractere antes
	if start < len(line) && !isWordChar(line[start]) && start > 0 && isWordChar(line[start-1]) {
		start--
	}
	if start >= len(line) || !isWordChar(line[start]) {
		return ""
	}

	// Anda para a esquerda
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}

	// Anda para a direita
	end := pos.Character
	for end < len(line) && isWordChar(line[end]) {
		end++
	}

	return line[start:end]
}

func isWordChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' || b == '-'
}
