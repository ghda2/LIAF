// Package checker validates the complete module before code generation.
package checker

import (
	"fmt"
	"strings"

	"liaf/pkg/ast"
	"liaf/pkg/diagnostic"
)

type Info struct {
	Types     map[ast.Expr]ast.Type
	Functions map[string]*ast.FuncDecl
	Structs   map[string]*ast.StructDecl
	Routes    []*ast.RouteDecl
	WSRoutes  []*ast.WSRouteDecl
}

// knownEffects e a lista unica de efeitos declaraveis. `db` entrou com a
// issue #015: e distinto de `net` porque uma assinatura que diz `db` promete
// mais do que trafego de rede — promete estado externo compartilhado.
// `rand` e `env` entraram com a issue #026 (#026).
const knownEffects = "|io|net|fs|clock|spawn|db|rand|env|"

func validEffect(e string) bool { return strings.Contains(knownEffects, "|"+e+"|") }

// opaqueTypes sao os nomes de tipo que a linguagem reconhece sem que exista
// uma (struct ...) correspondente. Sao opacos de proposito: o programa recebe
// o valor de um builtin, passa adiante e nunca le campo nenhum.
var opaqueTypes = map[string]bool{
	"Request": true, "Response": true, "DBConnection": true, "WSConn": true, "HttpReply": true,
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
		case *ast.WSRouteDecl:
			c.Info.WSRoutes = append(c.Info.WSRoutes, v)
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
			if !validEffect(e) {
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
			if !validEffect(e) {
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
	for _, d := range mod.Decls {
		if w, ok := d.(*ast.WSRouteDecl); ok {
			c.wsRoute(w)
		}
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
