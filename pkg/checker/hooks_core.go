package checker

import (
	"liaf/pkg/ast"
	"liaf/pkg/builtins"
)

func init() {
	// Variádicos com validação escalar (println, print, concat)
	registerScalarVariadic := func(name string, retType string) {
		builtins.RegisterSpecialCheck(name, func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
			c := ctx.(*Checker)
			for _, a := range v.Args {
				t := c.expr(a, nil)
				if t != nil && !scalar(t) {
					c.error(v, "E_TYPE_MISMATCH", "Use explicit serialization for composite values")
				}
			}
			return primitive(retType)
		})
	}
	registerScalarVariadic("println", "void")
	registerScalarVariadic("print", "void")
	registerScalarVariadic("concat", "str")

	// make-chan
	builtins.RegisterSpecialCheck("make-chan", func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
		c := ctx.(*Checker)
		if !c.arity(v, 1) {
			return nil
		}
		t := applied("chan", c.typeArg(v.Args[0]))
		c.validType(t, false)
		return t
	})

	// ok e err
	registerResultCtor := func(name string, argIdx int) {
		builtins.RegisterSpecialCheck(name, func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
			c := ctx.(*Checker)
			if !c.arity(v, 1) {
				return nil
			}
			a := parts(want, "result", 2)
			if a == nil {
				c.error(v, "E_RESULT_CONTEXT", "ok/err require an explicit result type")
				c.expr(v.Args[0], nil)
				return nil
			}
			c.require(v, c.expr(v.Args[0], a[argIdx]), a[argIdx])
			return want
		})
	}
	registerResultCtor("ok", 0)
	registerResultCtor("err", 1)

	// json-decode
	builtins.RegisterSpecialCheck("json-decode", func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
		c := ctx.(*Checker)
		if !c.arity(v, 2) {
			return nil
		}
		c.require(v, c.expr(v.Args[0], nil), primitive("str"))
		return applied("result", c.typeArg(v.Args[1]), primitive("str"))
	})

	// json-encode
	builtins.RegisterSpecialCheck("json-encode", func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
		c := ctx.(*Checker)
		c.arity(v, 1)
		c.expr(v.Args[0], nil)
		return applied("result", primitive("str"), primitive("str"))
	})

	// new
	builtins.RegisterSpecialCheck("new", func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
		c := ctx.(*Checker)
		if len(v.Args) == 0 {
			c.arity(v, 1)
			return nil
		}
		t := c.typeArg(v.Args[0])
		s := c.Info.Structs[name(t)]
		if s == nil {
			c.error(v, "E_INVALID_TYPE", "new requires struct")
			return nil
		}
		if !c.arity(v, len(s.Fields)+1) {
			return nil
		}
		for i, f := range s.Fields {
			c.require(v, c.expr(v.Args[i+1], f.Type), f.Type)
		}
		return t
	})

	// field
	builtins.RegisterSpecialCheck("field", func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
		c := ctx.(*Checker)
		if !c.arity(v, 2) {
			return nil
		}
		t := c.expr(v.Args[0], nil)
		id, ok := v.Args[1].(*ast.IdentExpr)
		if !ok {
			c.error(v, "E_UNKNOWN_FIELD", "Expected field name")
			return nil
		}
		if s := c.Info.Structs[name(t)]; s != nil {
			for _, f := range s.Fields {
				if f.Name == id.Name {
					return f.Type
				}
			}
		}
		c.error(v, "E_UNKNOWN_FIELD", id.Name)
		return nil
	})

	// some
	builtins.RegisterSpecialCheck("some", func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
		c := ctx.(*Checker)
		if !c.arity(v, 1) {
			return nil
		}
		var elemType ast.Type
		if wantOpt := parts(want, "option", 1); wantOpt != nil {
			elemType = wantOpt[0]
		}
		t := c.expr(v.Args[0], elemType)
		if elemType != nil {
			c.require(v, t, elemType)
			return want
		}
		return applied("option", t)
	})

	// none
	builtins.RegisterSpecialCheck("none", func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
		c := ctx.(*Checker)
		if !c.arity(v, 1) {
			return nil
		}
		t := c.typeArg(v.Args[0])
		c.validType(t, false)
		return applied("option", t)
	})

	// unwrap-or
	builtins.RegisterSpecialCheck("unwrap-or", func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
		c := ctx.(*Checker)
		if !c.arity(v, 2) {
			return nil
		}
		t1 := c.expr(v.Args[0], nil)
		if res := parts(t1, "result", 2); res != nil {
			elemType := res[0]
			c.consume(v.Args[0])
			t2 := c.expr(v.Args[1], elemType)
			c.require(v.Args[1], t2, elemType)
			return elemType
		}
		if opt := parts(t1, "option", 1); opt != nil {
			elemType := opt[0]
			c.consume(v.Args[0])
			t2 := c.expr(v.Args[1], elemType)
			c.require(v.Args[1], t2, elemType)
			return elemType
		}
		c.error(v.Args[0], "E_TYPE_MISMATCH", "unwrap-or requires result or option as first argument")
		return primitive("void")
	})

	// fmt
	builtins.RegisterSpecialCheck("fmt", func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
		c := ctx.(*Checker)
		if len(v.Args) < 1 {
			c.error(v, "E_ARITY_MISMATCH", "fmt requires at least 1 argument")
			return primitive("str")
		}
		t0 := c.expr(v.Args[0], primitive("str"))
		c.require(v.Args[0], t0, primitive("str"))
		for i := 1; i < len(v.Args); i++ {
			c.expr(v.Args[i], nil)
		}
		return primitive("str")
	})
}
