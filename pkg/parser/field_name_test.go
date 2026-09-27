package parser

import (
	"testing"

	"liaf/pkg/ast"
	"liaf/pkg/lexer"
)

// Palavras reservadas servem de nome de campo (claims de JWT como "sub",
// chaves de APIs externas), sem perder o significado no resto do codigo.
func TestReservedWordsAsFieldNames(t *testing.T) {
	src := `(module m
  (struct Claims (fields (sub str) (exp int) (return str) (if bool)))
  (fn f (params (c Claims)) (returns int) (effects)
    (body
      (let s str (field c sub))
      (let r str (call field c return))
      (return (sub (field c exp) 1)))))`
	p := New(lexer.New(src), "m.liaf")
	mod := p.ParseModule()
	if len(p.Diagnostics) > 0 {
		t.Fatalf("diagnosticos: %+v", p.Diagnostics)
	}
	st := mod.Decls[0].(*ast.StructDecl)
	var names []string
	for _, f := range st.Fields {
		names = append(names, f.Name)
	}
	if got := len(names); got != 4 || names[0] != "sub" || names[2] != "return" || names[3] != "if" {
		t.Fatalf("campos: %v", names)
	}

	// A forma canonica tem de ser lida de volta igual: liafc fmt nao pode
	// transformar um programa valido num invalido.
	printed := ast.Format(mod)
	p2 := New(lexer.New(printed), "m.liaf")
	mod2 := p2.ParseModule()
	if len(p2.Diagnostics) > 0 {
		t.Fatalf("forma canonica nao reparseia: %+v\n%s", p2.Diagnostics, printed)
	}
	if again := ast.Format(mod2); again != printed {
		t.Fatalf("formato instavel:\n%s\n---\n%s", printed, again)
	}
}

// true e false tem valor: como nome de campo seriam ambiguos.
func TestBoolLiteralIsNotAFieldName(t *testing.T) {
	p := New(lexer.New(`(module m (struct S (fields (true int))))`), "m.liaf")
	p.ParseModule()
	if len(p.Diagnostics) == 0 || p.Diagnostics[0].Code != "E_EXPECTED_FIELD_NAME" {
		t.Fatalf("esperado E_EXPECTED_FIELD_NAME: %+v", p.Diagnostics)
	}
}
