package builtins

import (
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
