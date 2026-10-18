package ast_test

import (
	"strings"
	"testing"

	"liaf/pkg/ast"
	"liaf/pkg/lexer"
	"liaf/pkg/parser"
)

func TestFormatIfExpr(t *testing.T) {
	src := `(module test
  (fn main (params) (returns void) (effects)
    (body
      (let x int (if true (then 1) (else 2))))))`

	p := parser.New(lexer.New(src), "test.liaf")
	m := p.ParseModule()
	if len(p.Diagnostics) > 0 {
		t.Fatalf("parser diagnostics: %v", p.Diagnostics)
	}

	formatted := ast.Format(m)
	if !strings.Contains(formatted, "(if true (then 1) (else 2))") {
		t.Fatalf("esperado formato canonico para IfExpr, obtido:\n%s", formatted)
	}
}
