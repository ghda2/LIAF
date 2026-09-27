package builtins

import (
	"fmt"
	"strings"

	"liaf/pkg/ast"
)

// Hook functions that can be registered by packages if they require checker/compiler internals.
var specialChecks = map[string]func(ctx CheckContext, call *ast.CallExpr, want ast.Type) ast.Type{}

// RegisterSpecialCheck registra um gancho de checagem customizado para um builtin especial.
func RegisterSpecialCheck(name string, fn func(ctx CheckContext, call *ast.CallExpr, want ast.Type) ast.Type) {
	n := normalize(name)
	specialChecks[n] = fn
	if b, ok := registry[n]; ok {
		b.SpecialCheck = fn
	}
}

func parts(t ast.Type, kind string, n int) []ast.Type {
	if a, ok := t.(*ast.AppliedType); ok && a.Constructor == kind && len(a.Args) == n {
		return a.Args
	}
	return nil
}

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
	// ---------------------------------------------------------
	// Core / IO / System
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:     "println",
		Category: "io",
		Arity:    MinArity(0),
		Effects:  []string{"io"},
		Return:   "void",
		GoTemplate: "fmt.Println(%s)",
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			return ctx.Value(fmt.Sprintf("vprint(%d,%s,1)", len(args), ctx.Array(args))), true
		},
	})

	Register(&Builtin{
		Name:     "print",
		Category: "io",
		Arity:    MinArity(0),
		Effects:  []string{"io"},
		Return:   "void",
		GoTemplate: "fmt.Print(%s)",
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			return ctx.Value(fmt.Sprintf("vprint(%d,%s,0)", len(args), ctx.Array(args))), true
		},
	})

	Register(&Builtin{
		Name:     "concat",
		Category: "string",
		Arity:    MinArity(0),
		Return:   "str",
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

	Register(&Builtin{
		Name:       "not",
		Category:   "core",
		Arity:      ExactArity(1),
		Params:     []string{"bool"},
		Return:     "bool",
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

	Register(&Builtin{
		Name:       "args",
		Category:   "system",
		Arity:      ExactArity(0),
		Effects:    []string{"io"},
		Return:     "(list str)",
		GoCall:     "rt.Args",
		CEmit: func(ctx CContext, call *ast.CallExpr, args []string) (string, bool) {
			return ctx.Value("program_args"), true
		},
	})

	Register(&Builtin{
		Name:       "sleep-ms",
		Category:   "system",
		Arity:      ExactArity(1),
		Params:     []string{"int"},
		Effects:    []string{"clock"},
		Return:     "void",
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			if len(args) > 0 {
				return fmt.Sprintf("_liaf_sleep_ms(int64(%s))", args[0]), true
			}
			return "", true
		},
		CCall:      "vsleep",
	})

	Register(&Builtin{
		Name:         "now-ms",
		Category:     "system",
		Arity:        ExactArity(0),
		Effects:      []string{"clock"},
		Return:       "int",
		GoCall:       "rt.NowMs",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "rand-int",
		Category:     "system",
		Arity:        ExactArity(1),
		Effects:      []string{"rand"},
		Params:       []string{"int"},
		Return:       "int",
		GoCall:       "rt.RandInt",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "env-get",
		Category:     "system",
		Arity:        ExactArity(1),
		Effects:      []string{"env"},
		Params:       []string{"str"},
		Return:       "(result str str)",
		GoCall:       "rt.EnvGet",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "read-line",
		Category:     "system",
		Arity:        ExactArity(0),
		Effects:      []string{"io"},
		Return:       "(result str str)",
		GoCall:       "rt.ReadLine",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "exit",
		Category:     "system",
		Arity:        ExactArity(1),
		Effects:      []string{"io"},
		Params:       []string{"int"},
		Return:       "void",
		GoCall:       "rt.Exit",
		UnsupportedC: true,
	})

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

	// ---------------------------------------------------------
	// File System
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:     "fs-read-file",
		Category: "fs",
		Arity:    ExactArity(1),
		Params:   []string{"str"},
		Effects:  []string{"fs"},
		Return:   "(result str str)",
		GoCall:   "rt.ReadFile",
		CCall:    "vread",
	})

	Register(&Builtin{
		Name:     "fs-write-file",
		Category: "fs",
		Arity:    ExactArity(2),
		Params:   []string{"str", "str"},
		Effects:  []string{"fs"},
		Return:   "(result void str)",
		GoCall:   "rt.WriteFile",
		CCall:    "vwrite",
	})

	Register(&Builtin{
		Name:         "fs-rename",
		Category:     "fs",
		Arity:        ExactArity(2),
		Params:       []string{"str", "str"},
		Effects:      []string{"fs"},
		Return:       "(result void str)",
		GoCall:       "rt.Rename",
		UnsupportedC: true,
		UnsupportedCReason: "fs-rename nao suportado no backend C",
	})

	Register(&Builtin{
		Name:         "fs-write-atomic",
		Category:     "fs",
		Arity:        ExactArity(2),
		Params:       []string{"str", "str"},
		Effects:      []string{"fs"},
		Return:       "(result void str)",
		GoCall:       "rt.WriteAtomic",
		UnsupportedC: true,
		UnsupportedCReason: "fs-write-atomic nao suportado no backend C",
	})

	Register(&Builtin{
		Name:     "fs-remove",
		Category: "fs",
		Arity:    ExactArity(1),
		Params:   []string{"str"},
		Effects:  []string{"fs"},
		Return:   "(result void str)",
		GoCall:   "rt.Remove",
		CCall:    "vremove",
	})

	Register(&Builtin{
		Name:     "fs-exists",
		Category: "fs",
		Arity:    ExactArity(1),
		Params:   []string{"str"},
		Effects:  []string{"fs"},
		Return:   "bool",
		GoCall:   "rt.Exists",
		CCall:    "vexists",
	})

	// ---------------------------------------------------------
	// JSON
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:         "json-encode",
		Category:     "json",
		Arity:        ExactArity(1),
		Return:       "(result str str)",
		GoCall:       "rt.JSONEncode",
		UnsupportedC: true,
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
	// Collections & Channels Constructors
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
	// Result constructors: ok / err
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

	// ---------------------------------------------------------
	// Web / HTTP
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:         "json-response",
		Category:     "web",
		Arity:        ExactArity(2),
		Params:       []string{"int", "str"},
		Return:       "Response",
		GoCall:       "web.JSONResponse",
		UnsupportedC: true,
	})

	for _, m := range []string{"http-get", "http-post", "http-put", "http-delete"} {
		method := strings.ToUpper(strings.TrimPrefix(m, "http-"))
		Register(&Builtin{
			Name:         m,
			Category:     "web",
			Arity:        ExactArity(2),
			Effects:      []string{"net"},
			Return:       "void",
			GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
				return fmt.Sprintf("web.Register(%q, %s)", method, strings.Join(args, ", ")), true
			},
			UnsupportedC: true,
		})
	}

	Register(&Builtin{
		Name:         "serve-site",
		Category:     "web",
		Arity:        ExactArity(4),
		Params:       []string{"str", "str", "str", "bool"},
		Effects:      []string{"net", "fs"},
		Return:       "void",
		GoCall:       "web.ServeSite",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "serve-hybrid",
		Category:     "web",
		Arity:        ExactArity(2),
		Params:       []string{"str", "str"},
		Effects:      []string{"net", "fs"},
		Return:       "(result bool str)",
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			return "rt.Failure(web.ServeHybrid(" + strings.Join(args, ", ") + "))", true
		},
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "request-body",
		Category:     "web",
		Arity:        ExactArity(1),
		Params:       []string{"Request"},
		Return:       "str",
		GoTemplate:   "(%s).Body",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "request-path",
		Category:     "web",
		Arity:        ExactArity(1),
		Params:       []string{"Request"},
		Return:       "str",
		GoTemplate:   "(%s).Path",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "request-method",
		Category:     "web",
		Arity:        ExactArity(1),
		Params:       []string{"Request"},
		Return:       "str",
		GoTemplate:   "(%s).Method",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "request-header",
		Category:     "web",
		Arity:        ExactArity(2),
		Params:       []string{"Request", "str"},
		Return:       "(result str str)",
		GoCall:       "rt.RequestHeader",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "request-query",
		Category:     "web",
		Arity:        ExactArity(2),
		Params:       []string{"Request", "str"},
		Return:       "(result str str)",
		GoCall:       "rt.RequestQuery",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "response-set-header",
		Category:     "web",
		Arity:        ExactArity(3),
		Params:       []string{"Response", "str", "str"},
		Return:       "Response",
		GoCall:       "rt.ResponseSetHeader",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "raw-response",
		Category:     "web",
		Arity:        ExactArity(3),
		Params:       []string{"int", "str", "str"},
		Return:       "Response",
		GoCall:       "web.RawResponse",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "file-response",
		Category:     "web",
		Arity:        ExactArity(2),
		Params:       []string{"str", "str"},
		Effects:      []string{"fs"},
		Return:       "(result Response str)",
		GoCall:       "rt.FileResponse",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "request-file-data",
		Category:     "web",
		Arity:        ExactArity(2),
		Params:       []string{"Request", "str"},
		Return:       "(result str str)",
		GoCall:       "rt.RequestFileData",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "request-file-name",
		Category:     "web",
		Arity:        ExactArity(2),
		Params:       []string{"Request", "str"},
		Return:       "(result str str)",
		GoCall:       "rt.RequestFileName",
		UnsupportedC: true,
	})

	// ---------------------------------------------------------
	// Imagem e Compressão WebP
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:         "image-to-webp",
		Category:     "image",
		Arity:        ExactArity(1),
		Params:       []string{"str"},
		Return:       "(result str str)",
		GoCall:       "rt.ImageToWebP",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "image-to-webp-quality",
		Category:     "image",
		Arity:        ExactArity(2),
		Params:       []string{"str", "int"},
		Return:       "(result str str)",
		GoCall:       "rt.ImageToWebPQuality",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "image-dimensions",
		Category:     "image",
		Arity:        ExactArity(1),
		Params:       []string{"str"},
		Return:       "(result str str)",
		GoCall:       "rt.ImageDimensions",
		UnsupportedC: true,
	})

	// ---------------------------------------------------------
	// Storage (Estilo MinIO)
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:         "storage-dir-init",
		Category:     "storage",
		Arity:        ExactArity(1),
		Params:       []string{"str"},
		Effects:      []string{"fs"},
		Return:       "(result bool str)",
		GoCall:       "rt.StorageInit",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "storage-save-image",
		Category:     "storage",
		Arity:        ExactArity(3),
		Params:       []string{"str", "str", "str"},
		Effects:      []string{"fs"},
		Return:       "(result str str)",
		GoCall:       "rt.StorageSaveImage",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "storage-file-get",
		Category:     "storage",
		Arity:        ExactArity(2),
		Params:       []string{"str", "str"},
		Effects:      []string{"fs"},
		Return:       "(result str str)",
		GoCall:       "rt.StorageGet",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "storage-file-delete",
		Category:     "storage",
		Arity:        ExactArity(2),
		Params:       []string{"str", "str"},
		Effects:      []string{"fs"},
		Return:       "(result bool str)",
		GoCall:       "rt.StorageDelete",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "storage-file-exists",
		Category:     "storage",
		Arity:        ExactArity(2),
		Params:       []string{"str", "str"},
		Effects:      []string{"fs"},
		Return:       "bool",
		GoCall:       "rt.StorageExists",
		UnsupportedC: true,
	})

	// ---------------------------------------------------------
	// Cliente HTTP
	// ---------------------------------------------------------
	// http-get e afins ja sao a forma v0.2 de registrar rotas, dai o nome
	// http-fetch. O metodo literal e conferido pelo checker.
	Register(&Builtin{
		Name:         "http-fetch",
		Category:     "http-client",
		Arity:        ExactArity(4),
		Params:       []string{"str", "str", "(list str)", "str"},
		Effects:      []string{"net"},
		Return:       "(result HttpReply str)",
		GoCall:       "rt.HTTPFetch",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "reply-status",
		Category:     "http-client",
		Arity:        ExactArity(1),
		Params:       []string{"HttpReply"},
		Return:       "int",
		GoCall:       "rt.ReplyStatus",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "reply-body",
		Category:     "http-client",
		Arity:        ExactArity(1),
		Params:       []string{"HttpReply"},
		Return:       "str",
		GoCall:       "rt.ReplyBody",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "reply-header",
		Category:     "http-client",
		Arity:        ExactArity(2),
		Params:       []string{"HttpReply", "str"},
		Return:       "(result str str)",
		GoCall:       "rt.ReplyHeader",
		UnsupportedC: true,
	})

	// ---------------------------------------------------------
	// Crypto
	// ---------------------------------------------------------
	// sha256 e hmac-sha256 recebem a codificacao como literal ("hex" ou
	// "base64url"), conferido pelo checker como o driver de db-connect.
	Register(&Builtin{
		Name:         "sha256",
		Category:     "crypto",
		Arity:        ExactArity(2),
		Params:       []string{"str", "str"},
		Return:       "str",
		GoCall:       "rt.SHA256",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "hmac-sha256",
		Category:     "crypto",
		Arity:        ExactArity(3),
		Params:       []string{"str", "str", "str"},
		Return:       "str",
		GoCall:       "rt.HMACSHA256",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "secure-eq",
		Category:     "crypto",
		Arity:        ExactArity(2),
		Params:       []string{"str", "str"},
		Return:       "bool",
		GoCall:       "rt.SecureEq",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "base64url-encode",
		Category:     "crypto",
		Arity:        ExactArity(1),
		Params:       []string{"str"},
		Return:       "str",
		GoCall:       "rt.Base64URLEncode",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "base64url-decode",
		Category:     "crypto",
		Arity:        ExactArity(1),
		Params:       []string{"str"},
		Return:       "(result str str)",
		GoCall:       "rt.Base64URLDecode",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "random-token",
		Category:     "crypto",
		Arity:        ExactArity(0),
		Effects:      []string{"rand"},
		Return:       "str",
		GoCall:       "rt.RandomToken",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "password-hash",
		Category:     "crypto",
		Arity:        ExactArity(1),
		Params:       []string{"str"},
		Effects:      []string{"rand"},
		Return:       "str",
		GoCall:       "rt.PasswordHash",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "password-verify",
		Category:     "crypto",
		Arity:        ExactArity(2),
		Params:       []string{"str", "str"},
		Return:       "bool",
		GoCall:       "rt.PasswordVerify",
		UnsupportedC: true,
	})

	// ---------------------------------------------------------
	// Database (issue #015)
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:         "db-connect",
		Category:     "db",
		Arity:        ExactArity(2),
		Effects:      []string{"db"},
		Return:       "(result DBConnection str)",
		GoCall:       "rt.DBConnect",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "db-close",
		Category:     "db",
		Arity:        ExactArity(1),
		Effects:      []string{"db"},
		Return:       "(result void str)",
		GoCall:       "rt.DBClose",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:            "db-query",
		Category:        "db",
		Arity:           MinArity(3),
		Effects:         []string{"db"},
		UnevaluatedArgs: true,
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			tName := ctx.TypeName(call.Args[2])
			params := append([]string{args[0], args[1]}, args[3:]...)
			return fmt.Sprintf("rt.DBQuery[%s](%s)", tName, strings.Join(params, ", ")), true
		},
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "db-exec",
		Category:     "db",
		Arity:        MinArity(2),
		Effects:      []string{"db"},
		GoCall:       "rt.DBExec",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "redis-get",
		Category:     "db",
		Arity:        ExactArity(2),
		Effects:      []string{"db"},
		Return:       "(result str str)",
		GoCall:       "rt.RedisGet",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "redis-set",
		Category:     "db",
		Arity:        ExactArity(4),
		Effects:      []string{"db"},
		Return:       "(result void str)",
		GoCall:       "rt.RedisSet",
		UnsupportedC: true,
	})

	// ---------------------------------------------------------
	// WebSockets (issue #016)
	// ---------------------------------------------------------
	Register(&Builtin{
		Name:         "ws-send",
		Category:     "ws",
		Arity:        ExactArity(2),
		Effects:      []string{"net"},
		Return:       "(result void str)",
		GoCall:       "rt.WSSend",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "ws-send-json",
		Category:     "ws",
		Arity:        ExactArity(2),
		Effects:      []string{"net"},
		Return:       "(result void str)",
		GoCall:       "rt.WSSendJSON",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "ws-close",
		Category:     "ws",
		Arity:        ExactArity(3),
		Effects:      []string{"net"},
		Return:       "(result void str)",
		GoCall:       "rt.WSCloseConn",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "ws-broadcast",
		Category:     "ws",
		Arity:        ExactArity(2),
		Effects:      []string{"net"},
		Return:       "(result int str)",
		GoCall:       "rt.WSBroadcast",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "ws-join",
		Category:     "ws",
		Arity:        ExactArity(2),
		Effects:      []string{"net"},
		Return:       "void",
		GoCall:       "rt.WSJoin",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "ws-leave",
		Category:     "ws",
		Arity:        ExactArity(2),
		Effects:      []string{"net"},
		Return:       "void",
		GoCall:       "rt.WSLeave",
		UnsupportedC: true,
	})

	Register(&Builtin{
		Name:         "ws-topic-size",
		Category:     "ws",
		Arity:        ExactArity(1),
		Effects:      []string{"net"},
		Return:       "int",
		GoCall:       "rt.WSTopicSize",
		UnsupportedC: true,
	})
}
