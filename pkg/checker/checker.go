package checker

import (
	"fmt"
	"strings"

	"liaf/pkg/ast"
	"liaf/pkg/diagnostic"
)

type SymbolType string

const (
	TypeInt   SymbolType = "int"
	TypeFloat SymbolType = "float"
	TypeStr   SymbolType = "str"
	TypeBool  SymbolType = "bool"
	TypeVoid  SymbolType = "void"
)

type Scope struct {
	parent *Scope
	vars   map[string]string
}

func newScope(parent *Scope) *Scope {
	return &Scope{
		parent: parent,
		vars:   make(map[string]string),
	}
}

func (s *Scope) set(name, typeName string) {
	s.vars[name] = typeName
}

func (s *Scope) get(name string) (string, bool) {
	if t, ok := s.vars[name]; ok {
		return t, true
	}
	if s.parent != nil {
		return s.parent.get(name)
	}
	return "", false
}

type Checker struct {
	filename string
	errors   []diagnostic.Diagnostic
	funcs    map[string]*ast.FuncDecl
}

func New(filename string) *Checker {
	return &Checker{
		filename: filename,
		funcs:    make(map[string]*ast.FuncDecl),
	}
}

func (c *Checker) Check(prog *ast.Program) []diagnostic.Diagnostic {
	// 1. Coleta todas as funções para resolução antecipada
	for _, decl := range prog.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			c.funcs[fn.Name] = fn
		}
	}

	// 2. Valida cada função
	for _, decl := range prog.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			c.checkFunc(fn)
		}
	}

	return c.errors
}

func (c *Checker) checkFunc(fn *ast.FuncDecl) {
	scope := newScope(nil)
	for _, p := range fn.Params {
		scope.set(p.Name, p.Type)
	}

	for _, stmt := range fn.Body {
		c.checkStmt(stmt, scope, fn)
	}
}

func (c *Checker) checkStmt(stmt ast.Stmt, scope *Scope, currentFn *ast.FuncDecl) {
	switch s := stmt.(type) {
	case *ast.LetStmt:
		valType := c.inferExprType(s.Value, scope)
		if s.Type != "" && valType != "" && valType != "unknown" {
			if !typesCompatible(s.Type, valType) {
				c.addDiag(
					"TYPE_MISMATCH",
					"LetStmt",
					fmt.Sprintf("Variável '%s' declarada como '%s', mas recebeu valor do tipo '%s'", s.Name, s.Type, valType),
					s.Type,
					valType,
					fmt.Sprintf("[let %s: %s ...]", s.Name, valType),
					s.Line,
					s.Col,
				)
			}
		}
		targetType := s.Type
		if targetType == "" {
			targetType = valType
		}
		scope.set(s.Name, targetType)

	case *ast.SendStmt:
		chType := c.inferExprType(s.Channel, scope)
		valType := c.inferExprType(s.Value, scope)
		if strings.HasPrefix(chType, "chan[") && strings.HasSuffix(chType, "]") {
			expectedType := chType[5 : len(chType)-1]
			if valType != "" && valType != "unknown" && !typesCompatible(expectedType, valType) {
				chName := "ch"
				if ident, ok := s.Channel.(*ast.IdentifierExpr); ok {
					chName = ident.Name
				}
				c.addDiag(
					"CHANNEL_TYPE_MISMATCH",
					"SendStmt",
					fmt.Sprintf("Canal espera elemento do tipo '%s', mas tentou enviar '%s'", expectedType, valType),
					expectedType,
					valType,
					fmt.Sprintf("[send %s (%s val)]", chName, expectedType),
					s.Line,
					s.Col,
				)
			}
		}

	case *ast.ReturnStmt:
		if s.Value != nil {
			valType := c.inferExprType(s.Value, scope)
			if currentFn.ReturnType != "void" && !typesCompatible(currentFn.ReturnType, valType) {
				c.addDiag(
					"RETURN_TYPE_MISMATCH",
					"ReturnStmt",
					fmt.Sprintf("Função '%s' espera retorno '%s', mas retornou '%s'", currentFn.Name, currentFn.ReturnType, valType),
					currentFn.ReturnType,
					valType,
					fmt.Sprintf("[return (%s ...)]", currentFn.ReturnType),
					s.Line,
					s.Col,
				)
			}
		}

	case *ast.SpawnStmt:
		// Chamada concorrente
		if s.Call != nil {
			c.checkCall(s.Call, scope)
		}

	case *ast.IfStmt:
		condType := c.inferExprType(s.Condition, scope)
		if condType != "bool" && condType != "unknown" {
			c.addDiag(
				"CONDITION_NOT_BOOL",
				"IfStmt",
				fmt.Sprintf("Condição do 'if' deve ser do tipo 'bool', recebido '%s'", condType),
				"bool",
				condType,
				"(eq a b)",
				s.Line,
				s.Col,
			)
		}
		thenScope := newScope(scope)
		for _, sub := range s.ThenBody {
			c.checkStmt(sub, thenScope, currentFn)
		}
		elseScope := newScope(scope)
		for _, sub := range s.ElseBody {
			c.checkStmt(sub, elseScope, currentFn)
		}
	}
}

