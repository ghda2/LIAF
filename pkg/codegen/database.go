package codegen

import (
	"fmt"

	"liaf/pkg/ast"
)

// genDBTransaction emite (db-transaction conn ...).
//
// A forma gerada e o padrao begin/defer rollback/commit do Go. O defer e o
// que torna a transacao correta na presenca de try: um try que falha no meio
// do corpo faz `return` na funcao inteira, e so um defer alcanca essa saida
// para desfazer o que ficou pela metade. Depois do commit o rollback vira
// no-op, entao o caminho feliz nao paga nada.
//
// O bloco Go em volta existe para que o nome da conexao possa ser
// re-declarado: dentro dele `conn` e a transacao, fora continua sendo o pool.
func (g *Generator) genDBTransaction(s *ast.DBTransactionStmt) {
	g.serial++
	id := g.serial
	beginID := fmt.Sprintf("_liaf_tx_%d", id)
	commitID := fmt.Sprintf("_liaf_commit_%d", id)
	conn := sanitizeIdent(s.Conn)

	g.sb.WriteString("{\n")
	g.indent++

	g.writeIndent()
	g.sb.WriteString(fmt.Sprintf("%s := rt.DBBegin(%s)\n", beginID, conn))
	g.failGuard(beginID)

	g.writeIndent()
	g.sb.WriteString(fmt.Sprintf("%s := %s.Value\n", conn, beginID))
	g.writeIndent()
	g.sb.WriteString(fmt.Sprintf("defer rt.DBRollback(%s)\n", conn))

	for _, st := range s.Body {
		g.genStmt(st)
	}

	g.writeIndent()
	g.sb.WriteString(fmt.Sprintf("%s := rt.DBCommit(%s)\n", commitID, conn))
	g.failGuard(commitID)

	g.indent--
	g.writeIndent()
	g.sb.WriteString("}\n")
}
