// Package checker validates the complete module before code generation.
package checker

import (
	"fmt"
	"liaf/pkg/ast"
	"liaf/pkg/diagnostic"
	"sort"
	"strings"
)

type Info struct {
	Types     map[ast.Expr]ast.Type
	Functions map[string]*ast.FuncDecl
	Structs   map[string]*ast.StructDecl
	Routes    []*ast.RouteDecl
}
type binding struct {
	typ     ast.Type
	node    ast.Node
	pending bool
}
type Checker struct {
	Info   *Info
	Errors []diagnostic.Diagnostic
	file   string
	scopes []map[string]*binding
	fn     *ast.FuncDecl
	loops  int
	used   map[string]bool
}

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

func Check(mod *ast.Module, file string) (*Info, []diagnostic.Diagnostic) {
	c := &Checker{file: file, Info: &Info{Types: map[ast.Expr]ast.Type{}, Functions: map[string]*ast.FuncDecl{}, Structs: map[string]*ast.StructDecl{}}}
	seen := map[string]bool{}
	for _, d := range mod.Decls {
		var id string
		switch v := d.(type) {
		case *ast.FuncDecl:
			id = v.Name
			c.Info.Functions[id] = v
		case *ast.StructDecl:
			id = v.Name
			c.Info.Structs[id] = v
		case *ast.RouteDecl:
			c.Info.Routes = append(c.Info.Routes, v)
		case *ast.ImportDecl:
			c.error(v, "E_UNRESOLVED_IMPORT", "Imports must be resolved before checking")
		}
		if id != "" {
			if seen[id] {
				c.error(d, "E_DUPLICATE_SYMBOL", id)
			}
			seen[id] = true
		}
	}
	for _, decl := range mod.Decls {
		s, ok := decl.(*ast.StructDecl)
		if !ok {
			continue
		}
		fields := map[string]bool{}
		for _, f := range s.Fields {
			c.validType(f.Type, false)
			if fields[f.Name] {
				c.error(s, "E_DUPLICATE_SYMBOL", f.Name)
			}
			fields[f.Name] = true
		}
	}
	for _, d := range mod.Decls {
		f, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		c.fn = f
		c.scopes = nil
		c.push()
		c.loops = 0
		c.validType(f.ReturnType, true)
		for _, p := range f.Params {
			c.validType(p.Type, false)
			c.define(p.Name, p.Type, f, false)
		}
		effects := map[string]bool{}
		for _, e := range f.Effects {
			if !strings.Contains("|io|net|fs|clock|spawn|", "|"+e+"|") {
				c.error(f, "E_UNKNOWN_EFFECT", e)
			}
			if effects[e] {
				c.error(f, "E_DUPLICATE_EFFECT", e)
			}
			effects[e] = true
		}
		if f.Name == "main" && (len(f.Params) != 0 || name(f.ReturnType) != "void") {
			c.error(f, "E_MAIN_SIGNATURE", "main requires no parameters and returns void")
		}
		c.used = map[string]bool{}
		if f.OnErrVar != "" {
			c.push()
			c.define(f.OnErrVar, primitive("str"), f, false)
			c.statements(f.OnErrBody)
			c.pop()
		}
		c.statements(f.Body)
		if name(f.ReturnType) != "void" && !returns(f.Body) {
			c.error(f, "E_MISSING_RETURN", f.Name)
		}
		c.unusedEffects(f, f.Effects, "function "+f.Name)
		c.pop()
	}
	for _, d := range mod.Decls {
		r, ok := d.(*ast.RouteDecl)
		if !ok {
			continue
		}
		c.fn = &ast.FuncDecl{
			Name:       fmt.Sprintf("route_%s_%s", r.Method, r.Path),
			Params:     r.Params,
			ReturnType: r.ReturnType,
			Effects:    r.Effects,
			OnErrVar:   r.OnErrVar,
			OnErrBody:  r.OnErrBody,
			Body:       r.Body,
			Line:       r.Line,
			Col:        r.Col,
		}
		c.scopes = nil
		c.push()
		c.loops = 0
		c.validType(r.ReturnType, true)
		for _, p := range r.Params {
			c.validType(p.Type, false)
			c.define(p.Name, p.Type, r, false)
		}
		for _, e := range r.Effects {
			if !strings.Contains("|io|net|fs|clock|spawn|", "|"+e+"|") {
				c.error(r, "E_UNKNOWN_EFFECT", e)
			}
		}
		// Cada {nome} no path precisa de uma entrada correspondente em (params ...).
		// Sem isso o codegen geraria um parametro lido do corpo JSON em vez do path.
		declared := map[string]ast.Type{}
		for _, prm := range r.Params {
			declared[prm.Name] = prm.Type
		}
		for _, seg := range strings.Split(strings.Trim(r.Path, "/"), "/") {
			if len(seg) <= 2 || !strings.HasPrefix(seg, "{") || !strings.HasSuffix(seg, "}") {
				continue
			}
			pname := seg[1 : len(seg)-1]
			t, ok := declared[pname]
			if !ok {
				c.error(r, "E_ROUTE_PARAM", fmt.Sprintf("path parameter {%s} has no matching entry in (params ...)", pname))
				continue
			}
			if n := name(t); n != "int" && n != "str" {
				c.error(r, "E_ROUTE_PARAM", fmt.Sprintf("path parameter {%s} must be int or str, received %s", pname, n))
			}
		}
		c.used = map[string]bool{}
		if r.OnErrVar != "" {
			c.push()
			c.define(r.OnErrVar, primitive("str"), r, false)
			c.statements(r.OnErrBody)
			c.pop()
		}
		c.statements(r.Body)
		if name(r.ReturnType) != "void" && !returns(r.Body) {
			c.error(r, "E_MISSING_RETURN", fmt.Sprintf("route %s %s", r.Method, r.Path))
		}
		c.unusedEffects(r, r.Effects, fmt.Sprintf("route %s %s", r.Method, r.Path))
		c.pop()
	}
	if c.Info.Functions["main"] == nil {
		c.error(mod, "E_MISSING_MAIN", "Module requires main")
	}
	return c.Info, c.Errors
}
func (c *Checker) error(n ast.Node, code, msg string) {
	l, col := n.Pos()
	c.Errors = append(c.Errors, diagnostic.Diagnostic{Code: code, File: c.file, Line: l, Col: col, Message: msg})
}
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
		if c.Info.Structs[v.Name] == nil && v.Name != "Request" && v.Name != "Response" {
			c.error(t, "E_UNKNOWN_TYPE", v.Name)
		}
	case *ast.AppliedType:
		arity := map[string]int{"chan": 1, "list": 1, "map": 2, "result": 2}[v.Constructor]
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
func (c *Checker) effect(n ast.Node, e string) {
	for _, v := range c.fn.Effects {
		if e == v {
			if c.used != nil {
				c.used[e] = true
			}
			return
		}
	}
	c.error(n, "E_UNDECLARED_EFFECT", e)
}

