package parser

import (
	"liaf/pkg/ast"
)

// parseDBTransaction le (db-transaction conn instrucoes...).
//
// O alvo e um identificador, e nao uma expressao qualquer, porque dentro do
// bloco esse nome passa a designar a transacao. Um nome so pode ser
// re-designado se for, de fato, um nome.
func (p *Parser) parseDBTransaction(line, col int) ast.Stmt {
	p.nextToken() // consome 'db-transaction'

	s := &ast.DBTransactionStmt{Conn: p.identifier(), Line: line, Col: col}
	s.Body = p.statements()
	return s
}
