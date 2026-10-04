package checker

import (
	"fmt"
	"strings"

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
	registerScalarVariadic("log-info", "void")
	registerScalarVariadic("log-error", "void")
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
		s := c.findStruct(name(t))
		if s == nil {
			c.error(v, "E_INVALID_TYPE", "new requires struct")
			return nil
		}
		if !c.checkStructVisibility(s, v) {
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
		// Se a base for um enum diretamente (ex: Status.Pendente)
		if id0, ok := v.Args[0].(*ast.IdentExpr); ok {
			if enumDecl := c.findEnum(id0.Name); enumDecl != nil {
				if !c.checkEnumVisibility(enumDecl, v) {
					return nil
				}
				id1, ok1 := v.Args[1].(*ast.IdentExpr)
				if !ok1 {
					c.error(v, "E_UNKNOWN_FIELD", "Expected variant name")
					return nil
				}
				for _, variant := range enumDecl.Variants {
					if variant.Name == id1.Name {
						return &ast.NamedType{Name: enumDecl.Name}
					}
				}
				c.error(v, "E_UNKNOWN_VARIANT", fmt.Sprintf("Enum %s has no variant %s", enumDecl.Name, id1.Name))
				return nil
			}
		}

		t := c.expr(v.Args[0], nil)
		id, ok := v.Args[1].(*ast.IdentExpr)
		if !ok {
			c.error(v, "E_UNKNOWN_FIELD", "Expected field name")
			return nil
		}
		if s := c.findStruct(name(t)); s != nil {
			for _, f := range s.Fields {
				if f.Name == id.Name {
					return f.Type
				}
			}
		}
		c.error(v, "E_UNKNOWN_FIELD", id.Name)
		return nil
	})

	builtins.RegisterSpecialCheck("variant-call", func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
		c := ctx.(*Checker)
		if len(v.Args) < 1 {
			return nil
		}
		fieldCall, ok := v.Args[0].(*ast.CallExpr)
		if !ok || fieldCall.Func != "field" || len(fieldCall.Args) != 2 {
			c.error(v, "E_INVALID_VARIANT_CALL", "Expected Enum.Variant")
			return nil
		}
		id0, ok0 := fieldCall.Args[0].(*ast.IdentExpr)
		id1, ok1 := fieldCall.Args[1].(*ast.IdentExpr)
		if !ok0 || !ok1 {
			return nil
		}
		enumDecl := c.findEnum(id0.Name)
		if enumDecl == nil {
			c.error(v, "E_UNKNOWN_TYPE", id0.Name)
			return nil
		}
		if !c.checkEnumVisibility(enumDecl, v) {
			return nil
		}
		var targetVariant *ast.EnumVariant
		for _, variant := range enumDecl.Variants {
			if variant.Name == id1.Name {
				targetVariant = &variant
				break
			}
		}
		if targetVariant == nil {
			c.error(v, "E_UNKNOWN_VARIANT", fmt.Sprintf("Enum %s has no variant %s", enumDecl.Name, id1.Name))
			return nil
		}
		actualArgs := v.Args[1:]
		if len(actualArgs) != len(targetVariant.Fields) {
			c.error(v, "E_ARITY_MISMATCH", fmt.Sprintf("Variant %s expects %d arguments, received %d", targetVariant.Name, len(targetVariant.Fields), len(actualArgs)))
			return nil
		}
		for i, arg := range actualArgs {
			argType := c.expr(arg, targetVariant.Fields[i].Type)
			c.require(arg, argType, targetVariant.Fields[i].Type)
		}
		return &ast.NamedType{Name: enumDecl.Name}
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
		if lit, ok := v.Args[0].(*ast.StringLiteral); ok {
			trimmed := strings.TrimSpace(lit.Value)
			if (strings.HasPrefix(trimmed, "{\"") || strings.HasPrefix(trimmed, "[{\"")) && len(v.Args) > 1 {
				c.error(v.Args[0], "W_MANUAL_JSON_FMT", "do not build JSON strings using fmt; use json-encode with a struct to avoid injection attacks")
			}
		}
		for i := 1; i < len(v.Args); i++ {
			c.expr(v.Args[i], nil)
		}
		return primitive("str")
	})
}
