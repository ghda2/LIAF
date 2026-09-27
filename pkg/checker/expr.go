package checker

import (
	"fmt"
	"strings"

	"liaf/pkg/ast"
	"liaf/pkg/builtins"
)

func (c *Checker) expr(e ast.Expr, want ast.Type) (t ast.Type) {
	if e == nil {
		return primitive("void")
	}
	defer func() { c.Info.Types[e] = t }()
	switch v := e.(type) {
	case *ast.IntLiteral:
		return primitive("int")
	case *ast.FloatLiteral:
		return primitive("float")
	case *ast.StringLiteral:
		return primitive("str")
	case *ast.BoolLiteral:
		return primitive("bool")
	case *ast.IdentExpr:
		if b := c.lookup(v.Name); b != nil {
			return b.typ
		}
		c.error(e, "E_UNDEFINED_SYMBOL", v.Name)
	case *ast.RecvExpr:
		a := parts(c.expr(v.Channel, nil), "chan", 1)
		if a != nil {
			return a[0]
		}
		c.error(e, "E_CHANNEL_TYPE_MISMATCH", "recv requires channel")
	case *ast.BinaryOpExpr:
		l, r := c.expr(v.Left, nil), c.expr(v.Right, nil)
		c.require(e, r, l)
		switch v.Op {
		case "and", "or":
			c.require(e, l, primitive("bool"))
			return primitive("bool")
		case "eq", "neq":
			if l != nil && !scalar(l) {
				c.error(e, "E_TYPE_MISMATCH", "Equality requires scalar operands")
			}
			return primitive("bool")
		default:
			if name(l) != "int" && name(l) != "float" && l != nil {
				isStrComp := name(l) == "str" && (v.Op == "gt" || v.Op == "lt" || v.Op == "gte" || v.Op == "lte")
				if !isStrComp {
					c.error(e, "E_TYPE_MISMATCH", "Numeric operands required")
				}
			}
		}
		if v.Op == "gt" || v.Op == "lt" || v.Op == "gte" || v.Op == "lte" {
			return primitive("bool")
		}
		if v.Op == "div" && name(l) == "int" {
			return applied("result", primitive("int"), primitive("str"))
		}
		return l
	case *ast.CallExpr:
		return c.call(v, want)
	case *ast.TryExpr:
		subType := c.expr(v.Expr, nil)
		a := parts(subType, "result", 2)
		if a == nil {
			c.error(e, "E_TYPE_MISMATCH", "try requires result operand")
			return primitive("void")
		}
		if c.fn != nil {
			if c.fn.OnErrVar == "" && !IsResult(c.fn.ReturnType) {
				c.error(e, "E_UNHANDLED_RESULT", "try requires either an (on-err ...) block or a function returning (result ...)")
			}
		}
		return a[0]
	case *ast.IfExpr:
		c.require(v, c.expr(v.Condition, nil), primitive("bool"))
		if v.Else == nil {
			c.error(v, "E_IF_EXPR_MISSING_ELSE", "if used as expression requires else branch")
			if v.Then != nil {
				return c.expr(v.Then, want)
			}
			return primitive("void")
		}
		thenType := c.expr(v.Then, want)
		elseType := c.expr(v.Else, want)
		if thenType != nil && elseType != nil && name(thenType) != name(elseType) {
			c.error(v, "E_IF_EXPR_BRANCH_TYPE", fmt.Sprintf("if branches have different types: then is %s, else is %s", name(thenType), name(elseType)))
			return thenType
		}
		return thenType
	case *ast.MatchExpr:
		valType := c.expr(v.Value, nil)
		if v.IsOption {
			a := parts(valType, "option", 1)
			if a == nil {
				c.error(v, "E_TYPE_MISMATCH", "match requires option")
				return primitive("void")
			}
			c.consume(v.Value)
			c.push()
			if v.OKName != "" && v.OKName != "_" {
				c.define(v.OKName, a[0], v, false)
			}
			okType := c.expr(v.OK, want)
			c.pop()

			c.push()
			if v.ErrName != "" && v.ErrName != "_" {
				c.define(v.ErrName, primitive("void"), v, false)
			}
			errType := c.expr(v.Err, want)
			c.pop()

			if want != nil {
				c.require(v.OK, okType, want)
				c.require(v.Err, errType, want)
				return want
			}
			if okType != nil && errType != nil && name(okType) != name(errType) {
				c.error(v, "E_TYPE_MISMATCH", fmt.Sprintf("match branches have different types: %s and %s", name(okType), name(errType)))
				return okType
			}
			if okType != nil {
				return okType
			}
			return primitive("void")
		}
		a := parts(valType, "result", 2)
		if a == nil {
			c.error(v, "E_TYPE_MISMATCH", "match requires result")
			return primitive("void")
		}
		c.consume(v.Value)
		c.push()
		if v.OKName != "" && v.OKName != "_" {
			c.define(v.OKName, a[0], v, false)
		}
		okType := c.expr(v.OK, want)
		c.pop()

		c.push()
		if v.ErrName != "" && v.ErrName != "_" {
			c.define(v.ErrName, a[1], v, false)
		}
		errType := c.expr(v.Err, want)
		c.pop()

		if want != nil {
			c.require(v.OK, okType, want)
			c.require(v.Err, errType, want)
			return want
		}
		if okType != nil && errType != nil && name(okType) != name(errType) {
			c.error(v, "E_TYPE_MISMATCH", fmt.Sprintf("match branches have different types: %s and %s", name(okType), name(errType)))
			return okType
		}
		if okType != nil {
			return okType
		}
		return primitive("void")
	}
	return nil
}

