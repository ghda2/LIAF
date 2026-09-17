package checker

import (
	"fmt"
	"strings"

	"liaf/pkg/ast"
)

// Regras de tipo dos builtins e da declaracao (ws-route ...) da issue #016.

const wsConnType = "WSConn"

func (c *Checker) wsCall(v *ast.CallExpr, n string) (ast.Type, bool) {
	switch n {
	case "ws-send":
		c.effect(v, "net")
		if !c.arity(v, 2) {
			return nil, true
		}
		c.wsConnArg(v, 0)
		c.require(v, c.expr(v.Args[1], nil), primitive("str"))
		return applied("result", primitive("void"), primitive("str")), true

	case "ws-send-json":
		// Aceita qualquer valor: e o atalho para nao escrever
		// (ws-send conn (try (json-encode v))) a cada envio.
		c.effect(v, "net")
		if !c.arity(v, 2) {
			return nil, true
		}
		c.wsConnArg(v, 0)
		c.expr(v.Args[1], nil)
		return applied("result", primitive("void"), primitive("str")), true

	case "ws-close":
		c.effect(v, "net")
		if !c.arity(v, 3) {
			return nil, true
		}
		c.wsConnArg(v, 0)
		c.require(v, c.expr(v.Args[1], nil), primitive("int"))
		c.require(v, c.expr(v.Args[2], nil), primitive("str"))
		return applied("result", primitive("void"), primitive("str")), true

	case "ws-broadcast":
		c.effect(v, "net")
		if !c.arity(v, 2) {
			return nil, true
		}
		c.require(v, c.expr(v.Args[0], nil), primitive("str"))
		c.require(v, c.expr(v.Args[1], nil), primitive("str"))
		return applied("result", primitive("int"), primitive("str")), true

	case "ws-join", "ws-leave":
		// Nao devolvem result: inscrever e desinscrever sao operacoes locais
		// sobre um mapa em memoria e nao tem caminho de falha. Embrulha-las
		// em result obrigaria a um try que nunca desvia.
		c.effect(v, "net")
		if !c.arity(v, 2) {
			return nil, true
		}
		c.wsConnArg(v, 0)
		c.require(v, c.expr(v.Args[1], nil), primitive("str"))
		return primitive("void"), true

	case "ws-topic-size":
		c.effect(v, "net")
		if !c.arity(v, 1) {
			return nil, true
		}
		c.require(v, c.expr(v.Args[0], nil), primitive("str"))
		return primitive("int"), true
	}
	return nil, false
}

func (c *Checker) wsConnArg(v *ast.CallExpr, i int) {
	t := c.expr(v.Args[i], nil)
	if t != nil && name(t) != wsConnType {
		c.error(v, "E_TYPE_MISMATCH", fmt.Sprintf("%s expects a %s as its first argument, received %s", v.Func, wsConnType, name(t)))
	}
}

// wsRoute valida uma declaracao (ws-route ...). Os tres blocos compartilham os
// parametros mas nao as variaveis locais: cada um vira uma funcao propria no
// codigo gerado, chamada num momento diferente da vida da conexao.
func (c *Checker) wsRoute(w *ast.WSRouteDecl) {
	where := "ws-route " + w.Path

	// c.fn e o contexto que effect() e try consultam. O retorno void e o que
	// faz try exigir (on-err ...): um handler de evento nao devolve resultado
	// a ninguem, entao nao ha para onde propagar o erro sem o bloco.
	c.fn = &ast.FuncDecl{
		Name:       "ws_" + w.Path,
		Params:     w.Params,
		ReturnType: primitive("void"),
		Effects:    w.Effects,
		OnErrVar:   w.OnErrVar,
		OnErrBody:  w.OnErrBody,
		Line:       w.Line,
		Col:        w.Col,
	}
	c.scopes = nil
	c.push()
	c.loops = 0
	c.used = map[string]bool{}

	declaredEffects := map[string]bool{}
	for _, e := range w.Effects {
		if !validEffect(e) {
			c.error(w, "E_UNKNOWN_EFFECT", e)
		}
		if declaredEffects[e] {
			c.error(w, "E_DUPLICATE_EFFECT", e)
		}
		declaredEffects[e] = true
	}
	// Manter a conexao aberta ja e trafego de rede, mesmo que o corpo so
	// imprima: sem isto, (effects net io) de um handler que so loga cairia em
	// E_UNUSED_EFFECT, e tirar net tornaria ws-send impossivel.
	if !declaredEffects["net"] {
		c.error(w, "E_UNDECLARED_EFFECT", where+" holds a network connection and must declare the net effect")
	}
	c.used["net"] = true

	pathParams := map[string]bool{}
	for _, seg := range strings.Split(strings.Trim(w.Path, "/"), "/") {
		if len(seg) > 2 && strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			pathParams[seg[1:len(seg)-1]] = true
		}
	}

	connections := 0
	declared := map[string]ast.Type{}
	for _, p := range w.Params {
		c.validType(p.Type, false)
		c.define(p.Name, p.Type, w, false)
		declared[p.Name] = p.Type
		if name(p.Type) == wsConnType {
			connections++
			continue
		}
		if !pathParams[p.Name] {
			// Nao ha corpo JSON num handshake de WebSocket, entao um
			// parametro que nao vem do path nao teria de onde ser preenchido.
			c.error(w, "E_WS_PARAM", fmt.Sprintf(
				"parameter %s does not match any {name} in the path; a ws-route only takes path parameters and one %s", p.Name, wsConnType))
			continue
		}
		if n := name(p.Type); n != "int" && n != "str" {
			c.error(w, "E_WS_PARAM", fmt.Sprintf("path parameter {%s} must be int or str, received %s", p.Name, n))
		}
	}
	for pname := range pathParams {
		if _, ok := declared[pname]; !ok {
			c.error(w, "E_WS_PARAM", fmt.Sprintf("path parameter {%s} has no matching entry in (params ...)", pname))
		}
	}
	if connections != 1 {
		c.error(w, "E_WS_PARAM", fmt.Sprintf("a ws-route requires exactly one parameter of type %s, found %d", wsConnType, connections))
	}

	if w.OnErrVar != "" {
		c.push()
		c.define(w.OnErrVar, primitive("str"), w, false)
		c.statements(w.OnErrBody)
		c.pop()
	}

	c.push()
	c.statements(w.OnOpen)
	c.pop()

	c.push()
	c.define(w.MsgVar, primitive("str"), w, false)
	c.statements(w.OnMessage)
	c.pop()

	c.push()
	c.statements(w.OnClose)
	c.pop()

	c.unusedEffects(w, w.Effects, where)
	c.pop()
}
