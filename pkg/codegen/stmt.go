package codegen

import (
	"fmt"
	"strings"

	"liaf/pkg/ast"
)

// onErrReturn emite o desvio de um try que falhou para o bloco (on-err ...).
// Numa funcao void o handler nao devolve valor, entao `return f()` seria
// invalido em Go: a chamada e o return precisam ficar separados.
func (g *Generator) onErrReturn(resID string) string {
	if mapType(g.curRetType) == "" {
		return fmt.Sprintf("_liaf_on_err(%s.Error)\n%sreturn\n", resID, strings.Repeat("\t", g.indent))
	}
	return fmt.Sprintf("return _liaf_on_err(%s.Error)\n", resID)
}

// failGuard emite o desvio de erro de um resultado ja avaliado em resID:
// para o bloco (on-err ...) quando ele existe, ou propaga o erro no proprio
// result de retorno. E o mesmo desvio para try, para let/set com try e para
// as bordas de db-transaction.
func (g *Generator) failGuard(resID string) {
	g.writeIndent()
	g.sb.WriteString(fmt.Sprintf("if !%s.OK {\n", resID))
	g.indent++
	g.writeIndent()
	if g.curOnErrVar != "" {
		g.sb.WriteString(g.onErrReturn(resID))
	} else {
		g.sb.WriteString(fmt.Sprintf("var _zero %s; _zero.OK = false; _zero.Error = %s.Error; return _zero\n", mapType(g.curRetType), resID))
	}
	g.indent--
	g.writeIndent()
	g.sb.WriteString("}\n")
}

func (g *Generator) writeIndent() {
	for i := 0; i < g.indent; i++ {
		g.sb.WriteString("\t")
	}
}

