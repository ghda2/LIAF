package parser

import (
	"strings"
	"testing"

	"liaf/pkg/ast"
	"liaf/pkg/lexer"
)

func TestParseCanonicalMathExample(t *testing.T) {
	input := `
(module example
  (fn sum
    (params
      (a int)
      (b int))
    (returns int)
    (effects)
    (body
      (return (add a b))))
  (fn main
    (params)
    (returns void)
    (effects io)
    (body
      (do (call println (call sum 20 22))))))
`

	l := lexer.New(input)
	p := New(l, "example.liaf")
	mod := p.ParseModule()

	if len(p.Diagnostics) > 0 {
		t.Fatalf("diagnósticos inesperados: %+v", p.Diagnostics)
	}

	if mod == nil {
		t.Fatalf("mod é nulo")
	}

	if mod.Name != "example" {
		t.Errorf("esperado nome do módulo 'example', obteve %q", mod.Name)
	}

	if len(mod.Decls) != 2 {
		t.Fatalf("esperado 2 declarações, obteve %d", len(mod.Decls))
	}

	formatted := ast.Format(mod)
	expectedClean := strings.TrimSpace(input)
	formattedClean := strings.TrimSpace(formatted)

	if expectedClean != formattedClean {
		t.Errorf("formatação canônica divergente!\nEsperado:\n%s\nObtido:\n%s", expectedClean, formattedClean)
	}
}

func TestParseConcurrencyExample(t *testing.T) {
	input := `
(module concurrency-example
  (fn worker
    (params
      (channel (chan str))
      (id int))
    (returns void)
    (effects clock)
    (body
      (do (call sleep-ms 50))
      (send channel (call concat "worker-" (call str-from-int id)))))
  (fn main
    (params)
    (returns void)
    (effects io spawn)
    (body
      (let channel (chan str) (call make-chan str))
      (spawn (call worker channel 1))
      (spawn (call worker channel 2))
      (let first str (recv channel))
      (let second str (recv channel))
      (do (call println first))
      (do (call println second)))))
`

	l := lexer.New(input)
	p := New(l, "concurrency.liaf")
	mod := p.ParseModule()

	if len(p.Diagnostics) > 0 {
		t.Fatalf("diagnósticos inesperados: %+v", p.Diagnostics)
	}

	if mod == nil {
		t.Fatalf("mod é nulo")
	}

	if len(mod.Decls) != 2 {
		t.Fatalf("esperado 2 declarações, obteve %d", len(mod.Decls))
	}

	fnWorker, ok := mod.Decls[0].(*ast.FuncDecl)
	if !ok || fnWorker.Name != "worker" {
		t.Fatalf("esperado fn worker")
	}

	if len(fnWorker.Params) != 2 {
		t.Fatalf("esperado 2 params em worker, obteve %d", len(fnWorker.Params))
	}

	chanType, ok := fnWorker.Params[0].Type.(*ast.AppliedType)
	if !ok || chanType.Constructor != "chan" {
		t.Fatalf("esperado param channel do tipo (chan str)")
	}

	fnMain, ok := mod.Decls[1].(*ast.FuncDecl)
	if !ok || fnMain.Name != "main" {
		t.Fatalf("esperado fn main")
	}

	if len(fnMain.Effects) != 2 || fnMain.Effects[0] != "io" || fnMain.Effects[1] != "spawn" {
		t.Errorf("efeitos incorretos em main: %v", fnMain.Effects)
	}

	formatted := ast.Format(mod)
	if strings.TrimSpace(formatted) != strings.TrimSpace(input) {
		t.Errorf("formatação canônica divergente!\nEsperado:\n%s\nObtido:\n%s", strings.TrimSpace(input), strings.TrimSpace(formatted))
	}
}

func TestParseStructAndImport(t *testing.T) {
	input := `
(module domain
  (import "std/strings")
  (struct User
    (fields
      (id int)
      (name str)))
  (fn get-name
    (params
      (user User))
    (returns str)
    (effects)
    (body
      (return user))))
`

	l := lexer.New(input)
	p := New(l, "domain.liaf")
	mod := p.ParseModule()

	if len(p.Diagnostics) > 0 {
		t.Fatalf("diagnósticos inesperados: %+v", p.Diagnostics)
	}

	if len(mod.Decls) != 3 {
		t.Fatalf("esperado 3 declarações, obteve %d", len(mod.Decls))
	}

	imp, ok := mod.Decls[0].(*ast.ImportDecl)
	if !ok || imp.Path != "std/strings" {
		t.Fatalf("esperado import std/strings")
	}

	st, ok := mod.Decls[1].(*ast.StructDecl)
	if !ok || st.Name != "User" {
		t.Fatalf("esperado struct User")
	}

	if len(st.Fields) != 2 || st.Fields[0].Name != "id" || st.Fields[1].Name != "name" {
		t.Fatalf("campos de User incorretos: %+v", st.Fields)
	}
}

func TestParseIfThenElse(t *testing.T) {
	input := `
(module control
  (fn check
    (params
      (val int))
    (returns void)
    (effects io)
    (body
      (if (gt val 0)
        (then
          (do (call println "positivo")))
        (else
          (do (call println "zero ou negativo")))))))
`

	l := lexer.New(input)
	p := New(l, "control.liaf")
	mod := p.ParseModule()

	if len(p.Diagnostics) > 0 {
		t.Fatalf("diagnósticos inesperados: %+v", p.Diagnostics)
	}

	fn := mod.Decls[0].(*ast.FuncDecl)
	ifStmt := fn.Body[0].(*ast.IfStmt)

	if len(ifStmt.Then) != 1 {
		t.Errorf("esperado 1 stmt em then, obteve %d", len(ifStmt.Then))
	}
	if len(ifStmt.Else) != 1 {
		t.Errorf("esperado 1 stmt em else, obteve %d", len(ifStmt.Else))
	}
}

func TestParseErrors(t *testing.T) {
	badInput := `(module erro (fn foo (params) (returns void) (effects) (body)))`
	l := lexer.New(badInput)
	p := New(l, "erro.liaf")
	mod := p.ParseModule()
	if len(p.Diagnostics) == 0 && mod != nil {
		t.Logf("ParseModule com sucesso: %v", mod.Name)
	}

	badInput2 := `module sem-parenteses`
	l2 := lexer.New(badInput2)
	p2 := New(l2, "erro2.liaf")
	mod2 := p2.ParseModule()
	if mod2 != nil || len(p2.Diagnostics) == 0 {
		t.Errorf("esperava falha ao não iniciar com '('")
	}
}
