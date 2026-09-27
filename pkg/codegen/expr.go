package codegen

import (
	"fmt"
	"strings"

	"liaf/pkg/ast"
)

func (g *Generator) genExpr(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.IntLiteral:
		return e.Raw
	case *ast.FloatLiteral:
		return e.Raw
	case *ast.BoolLiteral:
		if e.Value {
			return "true"
		}
		return "false"
	case *ast.StringLiteral:
		return fmt.Sprintf("%q", e.Value)
	case *ast.IdentExpr:
		return sanitizeIdent(e.Name)
	case *ast.CallExpr:
		return g.genCall(e)
	case *ast.BinaryOpExpr:
		return g.genBinaryOp(e)
	case *ast.RecvExpr:
		return fmt.Sprintf("<-%s", g.genExpr(e.Channel))
	case *ast.TryExpr:
		return fmt.Sprintf("(%s).Value", g.genExpr(e.Expr))
	case *ast.IfExpr:
		retType := g.MapType(g.TypeOf(e))
		if retType == "" {
			retType = "interface{}"
		}
		return fmt.Sprintf("func() %s { if %s { return %s }; return %s }()", retType, g.genExpr(e.Condition), g.genExpr(e.Then), g.genExpr(e.Else))
	case *ast.MatchExpr:
		retType := g.MapType(g.TypeOf(e))
		if retType == "" {
			retType = "interface{}"
		}
		g.serial++
		if e.IsOption {
			id := fmt.Sprintf("_liaf_opt_%d", g.serial)
			okVar := sanitizeIdent(e.OKName)
			errVar := sanitizeIdent(e.ErrName)
			if okVar == "" || okVar == "_u_" {
				okVar = "_"
			}
			var okInit string
			if okVar != "_" {
				okInit = fmt.Sprintf("%s := %s.Value; _ = %s\n", okVar, id, okVar)
			}
			var errInit string
			if errVar != "" && errVar != "_" && errVar != "_u_" {
				errInit = fmt.Sprintf("%s := struct{}{}; _ = %s\n", errVar, errVar)
			}
			return fmt.Sprintf("func() %s {\n%s := %s\nif %s.Some {\n%sreturn %s\n}\n%sreturn %s\n}()",
				retType, id, g.genExpr(e.Value), id, okInit, g.genExpr(e.OK), errInit, g.genExpr(e.Err))
		}
		id := fmt.Sprintf("_liaf_result_%d", g.serial)
		okVar := sanitizeIdent(e.OKName)
		errVar := sanitizeIdent(e.ErrName)
		if okVar == "" || okVar == "_u_" {
			okVar = "_"
		}
		if errVar == "" || errVar == "_u_" {
			errVar = "_"
		}
		var okInit string
		if okVar != "_" {
			okInit = fmt.Sprintf("%s := %s.Value; _ = %s\n", okVar, id, okVar)
		}
		var errInit string
		if errVar != "_" {
			errInit = fmt.Sprintf("%s := %s.Error; _ = %s\n", errVar, id, errVar)
		}
		return fmt.Sprintf("func() %s {\n%s := %s\nif %s.OK {\n%sreturn %s\n}\n%sreturn %s\n}()",
			retType, id, g.genExpr(e.Value), id, okInit, g.genExpr(e.OK), errInit, g.genExpr(e.Err))
	default:
		return ""
	}
}

func (g *Generator) GenExpr(e ast.Expr) string   { return g.genExpr(e) }
func (g *Generator) MapType(t ast.Type) string    { return mapType(t) }
func (g *Generator) TypeOf(e ast.Expr) ast.Type   { return g.info.Types[e] }
func (g *Generator) FieldIdent(f string) string   { return fieldIdent(f) }
func (g *Generator) TypeName(e ast.Expr) string {
	if t, ok := ast.TypeFromExpr(e); ok {
		return mapType(t)
	}
	return "interface{}"
}

func (g *Generator) genCall(c *ast.CallExpr) string {
	if value, ok := g.libraryCall(c); ok {
		return value
	}
	var args []string
	for _, a := range c.Args {
		args = append(args, g.genExpr(a))
	}
	return fmt.Sprintf("%s(%s)", sanitizeIdent(c.Func), strings.Join(args, ", "))
}

func (g *Generator) genBinaryOp(b *ast.BinaryOpExpr) string {
	left := g.genExpr(b.Left)
	right := g.genExpr(b.Right)

	isFloat := false
	if t := g.TypeOf(b.Left); t != nil && t.String() == "float" {
		isFloat = true
	} else if t := g.TypeOf(b.Right); t != nil && t.String() == "float" {
		isFloat = true
	}

	switch b.Op {
	case "add":
		if isFloat {
			return fmt.Sprintf("(%s + %s)", left, right)
		}
		return fmt.Sprintf("rt.AddInt(%s, %s)", left, right)
	case "sub":
		if isFloat {
			return fmt.Sprintf("(%s - %s)", left, right)
		}
		return fmt.Sprintf("rt.SubInt(%s, %s)", left, right)
	case "mul":
		if isFloat {
			return fmt.Sprintf("(%s * %s)", left, right)
		}
		return fmt.Sprintf("rt.MulInt(%s, %s)", left, right)
	case "div":
		if isFloat {
			return fmt.Sprintf("(%s / %s)", left, right)
		}
		return fmt.Sprintf("rt.DivInt(%s, %s)", left, right)
	case "eq":
		return fmt.Sprintf("(%s == %s)", left, right)
	case "neq":
		return fmt.Sprintf("(%s != %s)", left, right)
	case "gt":
		return fmt.Sprintf("(%s > %s)", left, right)
	case "lt":
		return fmt.Sprintf("(%s < %s)", left, right)
	case "gte":
		return fmt.Sprintf("(%s >= %s)", left, right)
	case "lte":
		return fmt.Sprintf("(%s <= %s)", left, right)
	case "and":
		return fmt.Sprintf("(%s && %s)", left, right)
	case "or":
		return fmt.Sprintf("(%s || %s)", left, right)
	default:
		return fmt.Sprintf("%s(%s, %s)", sanitizeIdent(b.Op), left, right)
	}
}
