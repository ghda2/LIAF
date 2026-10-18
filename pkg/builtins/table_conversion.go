package builtins

import (
	"fmt"

	"liaf/pkg/ast"
)

func init() {
	// ---------------------------------------------------------
	// Primitive conversions
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:     "str-from-int",
		Category: "conversion",
		Arity:    ExactArity(1),
		Params:   []string{"int"},
		Return:   "str",
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			if len(args) > 0 {
				return fmt.Sprintf("_liaf_str_from_int(int64(%s))", args[0]), true
			}
			return `""`, true
		},
		CCall: "vintstr",
	})

	Register(&Builtin{
		Name:     "float-from-int",
		Category: "conversion",
		Arity:    ExactArity(1),
		Params:   []string{"int"},
		Return:   "float",
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			if len(args) > 0 {
				return fmt.Sprintf("_liaf_float_from_int(int64(%s))", args[0]), true
			}
			return "0.0", true
		},
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			if len(args) > 0 {
				return ctx.Value(fmt.Sprintf("vf((double)%s.i)", args[0])), true
			}
			return ctx.Value("vf(0.0)"), true
		},
	})

	Register(&Builtin{
		Name:     "int-from-str",
		Category: "conversion",
		Arity:    ExactArity(1),
		Params:   []string{"str"},
		Return:   "(result int str)",
		GoCall:   "rt.IntFromStr",
		CCall:    "vintparse",
	})

	Register(&Builtin{
		Name:      "int-from-float",
		Category:  "conversion",
		Arity:     ExactArity(1),
		Params:    []string{"float"},
		Return:    "int",
		GoCall:    "rt.IntFromFloat",
		CTemplate: "vi((int64_t)%s.f)",
	})

	Register(&Builtin{
		Name:     "str-from-float",
		Category: "conversion",
		Arity:    ExactArity(1),
		Params:   []string{"float"},
		Return:   "str",
		GoCall:   "rt.StrFromFloat",
		CCall:    "vstrfloat",
	})

	Register(&Builtin{
		Name:     "float-from-str",
		Category: "conversion",
		Arity:    ExactArity(1),
		Params:   []string{"str"},
		Return:   "(result float str)",
		GoCall:   "rt.FloatFromStr",
		CCall:    "vfloatparse",
	})

	Register(&Builtin{
		Name:     "str-from-bool",
		Category: "conversion",
		Arity:    ExactArity(1),
		Params:   []string{"bool"},
		Return:   "str",
		GoCall:   "rt.StrFromBool",
		CCall:    "vstrbool",
	})

	Register(&Builtin{
		Name:     "bool-from-str",
		Category: "conversion",
		Arity:    ExactArity(1),
		Params:   []string{"str"},
		Return:   "(result bool str)",
		GoCall:   "rt.BoolFromStr",
		CCall:    "vboolparse",
	})
}
