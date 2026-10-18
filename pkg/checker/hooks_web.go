package checker

import (
	"liaf/pkg/ast"
	"liaf/pkg/builtins"
)

func init() {
	// http-* handlers
	for _, m := range []string{"http-get", "http-post", "http-put", "http-delete"} {
		builtins.RegisterSpecialCheck(m, func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
			c := ctx.(*Checker)
			if !c.arity(v, 2) {
				return nil
			}
			c.require(v, c.expr(v.Args[0], nil), primitive("str"))
			id, ok := v.Args[1].(*ast.IdentExpr)
			var f *ast.FuncDecl
			if ok {
				f = c.Info.Functions[id.Name]
			}
			if f == nil || len(f.Params) != 1 || name(f.Params[0].Type) != "Request" || name(f.ReturnType) != "Response" {
				c.error(v, "E_HANDLER_SIGNATURE", "Handler requires (Request) -> Response")
			} else {
				for _, eff := range f.Effects {
					c.effect(v, eff)
				}
			}
			return primitive("void")
		})
	}
}
