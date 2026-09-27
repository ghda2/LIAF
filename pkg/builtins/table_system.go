package builtins

import (
	"fmt"

	"liaf/pkg/ast"
)

func init() {
	// ---------------------------------------------------------
	// System Builtins
	// ---------------------------------------------------------
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
		Name:     "sleep-ms",
		Category: "system",
		Arity:    ExactArity(1),
		Params:   []string{"int"},
		Effects:  []string{"clock"},
		Return:   "void",
		GoEmit: func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool) {
			if len(args) > 0 {
				return fmt.Sprintf("_liaf_sleep_ms(int64(%s))", args[0]), true
			}
			return "", true
		},
		CCall: "vsleep",
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
		Name:               "fs-rename",
		Category:           "fs",
		Arity:              ExactArity(2),
		Params:             []string{"str", "str"},
		Effects:            []string{"fs"},
		Return:             "(result void str)",
		GoCall:             "rt.Rename",
		UnsupportedC:       true,
		UnsupportedCReason: "fs-rename nao suportado no backend C",
	})

	Register(&Builtin{
		Name:               "fs-write-atomic",
		Category:           "fs",
		Arity:              ExactArity(2),
		Params:             []string{"str", "str"},
		Effects:            []string{"fs"},
		Return:             "(result void str)",
		GoCall:             "rt.WriteAtomic",
		UnsupportedC:       true,
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
}
