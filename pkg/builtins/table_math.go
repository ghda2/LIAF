package builtins

import (
	"fmt"

	"liaf/pkg/ast"
)

func init() {
	// ---------------------------------------------------------
	// Math & Arithmetic (#020)
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:     "mod",
		Category: "math",
		Arity:    ExactArity(2),
		Params:   []string{"int", "int"},
		Return:   "(result int str)",
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			return fmt.Sprintf("rt.ModInt(%s, %s)", args[0], args[1]), true
		},
		CCall: "vmod",
	})

	Register(&Builtin{
		Name:     "neg",
		Category: "math",
		Arity:    ExactArity(1),
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			if len(args) > 0 {
				t := ctx.TypeOf(call.Args[0])
				if t != nil && t.String() == "float" {
					return fmt.Sprintf("-(%s)", args[0]), true
				}
				return fmt.Sprintf("rt.NegInt(%s)", args[0]), true
			}
			return "0", true
		},
		CCall: "vneg",
	})

	Register(&Builtin{
		Name:     "abs",
		Category: "math",
		Arity:    ExactArity(1),
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			t := ctx.TypeOf(call.Args[0])
			if t != nil && t.String() == "float" {
				return fmt.Sprintf("rt.AbsFloat(%s)", args[0]), true
			}
			return fmt.Sprintf("rt.AbsInt(%s)", args[0]), true
		},
		CCall: "vabs",
	})

	Register(&Builtin{
		Name:     "min",
		Category: "math",
		Arity:    ExactArity(2),
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			t := ctx.TypeOf(call.Args[0])
			if t != nil && t.String() == "float" {
				return fmt.Sprintf("rt.MinFloat(%s, %s)", args[0], args[1]), true
			}
			return fmt.Sprintf("rt.MinInt(%s, %s)", args[0], args[1]), true
		},
		CCall: "vmin",
	})

	Register(&Builtin{
		Name:     "max",
		Category: "math",
		Arity:    ExactArity(2),
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			t := ctx.TypeOf(call.Args[0])
			if t != nil && t.String() == "float" {
				return fmt.Sprintf("rt.MaxFloat(%s, %s)", args[0], args[1]), true
			}
			return fmt.Sprintf("rt.MaxInt(%s, %s)", args[0], args[1]), true
		},
		CCall: "vmax",
	})

	Register(&Builtin{
		Name:     "pow",
		Category: "math",
		Arity:    ExactArity(2),
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			t := ctx.TypeOf(call.Args[0])
			if t != nil && t.String() == "float" {
				return fmt.Sprintf("rt.PowFloat(%s, %s)", args[0], args[1]), true
			}
			return fmt.Sprintf("rt.PowInt(%s, %s)", args[0], args[1]), true
		},
		CCall: "vpow",
	})

	Register(&Builtin{
		Name:     "sqrt",
		Category: "math",
		Arity:    ExactArity(1),
		Params:   []string{"float"},
		Return:   "float",
		GoCall:   "rt.Sqrt",
		CCall:    "vsqrt",
	})

	Register(&Builtin{
		Name:     "floor",
		Category: "math",
		Arity:    ExactArity(1),
		Params:   []string{"float"},
		Return:   "float",
		GoCall:   "rt.Floor",
		CCall:    "vfloor",
	})

	Register(&Builtin{
		Name:     "ceil",
		Category: "math",
		Arity:    ExactArity(1),
		Params:   []string{"float"},
		Return:   "float",
		GoCall:   "rt.Ceil",
		CCall:    "vceil",
	})

	Register(&Builtin{
		Name:     "round",
		Category: "math",
		Arity:    ExactArity(1),
		Params:   []string{"float"},
		Return:   "float",
		GoCall:   "rt.Round",
		CCall:    "vround",
	})
}