func (g *Generator) genStmt(stmt ast.Stmt) {
	g.writeIndent()
	switch s := stmt.(type) {
	case *ast.LoopControl:
		g.sb.WriteString(s.Kind + "\n")
	case *ast.LoopStmt:
		g.serial++
		id := g.serial
		switch s.Kind {
		case "while":
			g.sb.WriteString("for " + g.genExpr(s.Condition) + " {\n")
		case "for-range":
			g.sb.WriteString(fmt.Sprintf("for %s, _liaf_end_%d := int64(%s), int64(%s); %s < _liaf_end_%d; %s++ {\n", sanitizeIdent(s.Name), id, g.genExpr(s.Start), g.genExpr(s.End), sanitizeIdent(s.Name), id, sanitizeIdent(s.Name)))
		case "for-each":
			g.sb.WriteString(fmt.Sprintf("for _, %s := range (%s).Items {\n", sanitizeIdent(s.Name), g.genExpr(s.Collection)))
		}
		g.indent++
		if s.Name != "" {
			g.writeIndent()
			g.sb.WriteString("_ = " + sanitizeIdent(s.Name) + "\n")
		}
		for _, st := range s.Body {
			g.genStmt(st)
		}
		g.indent--
		g.writeIndent()
		g.sb.WriteString("}\n")
	case *ast.MatchStmt:
		g.serial++
		if s.IsOption {
			id := fmt.Sprintf("_liaf_opt_%d", g.serial)
			g.sb.WriteString(fmt.Sprintf("if %s := %s; %s.Some {\n", id, g.genExpr(s.Value), id))
			g.indent++
			g.writeIndent()
			g.sb.WriteString(fmt.Sprintf("%s := %s.Value; _ = %s\n", sanitizeIdent(s.OKName), id, sanitizeIdent(s.OKName)))
			for _, st := range s.OK {
				g.genStmt(st)
			}
			g.indent--
			g.writeIndent()
			g.sb.WriteString("} else {\n")
			g.indent++
			g.writeIndent()
			for _, st := range s.Err {
				g.genStmt(st)
			}
			g.indent--
			g.writeIndent()
			g.sb.WriteString("}\n")
			return
		}
		id := fmt.Sprintf("_liaf_result_%d", g.serial)
		g.sb.WriteString(fmt.Sprintf("if %s := %s; %s.OK {\n", id, g.genExpr(s.Value), id))
		g.indent++
		g.writeIndent()
		g.sb.WriteString(fmt.Sprintf("%s := %s.Value; _ = %s\n", sanitizeIdent(s.OKName), id, sanitizeIdent(s.OKName)))
		for _, st := range s.OK {
			g.genStmt(st)
		}
		g.indent--
		g.writeIndent()
		g.sb.WriteString("} else {\n")
		g.indent++
		g.writeIndent()
		g.sb.WriteString(fmt.Sprintf("%s := %s.Error; _ = %s\n", sanitizeIdent(s.ErrName), id, sanitizeIdent(s.ErrName)))
		for _, st := range s.Err {
			g.genStmt(st)
		}
		g.indent--
		g.writeIndent()
		g.sb.WriteString("}\n")
	case *ast.LetStmt:
		if tryExpr, ok := s.Value.(*ast.TryExpr); ok {
			g.serial++
			resID := fmt.Sprintf("_liaf_try_%d", g.serial)
			varType := mapType(s.Type)
			g.sb.WriteString(fmt.Sprintf("%s := %s\n", resID, g.genExpr(tryExpr.Expr)))
			g.failGuard(resID)
			g.writeIndent()
			g.sb.WriteString(fmt.Sprintf("var %s %s = %s.Value\n", sanitizeIdent(s.Name), varType, resID))
			g.writeIndent()
			g.sb.WriteString("_ = " + sanitizeIdent(s.Name) + "\n")
			return
		}
		val := g.genExpr(s.Value)
		varType := mapType(s.Type)
		g.sb.WriteString(fmt.Sprintf("var %s %s = %s\n", sanitizeIdent(s.Name), varType, val))
		g.writeIndent()
		g.sb.WriteString("_ = " + sanitizeIdent(s.Name) + "\n")

	case *ast.SetStmt:
		if tryExpr, ok := s.Value.(*ast.TryExpr); ok {
			g.serial++
			resID := fmt.Sprintf("_liaf_try_%d", g.serial)
			g.sb.WriteString(fmt.Sprintf("%s := %s\n", resID, g.genExpr(tryExpr.Expr)))
			g.failGuard(resID)
			g.writeIndent()
			g.sb.WriteString(fmt.Sprintf("%s = %s.Value\n", sanitizeIdent(s.Name), resID))
			return
		}
		val := g.genExpr(s.Value)
		g.sb.WriteString(fmt.Sprintf("%s = %s\n", sanitizeIdent(s.Name), val))

	case *ast.ReturnStmt:
		if s.Value != nil {
			g.sb.WriteString(fmt.Sprintf("return %s\n", g.genExpr(s.Value)))
		} else {
			g.sb.WriteString("return\n")
		}

	case *ast.IfStmt:
		cond := g.genExpr(s.Condition)
		g.sb.WriteString(fmt.Sprintf("if %s {\n", cond))
		g.indent++
		for _, st := range s.Then {
			g.genStmt(st)
		}
		g.indent--

		if len(s.Else) > 0 {
			g.writeIndent()
			g.sb.WriteString("} else {\n")
			g.indent++
			for _, st := range s.Else {
				g.genStmt(st)
			}
			g.indent--
		}
		g.writeIndent()
		g.sb.WriteString("}\n")

	case *ast.DBTransactionStmt:
		g.genDBTransaction(s)

	case *ast.SpawnStmt:
		callExpr := g.genExpr(s.Call)
		g.sb.WriteString(fmt.Sprintf("go %s\n", callExpr))

	case *ast.SendStmt:
		ch := g.genExpr(s.Channel)
		val := g.genExpr(s.Value)
		g.sb.WriteString(fmt.Sprintf("%s <- %s\n", ch, val))

	case *ast.ExprStmt:
		if tryExpr, ok := s.Expr.(*ast.TryExpr); ok {
			g.serial++
			resID := fmt.Sprintf("_liaf_try_%d", g.serial)
			g.sb.WriteString(fmt.Sprintf("%s := %s\n", resID, g.genExpr(tryExpr.Expr)))
			g.failGuard(resID)
			return
		}
		val := g.genExpr(s.Expr)
		if isStandAloneStmt(s.Expr) {
			g.sb.WriteString(fmt.Sprintf("%s\n", val))
		} else {
			g.sb.WriteString(fmt.Sprintf("_ = %s\n", val))
		}
	}
}

func isStandAloneStmt(expr ast.Expr) bool {
	switch expr.(type) {
	case *ast.CallExpr:
		return true
	}
	return false
}
