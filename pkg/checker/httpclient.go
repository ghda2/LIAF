package checker

import (
	"fmt"
	"strings"

	"liaf/pkg/ast"
	"liaf/pkg/builtins"
	rt "liaf/pkg/runtime"
)

// http-fetch confere o metodo quando ele e literal, que e o caso comum:
// "POTS" ou "get " virariam falha em producao, na primeira chamada. Um
// metodo vindo de variavel (uma rota que repassa request-method) e aceito e
// validado pelo runtime.
func init() {
	builtins.RegisterSpecialCheck("http-fetch", func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
		c := ctx.(*Checker)
		reply := applied("result", &ast.NamedType{Name: "HttpReply"}, primitive("str"))
		c.effect(v, "net")
		if !c.arity(v, 4) {
			return reply
		}
		if lit, ok := v.Args[0].(*ast.StringLiteral); ok {
			c.Info.Types[v.Args[0]] = primitive("str")
			known := false
			for _, m := range rt.FetchMethods {
				known = known || m == lit.Value
			}
			if !known {
				c.error(v, "E_UNKNOWN_METHOD", fmt.Sprintf("unknown HTTP method %q, expected one of %s", lit.Value, strings.Join(rt.FetchMethods, ", ")))
			}
		} else {
			c.require(v, c.expr(v.Args[0], nil), primitive("str"))
		}
		c.require(v, c.expr(v.Args[1], nil), primitive("str"))
		c.require(v, c.expr(v.Args[2], applied("list", primitive("str"))), applied("list", primitive("str")))
		c.require(v, c.expr(v.Args[3], nil), primitive("str"))
		return reply
	})
}
