package checker

import (
	"fmt"

	"liaf/pkg/ast"
)

func primitive(s string) ast.Type { return &ast.PrimitiveType{Name: s} }

func applied(s string, args ...ast.Type) ast.Type {
	return &ast.AppliedType{Constructor: s, Args: args}
}

func name(t ast.Type) string {
	if t == nil {
		return "?"
	}
	return t.String()
}

func parts(t ast.Type, kind string, n int) []ast.Type {
	if a, ok := t.(*ast.AppliedType); ok && a.Constructor == kind && len(a.Args) == n {
		return a.Args
	}
	return nil
}

func IsResult(t ast.Type) bool { return parts(t, "result", 2) != nil }

func (c *Checker) validType(t ast.Type, allowVoid bool) {
	if t == nil {
		return
	}
	switch v := t.(type) {
	case *ast.PrimitiveType:
		if v.Name == "void" && !allowVoid {
			c.error(t, "E_INVALID_TYPE", "void is only a return type")
		}
	case *ast.NamedType:
		if c.Info.Structs[v.Name] == nil && !opaqueTypes[v.Name] {
			c.error(t, "E_UNKNOWN_TYPE", v.Name)
		}
	case *ast.AppliedType:
		arity := map[string]int{"chan": 1, "list": 1, "map": 2, "result": 2, "option": 1}[v.Constructor]
		if arity == 0 || len(v.Args) != arity {
			c.error(t, "E_INVALID_TYPE", v.String())
			return
		}
		for _, a := range v.Args {
			c.validType(a, false)
		}
		if v.Constructor == "map" && !scalar(v.Args[0]) {
			c.error(t, "E_INVALID_MAP_KEY", "Map keys must be scalar")
		}
	}
}

func scalar(t ast.Type) bool {
	switch name(t) {
	case "int", "float", "str", "bool":
		return true
	}
	return false
}

func (c *Checker) require(n ast.Node, actual, want ast.Type) {
	if actual != nil && want != nil && name(actual) != name(want) {
		c.error(n, "E_TYPE_MISMATCH", fmt.Sprintf("Expected %s, received %s", name(want), name(actual)))
	}
}
