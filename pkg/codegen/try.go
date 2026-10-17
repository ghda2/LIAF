package codegen

import (
	"fmt"

	"liaf/pkg/ast"
)

// Um `try` aninhado numa expressao — json-response(200, try json-encode(x)),
// return try f(), ok(2 * try g()) — precisa da mesma checagem de erro que o
// `let x = try ...`. Go nao tem um operador de desvio dentro de expressoes, entao
// cada try e avaliado antes da instrucao, com a guarda de falha, e a expressao
// passa a ler o .Value ja conferido.
//
// So sobe o que a instrucao avaliaria de qualquer jeito: o lado direito de && e
// ||, os ramos de if/match como expressao e a condicao do while ficam onde estao,
// porque antecipa-los mudaria quando (ou se) eles rodam.

// hoistStmtTries emite as guardas dos try aninhados da instrucao. O recuo da
// linha corrente ja foi escrito; devolve true se escreveu algo, e entao o
// chamador precisa recuar de novo antes da instrucao.
func (g *Generator) hoistStmtTries(stmt ast.Stmt) bool {
	var exprs []ast.Expr
	switch s := stmt.(type) {
	case *ast.LetStmt:
		exprs = append(exprs, innerOfTopTry(s.Value))
	case *ast.SetStmt:
		exprs = append(exprs, innerOfTopTry(s.Value))
	case *ast.ExprStmt:
		exprs = append(exprs, innerOfTopTry(s.Expr))
	case *ast.ReturnStmt:
		exprs = append(exprs, s.Value)
	case *ast.IfStmt:
		exprs = append(exprs, s.Condition)
	case *ast.MatchStmt:
		exprs = append(exprs, s.Value)
	case *ast.LoopStmt:
		if s.Kind == "for-range" {
			exprs = append(exprs, s.Start, s.End)
		} else if s.Kind == "for-each" {
			exprs = append(exprs, s.Collection)
		}
	}
	wrote := false
	for _, e := range exprs {
		g.hoistTries(e, &wrote)
	}
	return wrote
}

// innerOfTopTry pula o try do topo: let/set/expressao com try direto ja tem a
// guarda propria no genStmt; so o que esta dentro dele precisa subir.
func innerOfTopTry(e ast.Expr) ast.Expr {
	if t, ok := e.(*ast.TryExpr); ok {
		return t.Expr
	}
	return e
}

func (g *Generator) hoistTries(e ast.Expr, wrote *bool) {
	switch v := e.(type) {
	case nil:
		return
	case *ast.TryExpr:
		g.hoistTries(v.Expr, wrote) // try f(try g()): o de dentro primeiro
		g.serial++
		id := fmt.Sprintf("_liaf_try_%d", g.serial)
		if *wrote {
			g.writeIndent()
		}
		*wrote = true
		g.sb.WriteString(fmt.Sprintf("%s := %s\n", id, g.genExpr(v.Expr)))
		g.failGuard(id)
		if g.hoisted == nil {
			g.hoisted = map[*ast.TryExpr]string{}
		}
		g.hoisted[v] = id
	case *ast.CallExpr:
		if v.Func == "and" || v.Func == "or" {
			if len(v.Args) > 0 {
				g.hoistTries(v.Args[0], wrote)
			}
			return
		}
		for _, a := range v.Args {
			g.hoistTries(a, wrote)
		}
	case *ast.BinaryOpExpr:
		g.hoistTries(v.Left, wrote)
		switch v.Op {
		case "&&", "||", "and", "or":
			return
		}
		g.hoistTries(v.Right, wrote)
	case *ast.IfExpr:
		g.hoistTries(v.Condition, wrote)
	case *ast.MatchExpr:
		g.hoistTries(v.Value, wrote)
	case *ast.RecvExpr:
		g.hoistTries(v.Channel, wrote)
	}
}
