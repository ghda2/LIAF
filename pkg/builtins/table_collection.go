package builtins

import (
	"fmt"
	"strings"

	"liaf/pkg/ast"
)

func init() {
	// ---------------------------------------------------------
	// Collections Constructors
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:            "make-list",
		Category:        "collection",
		Arity:           ExactArity(1),
		UnevaluatedArgs: true,
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			tName := ctx.TypeName(call.Args[0])
			return fmt.Sprintf("rt.MakeList[%s]()", tName), true
		},
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			return ctx.Value("vlist()"), true
		},
	})

	Register(&Builtin{
		Name:            "make-map",
		Category:        "collection",
		Arity:           ExactArity(2),
		UnevaluatedArgs: true,
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			kName := ctx.TypeName(call.Args[0])
			vName := ctx.TypeName(call.Args[1])
			return fmt.Sprintf("make(map[%s]%s)", kName, vName), true
		},
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			return ctx.Value("vmap()"), true
		},
	})

	// ---------------------------------------------------------
	// Collection Operations: List & Map
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:       "list-len",
		Category:   "collection",
		Arity:      ExactArity(1),
		Return:     "int",
		GoTemplate: "int64(len((%s).Items))",
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			return ctx.Value("vi(((List*)" + args[0] + ".p)->n)"), true
		},
	})

	Register(&Builtin{
		Name:     "list-push",
		Category: "collection",
		Arity:    ExactArity(2),
		Return:   "void",
		GoCall:   "rt.ListPush",
		CCall:    "vpush",
	})

	Register(&Builtin{
		Name:     "list-get",
		Category: "collection",
		Arity:    ExactArity(2),
		GoCall:   "rt.ListGet",
		CCall:    "vget",
	})

	Register(&Builtin{
		Name:     "list-set",
		Category: "collection",
		Arity:    ExactArity(3),
		Return:   "(result bool str)",
		GoCall:   "rt.ListSet",
		CCall:    "vset",
	})

	Register(&Builtin{
		Name:       "map-len",
		Category:   "collection",
		Arity:      ExactArity(1),
		Return:     "int",
		GoTemplate: "int64(len(%s))",
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			return ctx.Value("vi(((Map*)" + args[0] + ".p)->n)"), true
		},
	})

	Register(&Builtin{
		Name:     "map-get",
		Category: "collection",
		Arity:    ExactArity(2),
		GoCall:   "rt.MapGet",
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			return ctx.Value("vmget(" + strings.Join(args, ",") + ",0)"), true
		},
	})

	Register(&Builtin{
		Name:     "map-set",
		Category: "collection",
		Arity:    ExactArity(3),
		Return:   "void",
		GoCall:   "rt.MapSet",
		CCall:    "vmset",
	})

	Register(&Builtin{
		Name:     "map-has",
		Category: "collection",
		Arity:    ExactArity(2),
		Return:   "bool",
		GoCall:   "rt.MapHas",
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			return ctx.Value("vmget(" + strings.Join(args, ",") + ",1)"), true
		},
	})

	Register(&Builtin{
		Name:     "list",
		Category: "collection",
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			t := ctx.TypeOf(call)
			elemType := "interface{}"
			if listType := parts(t, "list", 1); listType != nil {
				elemType = ctx.MapType(listType[0])
			}
			return fmt.Sprintf("&rt.List[%s]{Items: []%s{%s}}", elemType, elemType, strings.Join(args, ", ")), true
		},
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			if len(args) == 0 {
				return ctx.Value("vlist()"), true
			}
			return ctx.Value(fmt.Sprintf("vlist_literal(%d, (V[]){%s})", len(args), strings.Join(args, ", "))), true
		},
	})

	Register(&Builtin{
		Name:     "list-remove",
		Category: "collection",
		Arity:    ExactArity(2),
		GoCall:   "rt.ListRemove",
		CCall:    "vlistremove",
	})

	Register(&Builtin{
		Name:     "list-pop",
		Category: "collection",
		Arity:    ExactArity(1),
		GoCall:   "rt.ListPop",
		CCall:    "vlistpop",
	})

	Register(&Builtin{
		Name:     "list-sort",
		Category: "collection",
		Arity:    ExactArity(1),
		Return:   "void",
		GoCall:   "rt.ListSort",
		CCall:    "vlistsort",
	})

	Register(&Builtin{
		Name:     "list-contains",
		Category: "collection",
		Arity:    ExactArity(2),
		Return:   "bool",
		GoCall:   "rt.ListContains",
		CCall:    "vlistcontains",
	})

	Register(&Builtin{
		Name:     "map-delete",
		Category: "collection",
		Arity:    ExactArity(2),
		Return:   "void",
		GoCall:   "rt.MapDelete",
		CCall:    "vmdelete",
	})

	Register(&Builtin{
		Name:     "map-keys",
		Category: "collection",
		Arity:    ExactArity(1),
		GoCall:   "rt.MapKeys",
		CCall:    "vmkeys",
	})
}