// unusedEffects acusa efeitos declarados que nenhuma chamada do corpo consome.
// Um (effects fs) que nao toca o disco engana quem le a assinatura - inclusive
// um modelo gerando codigo a partir dela.
func (c *Checker) unusedEffects(n ast.Node, declared []string, where string) {
	for _, e := range declared {
		if !c.used[e] {
			c.error(n, "E_UNUSED_EFFECT", fmt.Sprintf("%s declares effect %s but never uses it", where, e))
		}
	}
}
func (c *Checker) push() { c.scopes = append(c.scopes, map[string]*binding{}) }
func (c *Checker) pop() {
	scope := c.scopes[len(c.scopes)-1]
	ids := make([]string, 0, len(scope))
	for id := range scope {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		b := scope[id]
		if b.pending {
			c.error(b.node, "E_UNHANDLED_RESULT", id+" must be matched or returned")
		}
	}
	c.scopes = c.scopes[:len(c.scopes)-1]
}
func (c *Checker) define(id string, t ast.Type, n ast.Node, pending bool) {
	s := c.scopes[len(c.scopes)-1]
	if s[id] != nil {
		c.error(n, "E_DUPLICATE_SYMBOL", id)
	}
	s[id] = &binding{t, n, pending}
}
func (c *Checker) lookup(id string) *binding {
	for i := len(c.scopes) - 1; i >= 0; i-- {
		if b := c.scopes[i][id]; b != nil {
			return b
		}
	}
	return nil
}
func (c *Checker) consume(e ast.Expr) {
	if id, ok := e.(*ast.IdentExpr); ok {
		if b := c.lookup(id.Name); b != nil {
			b.pending = false
		}
	}
}
func (c *Checker) block(body []ast.Stmt) { c.push(); c.statements(body); c.pop() }

