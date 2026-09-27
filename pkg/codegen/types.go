package codegen

import (
	"fmt"
	"strings"

	"liaf/pkg/ast"
)

func sanitizeIdent(id string) string {
	id = strings.ReplaceAll(id, "_", "_u_")
	id = strings.ReplaceAll(id, "-", "_h_")
	switch id {
	case "break", "case", "chan", "const", "continue", "default", "defer", "else", "fallthrough", "for", "func", "go", "goto", "if", "import", "interface", "map", "package", "range", "return", "select", "struct", "switch", "type", "var", "fmt", "rt", "web", "time", "strings", "strconv", "true", "false", "nil":
		return "_liaf_user_" + id
	}
	return id
}

func fieldIdent(id string) string { return "Field_" + sanitizeIdent(id) }

// opaqueGoTypes traduz os tipos que a LIAF reconhece sem struct declarada.
// A lista espelha checker.opaqueTypes; divergir faria o checker aceitar um
// tipo que o codegen nao sabe escrever.
var opaqueGoTypes = map[string]string{
	"Request":      "web.Request",
	"Response":     "web.Response",
	"DBConnection": "*rt.DBConn",
	"WSConn":       "*web.WSConn",
	"HttpReply":    "*rt.HttpReply",
}

func mapType(t ast.Type) string {
	if t == nil {
		return ""
	}
	switch ty := t.(type) {
	case *ast.PrimitiveType:
		return mapTypeName(ty.Name)
	case *ast.NamedType:
		if goType, ok := opaqueGoTypes[ty.Name]; ok {
			return goType
		}
		return sanitizeIdent(ty.Name)
	case *ast.AppliedType:
		switch ty.Constructor {
		case "chan":
			if len(ty.Args) > 0 {
				return fmt.Sprintf("chan %s", mapType(ty.Args[0]))
			}
			return "chan interface{}"
		case "list":
			if len(ty.Args) > 0 {
				return fmt.Sprintf("*rt.List[%s]", mapType(ty.Args[0]))
			}
			return "[]interface{}"
		case "map":
			if len(ty.Args) >= 2 {
				return fmt.Sprintf("map[%s]%s", mapType(ty.Args[0]), mapType(ty.Args[1]))
			}
			return "map[string]interface{}"
		case "result":
			return fmt.Sprintf("rt.Result[%s,%s]", mapType(ty.Args[0]), mapType(ty.Args[1]))
		case "option":
			if len(ty.Args) > 0 {
				return fmt.Sprintf("rt.Option[%s]", mapType(ty.Args[0]))
			}
			return "rt.Option[interface{}]"
		default:
			return sanitizeIdent(ty.Constructor)
		}
	default:
		return "interface{}"
	}
}

func mapTypeName(name string) string {
	switch name {
	case "int":
		return "int64"
	case "float":
		return "float64"
	case "str":
		return "string"
	case "bool":
		return "bool"
	case "void":
		return ""
	default:
		return sanitizeIdent(name)
	}
}
