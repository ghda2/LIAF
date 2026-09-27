package builtins

import (
	"fmt"
	"strings"

	"liaf/pkg/ast"
)

func init() {
	// ---------------------------------------------------------
	// Core IO
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:       "println",
		Category:   "io",
		Arity:      MinArity(0),
		Effects:    []string{"io"},
		Return:     "void",
		GoTemplate: "fmt.Println(%s)",
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			return ctx.Value(fmt.Sprintf("vprint(%d,%s,1)", len(args), ctx.Array(args))), true
		},
	})

	Register(&Builtin{
		Name:       "print",
		Category:   "io",
		Arity:      MinArity(0),
		Effects:    []string{"io"},
		Return:     "void",
		GoTemplate: "fmt.Print(%s)",
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			return ctx.Value(fmt.Sprintf("vprint(%d,%s,0)", len(args), ctx.Array(args))), true
		},
	})

	Register(&Builtin{
		Name:     "not",
		Category: "core",
		Arity:    ExactArity(1),
		Params:   []string{"bool"},
		Return:   "bool",
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			if len(args) > 0 {
				return fmt.Sprintf("!(%s)", args[0]), true
			}
			return "false", true
		},
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			if len(args) > 0 {
				return ctx.Value(fmt.Sprintf("vb(!%s.b)", args[0])), true
			}
			return ctx.Value("vb(0)"), true
		},
	})

	// ---------------------------------------------------------
	// Concurrency
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:            "make-chan",
		Category:        "concurrency",
		Arity:           ExactArity(1),
		UnevaluatedArgs: true,
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			elemType := "interface{}"
			if len(call.Args) > 0 {
				if id, ok := call.Args[0].(*ast.IdentExpr); ok {
					elemType = ctx.MapType(&ast.PrimitiveType{Name: id.Name})
				}
			}
			return fmt.Sprintf("make(chan %s)", elemType), true
		},
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			return ctx.Value("vchan()"), true
		},
	})

	// ---------------------------------------------------------
	// JSON
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:               "json-encode",
		Category:           "json",
		Arity:              ExactArity(1),
		Return:             "(result str str)",
		GoCall:             "rt.JSONEncode",
		UnsupportedC:       true,
		UnsupportedCReason: "json-encode nao implementado no backend C",
	})

	Register(&Builtin{
		Name:            "json-decode",
		Category:        "json",
		Arity:           ExactArity(2),
		UnevaluatedArgs: true,
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			tName := ctx.TypeName(call.Args[1])
			return fmt.Sprintf("rt.JSONDecode[%s](%s)", tName, args[0]), true
		},
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			ctx.SetError(fmt.Errorf("C backend: json-decode is not implemented yet"))
			return ctx.Value("vnone()"), true
		},
	})

	// ---------------------------------------------------------
	// Option: some / none
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:     "some",
		Category: "option",
		Arity:    ExactArity(1),
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			t := ctx.TypeOf(call)
			elemType := "interface{}"
			if optType := parts(t, "option", 1); optType != nil {
				elemType = ctx.MapType(optType[0])
			}
			return fmt.Sprintf("rt.Some[%s](%s)", elemType, args[0]), true
		},
		CCall: "vsome",
	})

	Register(&Builtin{
		Name:            "none",
		Category:        "option",
		Arity:           ExactArity(1),
		UnevaluatedArgs: true,
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			t := ctx.TypeOf(call)
			elemType := "interface{}"
			if optType := parts(t, "option", 1); optType != nil {
				elemType = ctx.MapType(optType[0])
			}
			return fmt.Sprintf("rt.None[%s]()", elemType), true
		},
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			return ctx.Value("voption_none()"), true
		},
	})

	// ---------------------------------------------------------
	// Result constructors: ok / err / unwrap-or
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:     "ok",
		Category: "result",
		Arity:    ExactArity(1),
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			t := ctx.TypeOf(call).(*ast.AppliedType)
			return fmt.Sprintf("rt.Ok[%s,%s](%s)", ctx.MapType(t.Args[0]), ctx.MapType(t.Args[1]), strings.Join(args, ", ")), true
		},
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			return ctx.Value("vr(1," + strings.Join(args, ",") + ")"), true
		},
	})

	Register(&Builtin{
		Name:     "err",
		Category: "result",
		Arity:    ExactArity(1),
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			t := ctx.TypeOf(call).(*ast.AppliedType)
			return fmt.Sprintf("rt.Err[%s,%s](%s)", ctx.MapType(t.Args[0]), ctx.MapType(t.Args[1]), strings.Join(args, ", ")), true
		},
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			return ctx.Value("vr(0," + strings.Join(args, ",") + ")"), true
		},
	})

	Register(&Builtin{
		Name:     "unwrap-or",
		Category: "result",
		Arity:    ExactArity(2),
		Params:   []string{"(result T E)", "T"},
		Return:   "T",
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			if len(args) != 2 {
				return "", false
			}
			t0 := ctx.TypeOf(call.Args[0])
			if t0 != nil && strings.HasPrefix(t0.String(), "(option ") {
				return fmt.Sprintf("rt.UnwrapOrOption(%s, %s)", args[0], args[1]), true
			}
			return fmt.Sprintf("rt.UnwrapOrResult(%s, %s)", args[0], args[1]), true
		},
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			if len(args) != 2 {
				return "", false
			}
			return ctx.Value(fmt.Sprintf("vunwrap_or(%s, %s)", args[0], args[1])), true
		},
	})

	// ---------------------------------------------------------
	// Struct operations: new / field
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:            "new",
		Category:        "object",
		Arity:           MinArity(1),
		UnevaluatedArgs: true,
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			tName := ctx.TypeName(call.Args[0])
			return tName + "{" + strings.Join(args[1:], ", ") + "}", true
		},
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			s := ctx.FindStruct(call.Args[0].(*ast.IdentExpr).Name)
			names, vargs := []string{}, []string{}
			for i, a := range call.Args[1:] {
				vargs = append(vargs, ctx.Expr(a))
				names = append(names, ctx.Quote(s.Fields[i].Name))
			}
			nms := "NULL"
			if len(names) > 0 {
				nms = "(const char*[]){" + strings.Join(names, ",") + "}"
			}
			return ctx.Value(fmt.Sprintf("vobject(%d,%s,%s)", len(vargs), nms, ctx.Array(vargs))), true
		},
	})

	Register(&Builtin{
		Name:            "field",
		Category:        "object",
		Arity:           ExactArity(2),
		UnevaluatedArgs: true,
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			fName := ctx.FieldIdent(call.Args[1].(*ast.IdentExpr).Name)
			return "(" + args[0] + ")." + fName, true
		},
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			v := ctx.Expr(call.Args[0])
			return ctx.Value("vfield(" + v + "," + ctx.Quote(call.Args[1].(*ast.IdentExpr).Name) + ")"), true
		},
	})
}
