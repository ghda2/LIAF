package builtins

import (
	"fmt"
	"strings"

	"liaf/pkg/ast"
)

func init() {
	// ---------------------------------------------------------
	// Core String Operations
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:       "concat",
		Category:   "string",
		Arity:      MinArity(0),
		Return:     "str",
		GoTemplate: "_liaf_concat(%s)",
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			return ctx.Value(fmt.Sprintf("vconcat(%d,%s)", len(args), ctx.Array(args))), true
		},
	})

	Register(&Builtin{
		Name:       "fmt",
		Category:   "string",
		Arity:      MinArity(1),
		Return:     "str",
		GoTemplate: "_liaf_fmt(%s)",
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			if len(args) == 0 {
				return ctx.Value(`vs("")`), true
			}
			pattern := args[0]
			rest := args[1:]
			return ctx.Value(fmt.Sprintf("vfmt(%s,%d,%s)", pattern, len(rest), ctx.Array(rest))), true
		},
	})

	// ---------------------------------------------------------
	// Strings
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:     "str-len",
		Category: "string",
		Arity:    ExactArity(1),
		Params:   []string{"str"},
		Return:   "int",
		GoCall:   "rt.StrLen",
		CCall:    "vstrlen",
	})

	Register(&Builtin{
		Name:     "str-byte-len",
		Category: "string",
		Arity:    ExactArity(1),
		Params:   []string{"str"},
		Return:   "int",
		GoCall:   "rt.StrByteLen",
		CCall:    "vstrbytelen",
	})

	Register(&Builtin{
		Name:     "str-slice",
		Category: "string",
		Arity:    ExactArity(3),
		Params:   []string{"str", "int", "int"},
		Return:   "(result str str)",
		GoCall:   "rt.StrSlice",
		CCall:    "vslice",
	})

	Register(&Builtin{
		Name:       "str-eq",
		Category:   "string",
		Arity:      ExactArity(2),
		Params:     []string{"str", "str"},
		Return:     "bool",
		GoTemplate: "(%s == %s)",
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			return ctx.Value("vb(eq(" + strings.Join(args, ",") + "))"), true
		},
	})

	Register(&Builtin{
		Name:     "str-contains",
		Category: "string",
		Arity:    ExactArity(2),
		Params:   []string{"str", "str"},
		Return:   "bool",
		GoCall:   "rt.StrContains",
		CCall:    "vstrcontains",
	})

	Register(&Builtin{
		Name:     "str-starts-with",
		Category: "string",
		Arity:    ExactArity(2),
		Params:   []string{"str", "str"},
		Return:   "bool",
		GoCall:   "rt.StrStartsWith",
		CCall:    "vstrstartswith",
	})

	Register(&Builtin{
		Name:     "str-ends-with",
		Category: "string",
		Arity:    ExactArity(2),
		Params:   []string{"str", "str"},
		Return:   "bool",
		GoCall:   "rt.StrEndsWith",
		CCall:    "vstrendswith",
	})

	Register(&Builtin{
		Name:     "str-index",
		Category: "string",
		Arity:    ExactArity(2),
		Params:   []string{"str", "str"},
		Return:   "int",
		GoCall:   "rt.StrIndex",
		CCall:    "vstrindex",
	})

	Register(&Builtin{
		Name:     "str-split",
		Category: "string",
		Arity:    ExactArity(2),
		Params:   []string{"str", "str"},
		Return:   "(list str)",
		GoCall:   "rt.StrSplit",
		CCall:    "vstrsplit",
	})

	Register(&Builtin{
		Name:     "str-join",
		Category: "string",
		Arity:    ExactArity(2),
		Params:   []string{"(list str)", "str"},
		Return:   "str",
		GoCall:   "rt.StrJoin",
		CCall:    "vstrjoin",
	})

	Register(&Builtin{
		Name:     "str-trim",
		Category: "string",
		Arity:    ExactArity(1),
		Params:   []string{"str"},
		Return:   "str",
		GoCall:   "rt.StrTrim",
		CCall:    "vstrtrim",
	})

	Register(&Builtin{
		Name:     "str-upper",
		Category: "string",
		Arity:    ExactArity(1),
		Params:   []string{"str"},
		Return:   "str",
		GoCall:   "rt.StrUpper",
		CCall:    "vstrupper",
	})

	Register(&Builtin{
		Name:     "str-lower",
		Category: "string",
		Arity:    ExactArity(1),
		Params:   []string{"str"},
		Return:   "str",
		GoCall:   "rt.StrLower",
		CCall:    "vstrlower",
	})

	Register(&Builtin{
		Name:     "str-replace",
		Category: "string",
		Arity:    ExactArity(3),
		Params:   []string{"str", "str", "str"},
		Return:   "str",
		GoCall:   "rt.StrReplace",
		CCall:    "vstrreplace",
	})

	Register(&Builtin{
		Name:     "str-get",
		Category: "string",
		Arity:    ExactArity(2),
		Params:   []string{"str", "int"},
		Return:   "(result str str)",
		GoCall:   "rt.StrGet",
		CCall:    "vstrget",
	})
}