func (c *Checker) pending() map[*binding]bool {
	state := map[*binding]bool{}
	for _, s := range c.scopes {
		for _, b := range s {
			state[b] = b.pending
		}
	}
	return state
}
func restore(state map[*binding]bool) {
	for b, p := range state {
		b.pending = p
	}
}
func merge(a, b map[*binding]bool) {
	for binding, p := range a {
		binding.pending = p || b[binding]
	}
}

func (c *Checker) statements(body []ast.Stmt) {
	for _, st := range body {
		switch s := st.(type) {
		case *ast.LetStmt:
			c.validType(s.Type, false)
			c.require(s, c.expr(s.Value, s.Type), s.Type)
			c.define(s.Name, s.Type, s, IsResult(s.Type))
		case *ast.SetStmt:
			b := c.lookup(s.Name)
			if b == nil {
				c.error(s, "E_UNDEFINED_SYMBOL", s.Name)
				c.expr(s.Value, nil)
			} else {
				if b.pending {
					c.error(s, "E_UNHANDLED_RESULT", s.Name)
				}
				c.require(s, c.expr(s.Value, b.typ), b.typ)
				b.pending = IsResult(b.typ)
			}
		case *ast.ReturnStmt:
			c.require(s, c.expr(s.Value, c.fn.ReturnType), c.fn.ReturnType)
			c.consume(s.Value)
			for _, scope := range c.scopes {
				for id, b := range scope {
					if b.pending {
						c.error(s, "E_UNHANDLED_RESULT", id+" is unhandled on this return path")
					}
				}
			}
		case *ast.ExprStmt:
			if IsResult(c.expr(s.Expr, nil)) {
				c.error(s, "E_UNHANDLED_RESULT", "Use match or return for fallible operations")
			}
		case *ast.IfStmt:
			c.require(s, c.expr(s.Condition, nil), primitive("bool"))
			before := c.pending()
			c.block(s.Then)
			afterThen := c.pending()
			restore(before)
			c.block(s.Else)
			if returns(s.Then) { /* only the else path continues */
			} else if returns(s.Else) {
				restore(afterThen)
			} else {
				merge(afterThen, c.pending())
			}
		case *ast.LoopStmt:
			before := c.pending()
			c.push()
			switch s.Kind {
			case "while":
				c.require(s, c.expr(s.Condition, nil), primitive("bool"))
			case "for-range":
				c.require(s, c.expr(s.Start, nil), primitive("int"))
				c.require(s, c.expr(s.End, nil), primitive("int"))
				c.define(s.Name, primitive("int"), s, false)
			case "for-each":
				a := parts(c.expr(s.Collection, nil), "list", 1)
				if a == nil {
					c.error(s, "E_TYPE_MISMATCH", "for-each requires list")
				} else {
					c.define(s.Name, a[0], s, false)
				}
			}
			c.loops++
			c.statements(s.Body)
			c.loops--
			c.pop()
			merge(before, c.pending())
		case *ast.LoopControl:
			if c.loops == 0 {
				c.error(s, "E_LOOP_CONTROL", s.Kind+" outside loop")
			}
		case *ast.MatchStmt:
			a := parts(c.expr(s.Value, nil), "result", 2)
			if a == nil {
				c.error(s, "E_TYPE_MISMATCH", "match requires result")
				continue
			}
			c.consume(s.Value)
			before := c.pending()
			c.push()
			c.define(s.OKName, a[0], s, IsResult(a[0]))
			c.statements(s.OK)
			c.pop()
			afterOK := c.pending()
			restore(before)
			c.push()
			c.define(s.ErrName, a[1], s, IsResult(a[1]))
			c.statements(s.Err)
			c.pop()
			if returns(s.OK) {
			} else if returns(s.Err) {
				restore(afterOK)
			} else {
				merge(afterOK, c.pending())
			}
		case *ast.SpawnStmt:
			c.effect(s, "spawn")
			c.require(s, c.expr(s.Call, nil), primitive("void"))
		case *ast.SendStmt:
			a := parts(c.expr(s.Channel, nil), "chan", 1)
			v := c.expr(s.Value, nil)
			if a == nil {
				c.error(s, "E_CHANNEL_TYPE_MISMATCH", "send requires channel")
			} else {
				c.require(s, v, a[0])
			}
		}
	}
}
func returns(body []ast.Stmt) bool {
	for _, st := range body {
		switch s := st.(type) {
		case *ast.ReturnStmt:
			return true
		case *ast.IfStmt:
			if returns(s.Then) && returns(s.Else) {
				return true
			}
		case *ast.MatchStmt:
			if returns(s.OK) && returns(s.Err) {
				return true
			}
		}
	}
	return false
}
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
				c.error(e, "E_TYPE_MISMATCH", "Numeric operands required")
			}
		}
		if v.Op == "gt" || v.Op == "lt" || v.Op == "gte" || v.Op == "lte" {
			return primitive("bool")
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
	id, ok := e.(*ast.IdentExpr)
	if !ok {
		c.error(e, "E_INVALID_TYPE", "Expected type name")
		return primitive("int")
	}
	var t ast.Type = &ast.NamedType{Name: id.Name}
	if scalar(primitive(id.Name)) {
		t = primitive(id.Name)
	}
	c.validType(t, false)
	return t
}
func (c *Checker) call(v *ast.CallExpr, want ast.Type) ast.Type {
	n := strings.ReplaceAll(v.Func, "_", "-")
	if n == "make-list" || n == "make-chan" || n == "make-map" {
		count := 1
		if n == "make-map" {
			count = 2
		}
		if !c.arity(v, count) {
			return nil
		}
		a := []ast.Type{}
		for _, e := range v.Args {
			a = append(a, c.typeArg(e))
		}
		t := applied(strings.TrimPrefix(n, "make-"), a...)
		c.validType(t, false)
		return t
	}
	if n == "ok" || n == "err" {
		if !c.arity(v, 1) {
			return nil
		}
		a := parts(want, "result", 2)
		if a == nil {
			c.error(v, "E_RESULT_CONTEXT", "ok/err require an explicit result type")
			c.expr(v.Args[0], nil)
			return nil
		}
		i := 0
		if n == "err" {
			i = 1
		}
		c.require(v, c.expr(v.Args[0], a[i]), a[i])
		return want
	}
	if n == "json-decode" {
		c.effect(v, "io")
		if !c.arity(v, 2) {
			return nil
		}
		c.require(v, c.expr(v.Args[0], nil), primitive("str"))
		return applied("result", c.typeArg(v.Args[1]), primitive("str"))
	}
	if n == "new" {
		if len(v.Args) == 0 {
			c.arity(v, 1)
			return nil
		}
		t := c.typeArg(v.Args[0])
		s := c.Info.Structs[name(t)]
		if s == nil {
			c.error(v, "E_INVALID_TYPE", "new requires struct")
			return nil
		}
		if !c.arity(v, len(s.Fields)+1) {
			return nil
		}
		for i, f := range s.Fields {
			c.require(v, c.expr(v.Args[i+1], f.Type), f.Type)
		}
		return t
	}
	if n == "field" {
		if !c.arity(v, 2) {
			return nil
		}
		t := c.expr(v.Args[0], nil)
		id, ok := v.Args[1].(*ast.IdentExpr)
		if !ok {
			c.error(v, "E_UNKNOWN_FIELD", "Expected field name")
			return nil
		}
		if s := c.Info.Structs[name(t)]; s != nil {
			for _, f := range s.Fields {
				if f.Name == id.Name {
					return f.Type
				}
			}
		}
		c.error(v, "E_UNKNOWN_FIELD", id.Name)
		return nil
	}
	if n == "http-get" || n == "http-post" || n == "http-put" || n == "http-delete" {
		c.effect(v, "net")
		if !c.arity(v, 2) {
			return nil
		}
		c.require(v, c.expr(v.Args[0], nil), primitive("str"))
		id, ok := v.Args[1].(*ast.IdentExpr)
		var f *ast.FuncDecl
		if ok {
			f = c.Info.Functions[id.Name]
		}
		if f == nil || len(f.Params) != 1 || name(f.Params[0].Type) != "Request" || name(f.ReturnType) != "Response" {
			c.error(v, "E_HANDLER_SIGNATURE", "Handler requires (Request) -> Response")
		} else {
			for _, eff := range f.Effects {
				c.effect(v, eff)
			}
		}
		return primitive("void")
	}
	args := make([]ast.Type, len(v.Args))
	for i, a := range v.Args {
		args[i] = c.expr(a, nil)
	}
	check := func(types ...string) bool {
		if !c.arity(v, len(types)) {
			return false
		}
		for i, t := range types {
			c.require(v, args[i], primitive(t))
		}
		return true
	}
	result := func(t string) ast.Type { return applied("result", primitive(t), primitive("str")) }
	switch n {
	case "println", "print", "concat":
		if n != "concat" {
			c.effect(v, "io")
		}
		for _, a := range args {
			if a != nil && !scalar(a) {
				c.error(v, "E_TYPE_MISMATCH", "Use explicit serialization for composite values")
			}
		}
		if n == "concat" {
			return primitive("str")
		}
		return primitive("void")
	case "not":
		check("bool")
		return primitive("bool")
	case "str-from-int":
		check("int")
		return primitive("str")
	case "float-from-int":
		check("int")
		return primitive("float")
	case "int-from-str":
		check("str")
		return result("int")
	case "sleep-ms":
		c.effect(v, "clock")
		check("int")
		return primitive("void")
	case "serve-site":
		c.effect(v, "net")
		c.effect(v, "fs")
		check("str", "str", "str", "bool")
		return primitive("void")
	case "serve-hybrid":
		c.effect(v, "net")
		c.effect(v, "fs")
		check("str", "str")
		return result("bool")
	case "json-response":
		check("int", "str")
		return &ast.NamedType{Name: "Response"}
	case "request-body", "request-path", "request-method":
		check("Request")
		return primitive("str")
	case "fs-read-file":
		c.effect(v, "fs")
		check("str")
		return result("str")
	case "fs-write-file":
		c.effect(v, "fs")
		check("str", "str")
		return result("void")
	case "fs-rename":
		c.effect(v, "fs")
		check("str", "str")
		return result("void")
	case "fs-write-atomic":
		c.effect(v, "fs")
		check("str", "str")
		return result("void")
	case "fs-remove":
		c.effect(v, "fs")
		check("str")
		return result("bool")
	case "fs-exists":
		c.effect(v, "fs")
		check("str")
		return primitive("bool")
	case "json-encode":
		c.effect(v, "io")
		c.arity(v, 1)
		return result("str")
	case "str-len":
		check("str")
		return primitive("int")
	case "str-slice":
		check("str", "int", "int")
		return result("str")
	case "str-eq":
		check("str", "str")
		return primitive("bool")
	case "args":
		c.effect(v, "io")
		check()
		return applied("list", primitive("str"))
	}
	if strings.HasPrefix(n, "list-") || strings.HasPrefix(n, "map-") {
		kind := "list"
		arity := 1
		if strings.HasPrefix(n, "map-") {
			kind = "map"
			arity = 2
		}
		if len(args) == 0 {
			c.arity(v, 1)
			return nil
		}
		a := parts(args[0], kind, arity)
		if a == nil {
			c.error(v, "E_TYPE_MISMATCH", "Expected "+kind)
			return nil
		}
		switch n {
		case "list-len", "map-len":
			c.arity(v, 1)
			return primitive("int")
		case "list-push":
			if c.arity(v, 2) {
				c.require(v, args[1], a[0])
			}
			return primitive("void")
		case "list-get":
			if c.arity(v, 2) {
				c.require(v, args[1], primitive("int"))
			}
			return applied("result", a[0], primitive("str"))
		case "list-set":
			if c.arity(v, 3) {
				c.require(v, args[1], primitive("int"))
				c.require(v, args[2], a[0])
			}
			return result("bool")
		case "map-get", "map-has":
			if c.arity(v, 2) {
				c.require(v, args[1], a[0])
			}
			if n == "map-has" {
				return primitive("bool")
			}
			return applied("result", a[1], primitive("str"))
		case "map-set":
			if c.arity(v, 3) {
				c.require(v, args[1], a[0])
				c.require(v, args[2], a[1])
			}
			return primitive("void")
		}
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
