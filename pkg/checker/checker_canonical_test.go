package checker_test

import (
	"testing"

	"liaf/pkg/checker"
	"liaf/pkg/lexer"
	"liaf/pkg/parser"
)

// Formas canonicas (issue #012): comparar com true/false e outra forma de
// escrever x ou (not x), e so pode existir uma.
func TestCheckRejectsRedundantBoolCompare(t *testing.T) {
	for _, tc := range []struct{ expr, patch string }{
		{"(eq ativo false)", "(not ativo)"},
		{"(eq false ativo)", "(not ativo)"},
		{"(neq ativo true)", "(not ativo)"},
		{"(eq ativo true)", "ativo"},
		{"(neq (gt n 0) false)", "(gt n 0)"},
	} {
		t.Run(tc.expr, func(t *testing.T) {
			src := `(module m
  (fn f (params (ativo bool) (n int)) (returns bool) (effects)
    (body (return ` + tc.expr + `)))
  (fn main (params) (returns void) (effects) (body)))`
			p := parser.New(lexer.New(src), "m.liaf")
			mod := p.ParseModule()
			if len(p.Diagnostics) > 0 {
				t.Fatalf("parser: %+v", p.Diagnostics)
			}
			_, errs := checker.Check(mod, "m.liaf")
			if len(errs) != 1 || errs[0].Code != "E_REDUNDANT_BOOL_COMPARE" {
				t.Fatalf("esperado E_REDUNDANT_BOOL_COMPARE, recebido %+v", errs)
			}
			if errs[0].SuggestedPatch != tc.patch {
				t.Fatalf("patch: esperado %q, recebido %q", tc.patch, errs[0].SuggestedPatch)
			}
		})
	}
}

func TestCheckAcceptsBoolCompareBetweenVariables(t *testing.T) {
	codes := diagCodes(t, `(module m
  (fn f (params (a bool) (b bool)) (returns bool) (effects)
    (body (return (and (eq a b) (not a) (neq a b)))))
  (fn main (params) (returns void) (effects) (body)))`)
	if len(codes) > 0 {
		t.Fatalf("comparacao entre variaveis foi recusada: %v", codes)
	}
}
