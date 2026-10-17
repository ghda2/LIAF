package lsp

import (
	"fmt"
	"strings"

	"liaf/pkg/ast"
	"liaf/pkg/builtins"
	"liaf/pkg/lexer"
	"liaf/pkg/parser"
)

var liafKeywords = []string{
	"fn", "pub", "let", "mut", "if", "else", "while", "for", "in",
	"return", "try", "on-err", "match", "route", "ws", "middleware", "with",
	"struct", "enum", "test", "module", "import", "effects", "params",
	"true", "false",
}

func computeCompletions(source string, pos Position) CompletionList {
	var items []CompletionItem

	// 1. Palavras-chave da linguagem
	for _, kw := range liafKeywords {
		items = append(items, CompletionItem{
			Label:  kw,
			Kind:   KindKeyword,
			Detail: "palavra-chave LIAF",
		})
	}

	// 2. Builtins da biblioteca padrão
	for _, b := range builtins.All() {
		paramsStr := strings.Join(b.Params, ", ")
		retStr := b.Return
		if retStr == "" {
			retStr = "void"
		}
		sig := fmt.Sprintf("(%s) -> %s", paramsStr, retStr)

		var effStr string
		if len(b.Effects) > 0 {
			effStr = fmt.Sprintf("\n\n**Efeitos:** `%s`", strings.Join(b.Effects, "`, `"))
		}

		doc := fmt.Sprintf("**Builtin LIAF** (`%s`)\n\nAssinatura: `%s %s`%s",
			b.Category, b.Name, sig, effStr)

		items = append(items, CompletionItem{
			Label:  b.Name,
			Kind:   KindFunction,
			Detail: fmt.Sprintf("%s %s", b.Name, sig),
			Documentation: &MarkupContent{
				Kind:  "markdown",
				Value: doc,
			},
		})
	}

	// 3. Símbolos locais declarados no documento
	lex := lexer.New(source)
	p := parser.New(lex, "temp.liaf")
	mod := p.ParseModule()
	if mod != nil {
		for _, decl := range mod.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.IsMiddleware {
					items = append(items, CompletionItem{
						Label:  d.Name,
						Kind:   KindMethod,
						Detail: fmt.Sprintf("middleware %s(req)", d.Name),
					})
				} else {
					paramNames := make([]string, len(d.Params))
					for i, param := range d.Params {
						paramNames[i] = fmt.Sprintf("%s %s", param.Name, param.Type.String())
					}
					ret := "void"
					if d.ReturnType != nil {
						ret = d.ReturnType.String()
					}
					items = append(items, CompletionItem{
						Label:  d.Name,
						Kind:   KindFunction,
						Detail: fmt.Sprintf("fn %s(%s) %s", d.Name, strings.Join(paramNames, ", "), ret),
					})
				}
			case *ast.StructDecl:
				items = append(items, CompletionItem{
					Label:  d.Name,
					Kind:   KindStruct,
					Detail: fmt.Sprintf("struct %s", d.Name),
				})
			case *ast.EnumDecl:
				items = append(items, CompletionItem{
					Label:  d.Name,
					Kind:   KindEnum,
					Detail: fmt.Sprintf("enum %s", d.Name),
				})
			}
		}
	}

	return CompletionList{
		IsIncomplete: false,
		Items:        items,
	}
}
