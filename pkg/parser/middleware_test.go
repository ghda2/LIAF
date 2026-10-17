package parser

import (
	"reflect"
	"testing"

	"liaf/pkg/ast"
	"liaf/pkg/lexer"
)

func parseMiddlewareModule(t *testing.T, src string) *ast.Module {
	t.Helper()
	p := New(lexer.New(src), "m.liaf")
	mod := p.ParseModule()
	if len(p.Diagnostics) > 0 {
		t.Fatalf("erros de parse: %+v", p.Diagnostics)
	}
	return mod
}

func TestParseMiddlewareBothSyntaxes(t *testing.T) {
	linear := `module m

pub middleware guard(req Request) (result Request Response)
  ok(req)
end

route GET "/p/{id}" with(guard, outro) (id int) Response
  json-response(200, "{}")
end
`
	mod := parseMiddlewareModule(t, linear)
	assertMiddlewareDecls(t, mod)

	// A forma canonica do fmt (S-expression) le de volta a mesma coisa.
	again := parseMiddlewareModule(t, ast.Format(mod))
	assertMiddlewareDecls(t, again)
	if ast.Format(again) != ast.Format(mod) {
		t.Errorf("fmt nao e idempotente:\n%s\n---\n%s", ast.Format(mod), ast.Format(again))
	}
}

func assertMiddlewareDecls(t *testing.T, mod *ast.Module) {
	t.Helper()
	if len(mod.Decls) != 2 {
		t.Fatalf("esperado 2 declaracoes, obteve %d", len(mod.Decls))
	}
	f, ok := mod.Decls[0].(*ast.FuncDecl)
	if !ok || !f.IsMiddleware || !f.IsPub || f.Name != "guard" {
		t.Fatalf("middleware lido errado: %+v", mod.Decls[0])
	}
	r, ok := mod.Decls[1].(*ast.RouteDecl)
	if !ok {
		t.Fatalf("esperado rota, obteve %T", mod.Decls[1])
	}
	if !reflect.DeepEqual(r.Middlewares, []string{"guard", "outro"}) {
		t.Errorf("with(...) lido errado: %v", r.Middlewares)
	}
	if len(r.Params) != 1 || r.Params[0].Name != "id" {
		t.Errorf("parametros da rota perdidos depois do with: %+v", r.Params)
	}
}

func TestMiddlewareAndWithStayIdentifiers(t *testing.T) {
	// Fora das duas posicoes contextuais, os nomes continuam livres.
	parseMiddlewareModule(t, `module m

fn with(middleware int) int
  middleware
end
`)
}

func TestWithRequiresAName(t *testing.T) {
	p := New(lexer.New("module m\nroute GET \"/p\" with() () Response\n  json-response(200, \"{}\")\nend\n"), "m.liaf")
	p.ParseModule()
	found := false
	for _, d := range p.Diagnostics {
		if d.Code == "E_EXPECTED_MIDDLEWARE_NAME" {
			found = true
		}
	}
	if !found {
		t.Fatalf("esperado E_EXPECTED_MIDDLEWARE_NAME, obteve %+v", p.Diagnostics)
	}
}
