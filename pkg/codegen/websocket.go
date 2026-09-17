package codegen

import (
	"fmt"
	"strings"

	"liaf/pkg/ast"
)

// genWSRoute emite uma declaracao (ws-route ...).
//
// Cada bloco vira uma funcao propria, e nao um case dentro de um laco unico,
// porque assim (on-err ...) se traduz no mesmo fechamento `_liaf_on_err` que
// funcoes e rotas ja usam: um `return` do bloco de erro sai do evento em
// curso sem derrubar a conexao.
func (g *Generator) genWSRoute(w *ast.WSRouteDecl) {
	g.serial++
	id := g.serial
	open := fmt.Sprintf("_liaf_ws_open_%d", id)
	message := fmt.Sprintf("_liaf_ws_message_%d", id)
	closed := fmt.Sprintf("_liaf_ws_close_%d", id)

	var params []string
	for _, p := range w.Params {
		params = append(params, fmt.Sprintf("%s %s", sanitizeIdent(p.Name), mapType(p.Type)))
	}

	g.genWSHandler(open, params, w, w.OnOpen)
	g.genWSHandler(message, append(params, fmt.Sprintf("%s string", sanitizeIdent(w.MsgVar))), w, w.OnMessage)
	g.genWSHandler(closed, params, w, w.OnClose)

	g.sb.WriteString("func init() {\n")
	g.sb.WriteString(fmt.Sprintf("\tweb.RegisterWS(%q, func(_liaf_conn *web.WSConn, _liaf_req web.Request) {\n", w.Path))
	g.indent += 2

	// Indice de cada {nome} no padrao do path, pela mesma razao de genRoute:
	// usar o ultimo segmento quebraria /ws/{sala}/eventos.
	pathIndex := map[string]int{}
	for i, seg := range strings.Split(strings.Trim(w.Path, "/"), "/") {
		if len(seg) > 2 && strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			pathIndex[seg[1:len(seg)-1]] = i
		}
	}
	partsEmitted := false
	var callArgs []string
	for _, p := range w.Params {
		pName := sanitizeIdent(p.Name)
		if mapType(p.Type) == "*web.WSConn" {
			callArgs = append(callArgs, "_liaf_conn")
			continue
		}
		if !partsEmitted {
			partsEmitted = true
			g.writeIndent()
			g.sb.WriteString("parts := strings.Split(strings.Trim(_liaf_req.Path, \"/\"), \"/\")\n")
			g.writeIndent()
			g.sb.WriteString("_ = parts\n")
		}
		idx := pathIndex[p.Name]
		g.writeIndent()
		g.sb.WriteString(fmt.Sprintf("var %s %s\n", pName, mapType(p.Type)))
		g.writeIndent()
		g.sb.WriteString(fmt.Sprintf("if len(parts) > %d {\n", idx))
		g.indent++
		g.writeIndent()
		if mapType(p.Type) == "int64" {
			g.sb.WriteString(fmt.Sprintf("%s, _ = strconv.ParseInt(parts[%d], 10, 64)\n", pName, idx))
		} else {
			g.sb.WriteString(fmt.Sprintf("%s = parts[%d]\n", pName, idx))
		}
		g.indent--
		g.writeIndent()
		g.sb.WriteString("}\n")
		callArgs = append(callArgs, pName)
	}

	joined := strings.Join(callArgs, ", ")
	g.writeIndent()
	g.sb.WriteString(fmt.Sprintf("%s(%s)\n", open, joined))
	g.writeIndent()
	g.sb.WriteString("for {\n")
	g.indent++
	g.writeIndent()
	g.sb.WriteString("_liaf_text, _liaf_alive := _liaf_conn.Next()\n")
	g.writeIndent()
	g.sb.WriteString("if !_liaf_alive {\n")
	g.indent++
	g.writeIndent()
	g.sb.WriteString("break\n")
	g.indent--
	g.writeIndent()
	g.sb.WriteString("}\n")
	g.writeIndent()
	g.sb.WriteString(fmt.Sprintf("%s(%s)\n", message, appendArg(joined, "_liaf_text")))
	g.indent--
	g.writeIndent()
	g.sb.WriteString("}\n")
	// Next devolve false tanto no fechamento limpo quanto na queda da
	// conexao, entao (on-close ...) roda sempre — que e a garantia de que
	// um cliente nunca fica pendurado num topico.
	g.writeIndent()
	g.sb.WriteString(fmt.Sprintf("%s(%s)\n", closed, joined))

	g.indent -= 2
	g.sb.WriteString("\t})\n")
	g.sb.WriteString("}\n\n")
}

func appendArg(joined, extra string) string {
	if joined == "" {
		return extra
	}
	return joined + ", " + extra
}

// genWSHandler emite um dos tres blocos como funcao void.
func (g *Generator) genWSHandler(name string, params []string, w *ast.WSRouteDecl, body []ast.Stmt) {
	oldOnErr, oldRet := g.curOnErrVar, g.curRetType
	g.curOnErrVar = w.OnErrVar
	g.curRetType = nil // void: o desvio de erro sai do evento, sem valor
	defer func() {
		g.curOnErrVar, g.curRetType = oldOnErr, oldRet
	}()

	g.sb.WriteString(fmt.Sprintf("func %s(%s) {\n", name, strings.Join(params, ", ")))
	g.indent++

	if w.OnErrVar != "" {
		g.writeIndent()
		g.sb.WriteString(fmt.Sprintf("_liaf_on_err := func(%s string) {\n", sanitizeIdent(w.OnErrVar)))
		g.indent++
		for _, stmt := range w.OnErrBody {
			g.genStmt(stmt)
		}
		g.indent--
		g.writeIndent()
		g.sb.WriteString("}\n")
		g.writeIndent()
		g.sb.WriteString("var _ = _liaf_on_err\n")
	}

	for _, stmt := range body {
		g.genStmt(stmt)
	}

	g.indent--
	g.sb.WriteString("}\n\n")
}