func (c *Checker) inferExprType(expr ast.Expr, scope *Scope) string {
	if expr == nil {
		return "void"
	}
	switch e := expr.(type) {
	case *ast.IntLiteral:
		return "int"
	case *ast.FloatLiteral:
		return "float"
	case *ast.StringLiteral:
		return "str"
	case *ast.BoolLiteral:
		return "bool"
	case *ast.IdentifierExpr:
		if t, ok := scope.get(e.Name); ok {
			return t
		}
		c.addDiag(
			"UNDEFINED_VARIABLE",
			"Identifier",
			fmt.Sprintf("Identificador '%s' não foi declarado no escopo", e.Name),
			"VariavelDeclarada",
			e.Name,
			fmt.Sprintf("[let %s: int 0]", e.Name),
			e.Line,
			e.Col,
		)
		return "unknown"
	case *ast.BinaryOpExpr:
		switch e.Op {
		case "add", "sub", "mul", "div":
			return "int"
		case "eq", "neq", "gt", "lt", "gte", "lte", "and", "or":
			return "bool"
		}
	case *ast.CallExpr:
		return c.checkCall(e, scope)
	case *ast.RecvExpr:
		chType := c.inferExprType(e.Channel, scope)
		if strings.HasPrefix(chType, "chan[") && strings.HasSuffix(chType, "]") {
			return chType[5 : len(chType)-1]
		}
		return "unknown"
	}
	return "unknown"
}

func (c *Checker) checkCall(call *ast.CallExpr, scope *Scope) string {
	switch call.Fn {
	case "println", "print":
		return "void"
	case "concat", "str":
		return "str"
	case "sleep_ms":
		return "void"
	case "http_serve", "serve_site":
		return "void"
	case "http_get":
		return "str"
	case "make_chan":
		if len(call.Args) > 0 {
			if ident, ok := call.Args[0].(*ast.IdentifierExpr); ok {
				return fmt.Sprintf("chan[%s]", ident.Name)
			}
		}
		return "chan[any]"
	default:
		if fn, ok := c.funcs[call.Fn]; ok {
			return fn.ReturnType
		}
	}
	return "unknown"
}

func typesCompatible(expected, received string) bool {
	if expected == received {
		return true
	}
	if expected == "any" || received == "unknown" {
		return true
	}
	return false
}

func (c *Checker) addDiag(code, node, msg, exp, rec, patch string, line, col int) {
	c.errors = append(c.errors, diagnostic.Diagnostic{
		Code:           code,
		File:           c.filename,
		Line:           line,
		Col:            col,
		Node:           node,
		Message:        msg,
		Expected:       exp,
		Received:       rec,
		SuggestedPatch: patch,
	})
}
