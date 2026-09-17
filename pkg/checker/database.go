package checker

import (
	"fmt"
	"strings"

	"liaf/pkg/ast"
	"liaf/pkg/dbdrv"
)

// Regras de tipo dos builtins de banco (issue #015).
//
// A mais importante nao e de tipo: o SQL tem de ser um literal escrito no
// fonte. Concatenar valor dentro de uma consulta e o erro que um modelo comete
// com mais naturalidade — a forma errada e a mais curta e a que mais aparece
// em exemplos na internet — e o unico ponto onde da para impedi-lo antes de
// virar producao e aqui.

const dbConnectionType = "DBConnection"

func (c *Checker) dbCall(v *ast.CallExpr, n string) (ast.Type, bool) {
	switch n {
	case "db-connect":
		c.effect(v, "db")
		if !c.arity(v, 2) {
			return nil, true
		}
		c.dbDriverName(v, 0)
		c.require(v, c.expr(v.Args[1], nil), primitive("str"))
		return applied("result", &ast.NamedType{Name: dbConnectionType}, primitive("str")), true

	case "db-close":
		c.effect(v, "db")
		if !c.arity(v, 1) {
			return nil, true
		}
		c.dbConnArg(v, 0)
		return applied("result", primitive("void"), primitive("str")), true

	case "db-query":
		c.effect(v, "db")
		if len(v.Args) < 3 {
			c.error(v, "E_WRONG_ARITY", "db-query expects a connection, a SQL literal, a row struct and one argument per placeholder")
			return nil, true
		}
		c.dbConnArg(v, 0)
		sql, literal := c.sqlLiteral(v, 1)
		row := c.dbRowType(v, 2)
		c.dbParams(v, v.Args[3:])
		if literal {
			c.sqlPlaceholders(v, sql, len(v.Args)-3)
		}
		return applied("result", applied("list", row), primitive("str")), true

	case "db-exec":
		c.effect(v, "db")
		if len(v.Args) < 2 {
			c.error(v, "E_WRONG_ARITY", "db-exec expects a connection, a SQL literal and one argument per placeholder")
			return nil, true
		}
		c.dbConnArg(v, 0)
		sql, literal := c.sqlLiteral(v, 1)
		c.dbParams(v, v.Args[2:])
		if literal {
			c.sqlPlaceholders(v, sql, len(v.Args)-2)
		}
		return applied("result", primitive("int"), primitive("str")), true

	case "redis-get":
		c.effect(v, "db")
		if !c.arity(v, 2) {
			return nil, true
		}
		c.dbConnArg(v, 0)
		c.require(v, c.expr(v.Args[1], nil), primitive("str"))
		return applied("result", primitive("str"), primitive("str")), true

	case "redis-set":
		c.effect(v, "db")
		if !c.arity(v, 4) {
			return nil, true
		}
		c.dbConnArg(v, 0)
		c.require(v, c.expr(v.Args[1], nil), primitive("str"))
		c.require(v, c.expr(v.Args[2], nil), primitive("str"))
		c.require(v, c.expr(v.Args[3], nil), primitive("int"))
		return applied("result", primitive("void"), primitive("str")), true
	}
	return nil, false
}

// dbDriverName exige o nome do driver como literal para poder confronta-lo
// com a lista implementada. "postgresql" e "pg" sao erros faceis de cometer e
// so apareceriam em tempo de execucao se o nome fosse dinamico.
func (c *Checker) dbDriverName(v *ast.CallExpr, i int) {
	lit, ok := v.Args[i].(*ast.StringLiteral)
	if !ok {
		c.expr(v.Args[i], nil)
		c.error(v, "E_UNKNOWN_DRIVER", "the db-connect driver must be a string literal, one of "+strings.Join(dbdrv.Drivers, ", "))
		return
	}
	c.Info.Types[v.Args[i]] = primitive("str")
	for _, d := range dbdrv.Drivers {
		if lit.Value == d {
			return
		}
	}
	c.error(v, "E_UNKNOWN_DRIVER", fmt.Sprintf("unknown driver %q, expected one of %s", lit.Value, strings.Join(dbdrv.Drivers, ", ")))
}

func (c *Checker) dbConnArg(v *ast.CallExpr, i int) {
	t := c.expr(v.Args[i], nil)
	if t != nil && name(t) != dbConnectionType {
		c.error(v, "E_TYPE_MISMATCH", fmt.Sprintf("%s expects a %s as its first argument, received %s", v.Func, dbConnectionType, name(t)))
	}
}

// dbRowType valida o tipo de linha. Exigir struct e proposital: a conversao
// casa o nome da coluna com o nome do campo, e um escalar nao tem nome de
// campo com que casar.
func (c *Checker) dbRowType(v *ast.CallExpr, i int) ast.Type {
	t := c.typeArg(v.Args[i])
	if c.Info.Structs[name(t)] == nil {
		c.error(v, "E_INVALID_TYPE", fmt.Sprintf("db-query requires a struct row type, received %s; declare a (struct ...) whose fields match the selected columns", name(t)))
	}
	return t
}

func (c *Checker) dbParams(v *ast.CallExpr, args []ast.Expr) {
	for _, a := range args {
		t := c.expr(a, nil)
		if t != nil && !scalar(t) {
			c.error(v, "E_TYPE_MISMATCH", fmt.Sprintf("query arguments must be int, float, str or bool, received %s", name(t)))
		}
	}
}

