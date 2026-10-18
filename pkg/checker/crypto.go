package checker

import (
	"fmt"

	"liaf/pkg/ast"
	"liaf/pkg/builtins"
)

// A codificacao de sha256 e hmac-sha256 e literal pela mesma razao que o
// driver de db-connect: "base64" (com padding e alfabeto padrao) ou "HEX"
// seriam erros faceis de cometer e so apareceriam como assinatura que nunca
// confere, em tempo de execucao.
var digestEncodings = []string{"hex", "base64url"}

func init() {
	digest := func(nStr int) func(builtins.CheckContext, *ast.CallExpr, ast.Type) ast.Type {
		return func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
			c := ctx.(*Checker)
			if !c.arity(v, nStr+1) {
				return primitive("str")
			}
			for i := 0; i < nStr; i++ {
				c.require(v, c.expr(v.Args[i], nil), primitive("str"))
			}
			c.digestEncoding(v, nStr)
			return primitive("str")
		}
	}
	builtins.RegisterSpecialCheck("sha256", digest(1))
	builtins.RegisterSpecialCheck("hmac-sha256", digest(2))
}

func (c *Checker) digestEncoding(v *ast.CallExpr, i int) {
	lit, ok := v.Args[i].(*ast.StringLiteral)
	if !ok {
		c.expr(v.Args[i], nil)
		c.error(v, "E_UNKNOWN_ENCODING", fmt.Sprintf("the %s encoding must be a string literal: \"hex\" or \"base64url\"", v.Func))
		return
	}
	c.Info.Types[v.Args[i]] = primitive("str")
	for _, e := range digestEncodings {
		if lit.Value == e {
			return
		}
	}
	c.error(v, "E_UNKNOWN_ENCODING", fmt.Sprintf("unknown %s encoding %q, expected \"hex\" or \"base64url\"", v.Func, lit.Value))
}
