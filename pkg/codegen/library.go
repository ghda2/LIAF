package codegen

import (
	"fmt"
	"strings"

	"liaf/pkg/ast"
	"liaf/pkg/builtins"
)

// libraryCall consulta a tabela única de builtins para gerar o código Go correspondente.
func (g *Generator) libraryCall(c *ast.CallExpr) (string, bool) {
	b := builtins.Lookup(c.Func)
	if b == nil {
		return "", false
	}
	args := make([]string, len(c.Args))
	for i, a := range c.Args {
		args[i] = g.genExpr(a)
	}
	if b.GoEmit != nil {
		return b.GoEmit(g, c, args)
	}
	if b.GoTemplate != "" {
		if (b.Arity.Max == -1 || len(args) != 1) && (b.Name == "println" || b.Name == "print" || b.Name == "concat" || b.Name == "fmt") {
			return fmt.Sprintf(b.GoTemplate, strings.Join(args, ", ")), true
		}
		anyArgs := make([]any, len(args))
		for i, v := range args {
			anyArgs[i] = v
		}
		return fmt.Sprintf(b.GoTemplate, anyArgs...), true
	}
	if b.GoCall != "" {
		return b.GoCall + "(" + strings.Join(args, ", ") + ")", true
	}
	return "", false
}
