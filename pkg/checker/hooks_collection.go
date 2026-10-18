package checker

import (
	"liaf/pkg/ast"
	"liaf/pkg/builtins"
)

func init() {
	// make-list e make-map
	builtins.RegisterSpecialCheck("make-list", func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
		c := ctx.(*Checker)
		if !c.arity(v, 1) {
			return nil
		}
		t := applied("list", c.typeArg(v.Args[0]))
		c.validType(t, false)
		return t
	})

	builtins.RegisterSpecialCheck("make-map", func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
		c := ctx.(*Checker)
		if !c.arity(v, 2) {
			return nil
		}
		t := applied("map", c.typeArg(v.Args[0]), c.typeArg(v.Args[1]))
		c.validType(t, false)
		return t
	})

	// Operações de List e Map com polimorfismo genérico
	for _, listFn := range []string{"list-len", "list-push", "list-get", "list-set", "list-remove", "list-pop", "list-sort", "list-contains"} {
		fnName := listFn
		builtins.RegisterSpecialCheck(fnName, func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
			c := ctx.(*Checker)
			if len(v.Args) == 0 {
				c.arity(v, 1)
				return nil
			}
			args := make([]ast.Type, len(v.Args))
			for i, a := range v.Args {
				args[i] = c.expr(a, nil)
			}
			a := parts(args[0], "list", 1)
			if a == nil {
				c.error(v, "E_TYPE_MISMATCH", "Expected list")
				return nil
			}
			switch fnName {
			case "list-len":
				c.arity(v, 1)
				return primitive("int")
			case "list-push":
				if c.arity(v, 2) {
					c.require(v, args[1], a[0])
				}
				return primitive("void")
			case "list-get":
				if c.arity(v, 2) {
					c.require(v, args[1], primitive("int"))
				}
				return applied("result", a[0], primitive("str"))
			case "list-set":
				if c.arity(v, 3) {
					c.require(v, args[1], primitive("int"))
					c.require(v, args[2], a[0])
				}
				return applied("result", primitive("bool"), primitive("str"))
			case "list-remove":
				if c.arity(v, 2) {
					c.require(v, args[1], primitive("int"))
				}
				return applied("result", primitive("bool"), primitive("str"))
			case "list-pop":
				c.arity(v, 1)
				return applied("result", a[0], primitive("str"))
			case "list-sort":
				c.arity(v, 1)
				elemName := name(a[0])
				if elemName != "int" && elemName != "float" && elemName != "str" {
					c.error(v, "E_TYPE_MISMATCH", "list-sort requires int, float, or str elements")
				}
				return primitive("void")
			case "list-contains":
				if c.arity(v, 2) {
					c.require(v, args[1], a[0])
				}
				return primitive("bool")
			}
			return nil
		})
	}

	for _, mapFn := range []string{"map-len", "map-get", "map-set", "map-has", "map-delete", "map-keys"} {
		fnName := mapFn
		builtins.RegisterSpecialCheck(fnName, func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
			c := ctx.(*Checker)
			if len(v.Args) == 0 {
				c.arity(v, 1)
				return nil
			}
			args := make([]ast.Type, len(v.Args))
			for i, a := range v.Args {
				args[i] = c.expr(a, nil)
			}
			a := parts(args[0], "map", 2)
			if a == nil {
				c.error(v, "E_TYPE_MISMATCH", "Expected map")
				return nil
			}
			switch fnName {
			case "map-len":
				c.arity(v, 1)
				return primitive("int")
			case "map-has":
				if c.arity(v, 2) {
					c.require(v, args[1], a[0])
				}
				return primitive("bool")
			case "map-get":
				if c.arity(v, 2) {
					c.require(v, args[1], a[0])
				}
				return applied("result", a[1], primitive("str"))
			case "map-set":
				if c.arity(v, 3) {
					c.require(v, args[1], a[0])
					c.require(v, args[2], a[1])
				}
				return primitive("void")
			case "map-delete":
				if c.arity(v, 2) {
					c.require(v, args[1], a[0])
				}
				return primitive("void")
			case "map-keys":
				c.arity(v, 1)
				return applied("list", a[0])
			}
			return nil
		})
	}

	builtins.RegisterSpecialCheck("list", func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
		c := ctx.(*Checker)
		if len(v.Args) == 0 {
			if wantList := parts(want, "list", 1); wantList != nil {
				return want
			}
			c.error(v, "E_TYPE_MISMATCH", "cannot infer element type for empty list")
			return applied("list", primitive("int"))
		}
		var elemType ast.Type
		if wantList := parts(want, "list", 1); wantList != nil {
			elemType = wantList[0]
		}
		first := c.expr(v.Args[0], elemType)
		if elemType == nil {
			elemType = first
		}
		for i := 1; i < len(v.Args); i++ {
			c.require(v.Args[i], c.expr(v.Args[i], elemType), elemType)
		}
		return applied("list", elemType)
	})
}