func (c *Checker) arity(v *ast.CallExpr, n int) bool {
	if len(v.Args) != n {
		c.error(v, "E_WRONG_ARITY", fmt.Sprintf("%s expects %d arguments", v.Func, n))
		return false
	}
	return true
}

func (c *Checker) typeArg(e ast.Expr) ast.Type {
	t, ok := ast.TypeFromExpr(e)
	if !ok {
		c.error(e, "E_INVALID_TYPE", "Expected type name")
		return primitive("int")
	}
	c.validType(t, false)
	return t
}

func (c *Checker) call(v *ast.CallExpr, want ast.Type) ast.Type {
	n := strings.ReplaceAll(v.Func, "_", "-")
	b := builtins.Lookup(n)
	if b != nil {
		for _, eff := range b.Effects {
			c.effect(v, eff)
		}
		if b.SpecialCheck != nil {
			return b.SpecialCheck(c, v, want)
		}
		args := make([]ast.Type, len(v.Args))
		for i, a := range v.Args {
			args[i] = c.expr(a, nil)
		}
		if b.Arity.Max != -1 {
			if !c.arity(v, b.Arity.Min) {
				return nil
			}
		} else if len(v.Args) < b.Arity.Min {
			c.error(v, "E_WRONG_ARITY", fmt.Sprintf("%s expects at least %d arguments", v.Func, b.Arity.Min))
			return nil
		}
		for i, paramSpec := range b.Params {
			if i < len(args) {
				c.require(v, args[i], builtins.ParseType(paramSpec))
			}
		}
		return builtins.ParseType(b.Return)
	}

	args := make([]ast.Type, len(v.Args))
	for i, a := range v.Args {
		args[i] = c.expr(a, nil)
	}
	if f := c.Info.Functions[v.Func]; f != nil {
		if c.arity(v, len(f.Params)) {
			for i, p := range f.Params {
				c.require(v, args[i], p.Type)
			}
		}
		for _, e := range f.Effects {
			c.effect(v, e)
		}
		return f.ReturnType
	}
	c.error(v, "E_UNDEFINED_SYMBOL", v.Func)
	return nil
}
