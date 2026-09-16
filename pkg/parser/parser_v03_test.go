package parser_test

import (
	"strings"
	"testing"

	"liaf/pkg/ast"
	"liaf/pkg/lexer"
	"liaf/pkg/parser"
)

func TestParseV03DirectCallsAndTry(t *testing.T) {
	input := `(module test-v03
  (fn calculate
    (params
      (val int))
    (returns (result int str))
    (effects)
    (on-err msg
      (return (err msg)))
    (body
      (let res int (try (ok val)))
      (return (ok res))))
  (fn main
    (params)
    (returns void)
    (effects io)
    (body
      (let r (result int str) (calculate 42))
      (println "Done"))))`

	l := lexer.New(input)
	p := parser.New(l, "test_v03.liaf")
	mod := p.ParseModule()

	if len(p.Diagnostics) > 0 {
		t.Fatalf("diagnostics unexpected: %+v", p.Diagnostics)
	}
	if mod == nil {
		t.Fatalf("mod is nil")
	}

	formatted := ast.Format(mod)
	expectedClean := strings.TrimSpace(input)
	formattedClean := strings.TrimSpace(formatted)

	if expectedClean != formattedClean {
		t.Errorf("Canonical formatting mismatch!\nExpected:\n%s\nGot:\n%s", expectedClean, formattedClean)
	}
}

func TestParseV03Route(t *testing.T) {
	input := `(module test-route
  (route GET "/tasks/{id}"
    (params
      (id int))
    (returns str)
    (effects io)
    (body
      (return "task-details")))
  (fn main
    (params)
    (returns void)
    (effects)
    (body)))`

	l := lexer.New(input)
	p := parser.New(l, "test_route.liaf")
	mod := p.ParseModule()

	if len(p.Diagnostics) > 0 {
		t.Fatalf("diagnostics unexpected: %+v", p.Diagnostics)
	}
	if len(mod.Decls) != 2 {
		t.Fatalf("expected 2 declarations, got %d", len(mod.Decls))
	}
	route, ok := mod.Decls[0].(*ast.RouteDecl)
	if !ok {
		t.Fatalf("expected RouteDecl, got %T", mod.Decls[0])
	}
	if route.Method != "GET" || route.Path != "/tasks/{id}" {
		t.Errorf("unexpected route: %s %s", route.Method, route.Path)
	}
}
