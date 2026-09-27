package checker

import (
	"fmt"

	"liaf/pkg/ast"
	"liaf/pkg/builtins"
)

func init() {
	// Unários numéricos polimórficos: neg, abs
	checkUnaryNumeric := func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
		c := ctx.(*Checker)
		if !c.arity(v, 1) {
			return nil
		}
		t := c.expr(v.Args[0], want)
		if t == nil {
			return nil
		}
		tn := name(t)
		if tn != "int" && tn != "float" {
			c.error(v.Args[0], "E_TYPE_MISMATCH", fmt.Sprintf("Expected numeric type (int or float), received %s", tn))
			return primitive("int")
		}
		return t
	}
	builtins.RegisterSpecialCheck("neg", checkUnaryNumeric)
	builtins.RegisterSpecialCheck("abs", checkUnaryNumeric)

	// Binários numéricos homogêneos: min, max, pow
	checkBinarySameNumeric := func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
		c := ctx.(*Checker)
		if !c.arity(v, 2) {
			return nil
		}
		t1 := c.expr(v.Args[0], want)
		t2 := c.expr(v.Args[1], want)
		if t1 == nil || t2 == nil {
			return nil
		}
		n1 := name(t1)
		n2 := name(t2)
		if n1 != "int" && n1 != "float" {
			c.error(v.Args[0], "E_TYPE_MISMATCH", fmt.Sprintf("Expected numeric type, received %s", n1))
			return primitive("int")
		}
		if n1 != n2 {
			c.error(v, "E_TYPE_MISMATCH", fmt.Sprintf("Type mismatch in %s: cannot mix %s and %s; use float-from-int to convert", v.Func, n1, n2))
			return t1
		}
		return t1
	}
	builtins.RegisterSpecialCheck("min", checkBinarySameNumeric)
	builtins.RegisterSpecialCheck("max", checkBinarySameNumeric)
	builtins.RegisterSpecialCheck("pow", checkBinarySameNumeric)
}
