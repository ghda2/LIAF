package checker

import (
	"fmt"

	"liaf/pkg/ast"
)

// Regras de (middleware ...) e do with(...) das rotas (issue #045).

const middlewareReturn = "(result Request Response)"

// middlewareSignature exige a assinatura fixa (req Request) (result Request Response).
// ok(req) segue para a rota (com a Request possivelmente alterada); err(resposta)
// encerra a cadeia e devolve essa resposta ao cliente.
func (c *Checker) middlewareSignature(f *ast.FuncDecl) {
	if len(f.Params) != 1 || name(f.Params[0].Type) != "Request" || f.ReturnType == nil || f.ReturnType.String() != middlewareReturn {
		c.error(f, "E_MIDDLEWARE_SIGNATURE", fmt.Sprintf("middleware %s requires exactly one Request parameter and returns %s", f.Name, middlewareReturn))
	}
}

// routeMiddlewares confere que cada nome do with(...) e um middleware declarado,
// sem repeticao. Os efeitos do middleware ficam na assinatura dele: a rota so
// declara os efeitos do proprio corpo.
func (c *Checker) routeMiddlewares(r *ast.RouteDecl) {
	seen := map[string]bool{}
	for _, mw := range r.Middlewares {
		if seen[mw] {
			c.error(r, "E_DUPLICATE_MIDDLEWARE", fmt.Sprintf("middleware %s listed twice in with(...) of route %s %s", mw, r.Method, r.Path))
			continue
		}
		seen[mw] = true
		f := c.findFunc(mw)
		if f == nil {
			c.error(r, "E_UNKNOWN_MIDDLEWARE", fmt.Sprintf("with(...) of route %s %s references %s, which is not declared", r.Method, r.Path, mw))
			continue
		}
		if !f.IsMiddleware {
			c.error(r, "E_UNKNOWN_MIDDLEWARE", fmt.Sprintf("with(...) of route %s %s references fn %s; declare it with 'middleware' instead of 'fn'", r.Method, r.Path, mw))
		}
	}
}