// sqlLiteral e a barreira contra injecao: o texto da consulta tem de estar
// escrito no fonte. Com isso nenhum valor de runtime consegue virar sintaxe
// SQL, porque o unico caminho para o banco e a lista de parametros.
func (c *Checker) sqlLiteral(v *ast.CallExpr, i int) (string, bool) {
	if lit, ok := v.Args[i].(*ast.StringLiteral); ok {
		c.Info.Types[v.Args[i]] = primitive("str")
		return lit.Value, true
	}
	c.expr(v.Args[i], nil)
	c.error(v, "E_SQL_INTERPOLATION", fmt.Sprintf(
		"the %s query must be a string literal; pass every value as an argument ($1, $2 ... on postgres, ? on mysql) instead of building the SQL with concat", v.Func))
	return "", false
}

// sqlPlaceholders confere a contagem de marcadores contra a de argumentos.
// O banco so reclamaria disso na primeira execucao real; aqui o erro aparece
// no liafc check.
func (c *Checker) sqlPlaceholders(v *ast.CallExpr, sql string, supplied int) {
	dollars, questions, ambiguous := countPlaceholders(sql)
	if ambiguous {
		// Os operadores de JSONB do postgres (?, ?| e ?&) sao indistinguiveis
		// de marcadores numa varredura lexica, entao a contagem e abandonada
		// em vez de acusar um falso positivo sem escapatoria.
		return
	}
	if dollars > 0 && questions > 0 {
		c.error(v, "E_SQL_PARAM_COUNT", "the query mixes $1 and ? placeholders; use the style of the driver you connected with")
		return
	}
	expected := dollars
	if questions > 0 {
		expected = questions
	}
	if expected != supplied {
		c.error(v, "E_SQL_PARAM_COUNT", fmt.Sprintf(
			"the query has %d placeholder(s) but %d argument(s) were supplied", expected, supplied))
	}
}

// countPlaceholders varre o SQL ignorando literais, identificadores entre
// aspas e comentarios — um "?" dentro de 'texto?' nao e marcador. Devolve o
// maior $n encontrado, a contagem de ?, e se a contagem de ? e confiavel.
func countPlaceholders(sql string) (dollars, questions int, ambiguous bool) {
	for i := 0; i < len(sql); i++ {
		switch sql[i] {
		case '\'':
			i = skipQuoted(sql, i, '\'')
		case '"':
			i = skipQuoted(sql, i, '"')
		case '-':
			if i+1 < len(sql) && sql[i+1] == '-' {
				for i < len(sql) && sql[i] != '\n' {
					i++
				}
			}
		case '/':
			if i+1 < len(sql) && sql[i+1] == '*' {
				end := strings.Index(sql[i+2:], "*/")
				if end < 0 {
					return dollars, questions, ambiguous
				}
				i += 2 + end + 1
			}
		case '$':
			j := i + 1
			for j < len(sql) && sql[j] >= '0' && sql[j] <= '9' {
				j++
			}
			if j > i+1 {
				if n := atoiSafe(sql[i+1 : j]); n > dollars {
					dollars = n
				}
				i = j - 1
				continue
			}
			// $tag$ ... $tag$: literal com delimitador proprio do postgres.
			if end := strings.IndexByte(sql[i+1:], '$'); end >= 0 {
				tag := sql[i : i+1+end+1]
				if close := strings.Index(sql[i+len(tag):], tag); close >= 0 {
					i += len(tag) + close + len(tag) - 1
				}
			}
		case '?':
			// ?, ?| e ?& tambem sao operadores de JSONB no postgres.
			if i+1 < len(sql) && (sql[i+1] == '|' || sql[i+1] == '&' || sql[i+1] == '?') {
				ambiguous = true
				i++
				continue
			}
			questions++
		}
	}
	return dollars, questions, ambiguous
}

func skipQuoted(sql string, start int, quote byte) int {
	for i := start + 1; i < len(sql); i++ {
		if sql[i] != quote {
			continue
		}
		if i+1 < len(sql) && sql[i+1] == quote { // '' escapa a aspa
			i++
			continue
		}
		return i
	}
	return len(sql)
}

func atoiSafe(s string) int {
	n := 0
	for _, ch := range s {
		n = n*10 + int(ch-'0')
		if n > 1<<20 {
			return 1 << 20
		}
	}
	return n
}

// dbTransaction valida (db-transaction conn ...). Dentro do bloco o nome da
// conexao passa a designar a transacao; e o mesmo tipo, entao as consultas do
// corpo nao mudam de forma, mas passam a cair dentro dela.
func (c *Checker) dbTransaction(s *ast.DBTransactionStmt) {
	c.effect(s, "db")

	b := c.lookup(s.Conn)
	switch {
	case b == nil:
		c.error(s, "E_UNDEFINED_SYMBOL", s.Conn)
	case b.pending:
		c.error(s, "E_UNHANDLED_RESULT", s.Conn+" must be matched or returned before opening a transaction")
	case name(b.typ) != dbConnectionType:
		c.error(s, "E_TYPE_MISMATCH", fmt.Sprintf("db-transaction requires a %s, received %s", dbConnectionType, name(b.typ)))
	}

	// Abrir e confirmar a transacao pode falhar, e o desvio gerado e o mesmo
	// do try: sem um destino para o erro nao ha para onde desviar.
	if c.fn != nil && c.fn.OnErrVar == "" && !IsResult(c.fn.ReturnType) {
		c.error(s, "E_UNHANDLED_RESULT", "db-transaction requires either an (on-err ...) block or a function returning (result ...)")
	}

	c.push()
	c.define(s.Conn, &ast.NamedType{Name: dbConnectionType}, s, false)
	c.statements(s.Body)
	c.pop()
}
