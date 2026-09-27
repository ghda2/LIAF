package parser

import (
	"strings"
	"testing"

	"liaf/pkg/ast"
	"liaf/pkg/lexer"
)

// s.campo (issue #012) e a mesma AST de (field s campo): checker e codegen nao
// mudam, e liafc fmt imprime a forma com ponto.
func TestFieldPathDesugarsToField(t *testing.T) {
	src := `(module m
  (struct A (fields (b B)))
  (struct B (fields (c int) (sub str)))
  (fn f (params (a A)) (returns int) (effects)
    (body
      (let s str a.b.sub)
      (let x int (field (field a b) c))
      (return a.b.c))))`
	p := New(lexer.New(src), "m.liaf")
	mod := p.ParseModule()
	if len(p.Diagnostics) > 0 {
		t.Fatalf("diagnosticos: %+v", p.Diagnostics)
	}
	body := mod.Decls[2].(*ast.FuncDecl).Body
	ret := body[2].(*ast.ReturnStmt).Value.(*ast.CallExpr)
	inner, ok := ret.Args[0].(*ast.CallExpr)
	if ret.Func != "field" || !ok || inner.Func != "field" || ret.Args[1].(*ast.IdentExpr).Name != "c" {
		t.Fatalf("a.b.c nao virou (field (field a b) c): %#v", ret)
	}

	printed := ast.Format(mod)
	for _, want := range []string{"(let s str a.b.sub)", "(let x int a.b.c)", "(return a.b.c)"} {
		if !strings.Contains(printed, want) {
			t.Fatalf("fmt nao produziu %q:\n%s", want, printed)
		}
	}
	p2 := New(lexer.New(printed), "m.liaf")
	mod2 := p2.ParseModule()
	if len(p2.Diagnostics) > 0 {
		t.Fatalf("forma canonica nao reparseia: %+v\n%s", p2.Diagnostics, printed)
	}
	if again := ast.Format(mod2); again != printed {
		t.Fatalf("formato instavel:\n%s\n---\n%s", printed, again)
	}
}

// Base que nao e variavel continua com parenteses: nao existe (get xs 0).nome.
func TestFieldPathKeepsParensForComplexBase(t *testing.T) {
	src := `(module m
  (struct U (fields (nome str)))
  (fn f (params (xs (list U))) (returns str) (effects)
    (body (return (field (try (get xs 0)) nome)))))`
	p := New(lexer.New(src), "m.liaf")
	mod := p.ParseModule()
	if len(p.Diagnostics) > 0 {
		t.Fatalf("diagnosticos: %+v", p.Diagnostics)
	}
	if printed := ast.Format(mod); !strings.Contains(printed, "(field (try (get xs 0)) nome)") {
		t.Fatalf("base composta foi alterada:\n%s", printed)
	}
}

func TestFieldPathRejectsBadSegments(t *testing.T) {
	for _, tc := range []struct{ src, code string }{
		{`(module m (fn f (params) (returns void) (effects) (body (let x int if.y))))`, "E_INVALID_EXPR"},
		{`(module m (fn f (params (s S)) (returns void) (effects) (body (let x int s.true))))`, "E_EXPECTED_FIELD_NAME"},
	} {
		p := New(lexer.New(tc.src), "m.liaf")
		p.ParseModule()
		if len(p.Diagnostics) == 0 || p.Diagnostics[0].Code != tc.code {
			t.Fatalf("%s: esperado %s, recebido %+v", tc.src, tc.code, p.Diagnostics)
		}
	}
}
