package checker

import (
	"fmt"

	"liaf/pkg/ast"
	"liaf/pkg/builtins"
)

// Implementação dos métodos de CheckContext em Checker
func (c *Checker) Expr(e ast.Expr, want ast.Type) ast.Type { return c.expr(e, want) }
func (c *Checker) Require(n ast.Node, actual, want ast.Type) { c.require(n, actual, want) }
func (c *Checker) Error(n ast.Node, code, msg string) { c.error(n, code, msg) }
func (c *Checker) Arity(v *ast.CallExpr, n int) bool { return c.arity(v, n) }
func (c *Checker) Effect(n ast.Node, eff string) { c.effect(n, eff) }
func (c *Checker) TypeArg(e ast.Expr) ast.Type { return c.typeArg(e) }
func (c *Checker) FindStruct(name string) *ast.StructDecl { return c.Info.Structs[name] }
func (c *Checker) FindFunc(name string) *ast.FuncDecl { return c.Info.Functions[name] }
func (c *Checker) TypeAt(e ast.Expr) ast.Type { return c.Info.Types[e] }

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
	registerScalarVariadic("fmt", "str")

	// make-list, make-map, make-chan
	builtins.RegisterSpecialCheck("make-list", func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
		c := ctx.(*Checker)
		if !c.arity(v, 1) {
			return nil
		}
		t := applied("list", c.typeArg(v.Args[0]))
		c.validType(t, false)
		return t
	})
	builtins.RegisterSpecialCheck("make-chan", func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
		c := ctx.(*Checker)
		if !c.arity(v, 1) {
			return nil
		}
		t := applied("chan", c.typeArg(v.Args[0]))
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

	// Banco de dados: db-connect, db-close, db-query, db-exec, redis-get, redis-set
	for _, dbFn := range []string{"db-connect", "db-close", "db-query", "db-exec", "redis-get", "redis-set"} {
		fnName := dbFn
		builtins.RegisterSpecialCheck(fnName, func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
			c := ctx.(*Checker)
			t, _ := c.dbCall(v, fnName)
			return t
		})
	}

	// WebSockets: ws-send, ws-send-json, ws-close, ws-broadcast, ws-join, ws-leave, ws-topic-size
	for _, wsFn := range []string{"ws-send", "ws-send-json", "ws-close", "ws-broadcast", "ws-join", "ws-leave", "ws-topic-size"} {
		fnName := wsFn
		builtins.RegisterSpecialCheck(fnName, func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
			c := ctx.(*Checker)
			t, _ := c.wsCall(v, fnName)
			return t
		})
	}

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

	builtins.RegisterSpecialCheck("none", func(ctx builtins.CheckContext, v *ast.CallExpr, want ast.Type) ast.Type {
		c := ctx.(*Checker)
		if !c.arity(v, 1) {
			return nil
		}
		t := c.typeArg(v.Args[0])
		c.validType(t, false)
		return applied("option", t)
	})

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
