package checker_test

import (
	"testing"

	"liaf/pkg/checker"
	"liaf/pkg/lexer"
	"liaf/pkg/parser"
)

func parseAndCheck(t *testing.T, src, file string) []string {
	t.Helper()
	l := lexer.New(src)
	p := parser.New(l, file)
	mod := p.ParseModule()
	if len(p.Diagnostics) > 0 {
		t.Fatalf("parser errors: %+v", p.Diagnostics)
	}
	_, diags := checker.Check(mod, file)
	var codes []string
	for _, d := range diags {
		codes = append(codes, d.Code)
	}
	return codes
}

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

func TestVisibilityChecker(t *testing.T) {
	t.Run("privado_no_mesmo_modulo_funciona", func(t *testing.T) {
		src := `(module main
			(struct Interno (fields (v int)))
			(enum Cor Vermelho Azul)
			(fn segredo (params) (returns int) (effects) (body (return 42)))
			(fn main (params) (returns void) (effects)
				(body
					(let s Interno (new Interno 1))
					(let c Cor Cor.Vermelho)
					(let x int (segredo))))
		)`
		codes := parseAndCheck(t, src, "main.liaf")
		if len(codes) > 0 {
			t.Fatalf("esperava nenhum erro no mesmo módulo, obteve: %v", codes)
		}
	})

	t.Run("funcao_privada_de_outro_modulo_gera_erro", func(t *testing.T) {
		l := lexer.New(`(module auth
			(fn helper-privado (params) (returns int) (effects) (body (return 1)))
			(pub fn login (params) (returns int) (effects) (body (return 2)))
		)`)
		p := parser.New(l, "auth.liaf")
		modAuth := p.ParseModule()

		lMain := lexer.New(`(module main
			(fn main (params) (returns void) (effects)
				(body
					(let ok int (login))
					(let err int (helper-privado))))
		)`)
		pMain := parser.New(lMain, "main.liaf")
		modMain := pMain.ParseModule()

		// Simula mescla pelo loader:
		modMain.Decls = append(modMain.Decls, modAuth.Decls...)

		_, diags := checker.Check(modMain, "main.liaf")
		foundPrivate := false
		for _, d := range diags {
			if d.Code == "E_PRIVATE_SYMBOL" {
				foundPrivate = true
				break
			}
		}
		if !foundPrivate {
			t.Fatalf("esperava E_PRIVATE_SYMBOL para helper-privado, obteve: %+v", diags)
		}
	})

	t.Run("struct_privada_de_outro_modulo_gera_erro", func(t *testing.T) {
		l := lexer.New(`(module auth
			(struct Segredo (fields (token str)))
			(pub struct Sessao (fields (user str)))
		)`)
		p := parser.New(l, "auth.liaf")
		modAuth := p.ParseModule()

		lMain := lexer.New(`(module main
			(fn main (params) (returns void) (effects)
				(body
					(let s Sessao (new Sessao "admin"))
					(let x Segredo (new Segredo "tok"))))
		)`)
		pMain := parser.New(lMain, "main.liaf")
		modMain := pMain.ParseModule()

		modMain.Decls = append(modMain.Decls, modAuth.Decls...)

		_, diags := checker.Check(modMain, "main.liaf")
		foundPrivate := false
		for _, d := range diags {
			if d.Code == "E_PRIVATE_SYMBOL" {
				foundPrivate = true
				break
			}
		}
		if !foundPrivate {
			t.Fatalf("esperava E_PRIVATE_SYMBOL para Segredo, obteve: %+v", diags)
		}
	})

	t.Run("modulos_distintos_com_funcoes_privadas_de_mesmo_nome_nao_colidem", func(t *testing.T) {
		lA := lexer.New(`(module a
			(fn helper (params) (returns int) (effects) (body (return 1)))
			(pub fn a-func (params) (returns int) (effects) (body (return (helper))))
		)`)
		modA := parser.New(lA, "a.liaf").ParseModule()

		lB := lexer.New(`(module b
			(fn helper (params) (returns int) (effects) (body (return 2)))
			(pub fn b-func (params) (returns int) (effects) (body (return (helper))))
		)`)
		modB := parser.New(lB, "b.liaf").ParseModule()

		lMain := lexer.New(`(module main
			(fn helper (params) (returns int) (effects) (body (return 3)))
			(fn main (params) (returns void) (effects)
				(body
					(let a int (a-func))
					(let b int (b-func))
					(let m int (helper))))
		)`)
		modMain := parser.New(lMain, "main.liaf").ParseModule()

		modMain.Decls = append(modMain.Decls, modA.Decls...)
		modMain.Decls = append(modMain.Decls, modB.Decls...)

		_, diags := checker.Check(modMain, "main.liaf")
		if len(diags) > 0 {
			t.Fatalf("esperava zero erros com helpers privados isolados em módulos distintos, obteve: %+v", diags)
		}
	})
}
