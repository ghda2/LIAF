package builtins

import (
	"fmt"
	"strings"

	"liaf/pkg/ast"
)

func init() {
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
}
