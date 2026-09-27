package codegen

import (
	"fmt"
	"strings"

	"liaf/pkg/ast"
)

func (g *Generator) genRoute(r *ast.RouteDecl) {
	g.serial++
	routeFnName := fmt.Sprintf("_liaf_route_%s_%d", strings.ToLower(r.Method), g.serial)

	oldOnErr := g.curOnErrVar
	oldRet := g.curRetType
	g.curOnErrVar = r.OnErrVar
	g.curRetType = r.ReturnType
	defer func() {
		g.curOnErrVar = oldOnErr
		g.curRetType = oldRet
	}()

	ret := mapType(r.ReturnType)
	if ret != "" {
		ret = " " + ret
	}

	var params []string
	for _, p := range r.Params {
		params = append(params, fmt.Sprintf("%s %s", sanitizeIdent(p.Name), mapType(p.Type)))
	}

	g.sb.WriteString(fmt.Sprintf("func %s(%s)%s {\n", routeFnName, strings.Join(params, ", "), ret))
	g.indent++

	if r.OnErrVar != "" {
		g.writeIndent()
		g.sb.WriteString(fmt.Sprintf("_liaf_on_err := func(%s string)%s {\n", sanitizeIdent(r.OnErrVar), ret))
		g.indent++
		for _, stmt := range r.OnErrBody {
			g.genStmt(stmt)
		}
		g.indent--
		g.writeIndent()
		g.sb.WriteString("}\n")
		g.writeIndent()
		g.sb.WriteString("var _ = _liaf_on_err\n")
	}

	for _, stmt := range r.Body {
		g.genStmt(stmt)
	}

	g.indent--
	g.sb.WriteString("}\n\n")

	g.sb.WriteString("func init() {\n")
	g.sb.WriteString(fmt.Sprintf("\tweb.Register(%q, %q, func(req web.Request) web.Response {\n", strings.ToUpper(r.Method), r.Path))
	g.indent += 2

	// Indice de cada {nome} no padrao do path. Usar sempre o ultimo segmento
	// quebraria rotas como /users/{id}/tasks.
	pathIndex := map[string]int{}
	for i, seg := range strings.Split(strings.Trim(r.Path, "/"), "/") {
		if len(seg) > 2 && strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			pathIndex[seg[1:len(seg)-1]] = i
		}
	}
	partsEmitted := false
	emitParts := func() {
		if partsEmitted {
			return
		}
		partsEmitted = true
		g.writeIndent()
		g.sb.WriteString("parts := strings.Split(strings.Trim(req.Path, \"/\"), \"/\")\n")
		g.writeIndent()
		g.sb.WriteString("_ = parts\n")
	}

	var callArgs []string
	for _, p := range r.Params {
		pName := sanitizeIdent(p.Name)
		pTypeStr := mapType(p.Type)
		if pTypeStr == "web.Request" {
			callArgs = append(callArgs, "req")
			continue
		}
		if idx, isPathParam := pathIndex[p.Name]; isPathParam {
			emitParts()
			g.writeIndent()
			g.sb.WriteString(fmt.Sprintf("var %s %s\n", pName, pTypeStr))
			g.writeIndent()
			g.sb.WriteString(fmt.Sprintf("if len(parts) > %d {\n", idx))
			g.indent++
			g.writeIndent()
			if pTypeStr == "int64" {
				g.sb.WriteString(fmt.Sprintf("%s, _ = strconv.ParseInt(parts[%d], 10, 64)\n", pName, idx))
			} else {
				g.sb.WriteString(fmt.Sprintf("%s = parts[%d]\n", pName, idx))
			}
			g.indent--
			g.writeIndent()
			g.sb.WriteString("}\n")
			callArgs = append(callArgs, pName)
			continue
		}
		g.writeIndent()
		g.sb.WriteString(fmt.Sprintf("var %s %s\n", pName, pTypeStr))
		g.writeIndent()
		g.sb.WriteString(fmt.Sprintf("if err := json.Unmarshal([]byte(req.Body), &%s); err != nil {\n", pName))
		g.indent++
		g.writeIndent()
		g.sb.WriteString("return web.JSONResponse(400, `{\"error\":\"Invalid JSON body\"}`)\n")
		g.indent--
		g.writeIndent()
		g.sb.WriteString("}\n")
		callArgs = append(callArgs, pName)
	}

	g.writeIndent()
	if ret == " web.Response" {
		g.sb.WriteString(fmt.Sprintf("return %s(%s)\n", routeFnName, strings.Join(callArgs, ", ")))
	} else if ret == " void" || ret == "" {
		g.sb.WriteString(fmt.Sprintf("%s(%s)\n", routeFnName, strings.Join(callArgs, ", ")))
		g.writeIndent()
		g.sb.WriteString("return web.JSONResponse(200, `{\"ok\":true}`)\n")
	} else {
		status := "200"
		if strings.ToUpper(r.Method) == "POST" {
			status = "201"
		}
		g.sb.WriteString(fmt.Sprintf("res := %s(%s)\n", routeFnName, strings.Join(callArgs, ", ")))
		g.writeIndent()
		g.sb.WriteString("data, err := json.Marshal(res)\n")
		g.writeIndent()
		g.sb.WriteString("if err != nil { return web.JSONResponse(500, `{\"error\":\"Serialization error\"}`) }\n")
		g.writeIndent()
		g.sb.WriteString(fmt.Sprintf("return web.JSONResponse(%s, string(data))\n", status))
	}

	g.indent -= 2
	g.sb.WriteString("\t})\n")
	g.sb.WriteString("}\n")
}
