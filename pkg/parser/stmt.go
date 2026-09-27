package parser

import (
	"fmt"

	"liaf/pkg/ast"
	"liaf/pkg/token"
)

func (p *Parser) parseStatement() ast.Stmt {
	line, col := p.curToken.Line, p.curToken.Col
	if !p.expectCur(token.LPAREN) {
		return nil
	}

	switch p.curToken.Type {
	case token.WHILE, token.FOR_RANGE, token.FOR_EACH:
		return p.parseLoop(line, col)
	case token.BREAK, token.CONTINUE:
		kind := p.curToken.Literal
		p.nextToken()
		p.expectCur(token.RPAREN)
		return &ast.LoopControl{Kind: kind, Line: line, Col: col}
	case token.MATCH:
		return p.parseMatch(line, col)
	case token.DB_TRANSACTION:
		return p.parseDBTransaction(line, col)
	case token.LET:
		return p.parseLetStmt(line, col)
	case token.SET:
		return p.parseSetStmt(line, col)
	case token.RETURN:
		return p.parseReturnStmt(line, col)
	case token.IF:
		return p.parseIfStmt(line, col)
	case token.SPAWN:
		return p.parseSpawnStmt(line, col)
	case token.SEND:
		return p.parseSendStmt(line, col)
	case token.DO:
		return p.parseExprStmt(line, col)
	case token.TRY:
		p.nextToken()
		sub := p.parseExpression()
		if !p.expectCur(token.RPAREN) {
			return nil
		}
		return &ast.ExprStmt{Expr: &ast.TryExpr{Expr: sub, Line: line, Col: col}, Line: line, Col: col}
	case token.CALL:
		p.nextToken()
		if !p.curTokenIs(token.IDENT) {
			p.addError("Esperado identificador de função em call", "E_EXPECTED_CALL_IDENT")
			return nil
		}
		fnName := p.curToken.Literal
		p.nextToken()
		var args []ast.Expr
		for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
			arg := p.parseExpression()
			if arg != nil {
				args = append(args, arg)
			}
		}
		if !p.expectCur(token.RPAREN) {
			return nil
		}
		return &ast.ExprStmt{Expr: &ast.CallExpr{Func: fnName, Args: args, HasCall: true, Line: line, Col: col}, Line: line, Col: col}
	case token.IDENT:
		fnName := p.curToken.Literal
		p.nextToken()
		var args []ast.Expr
		for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
			arg := p.parseExpression()
			if arg != nil {
				args = append(args, arg)
			}
		}
		if !p.expectCur(token.RPAREN) {
			return nil
		}
		return &ast.ExprStmt{Expr: &ast.CallExpr{Func: fnName, Args: args, HasCall: false, Line: line, Col: col}, Line: line, Col: col}
	default:
		if token.IsOperator(p.curToken.Type) {
			op := p.curToken.Literal
			p.nextToken()
			left := p.parseExpression()
			right := p.parseExpression()
			var cur ast.Expr = &ast.BinaryOpExpr{Op: op, Left: left, Right: right, Line: line, Col: col}
			if op == "add" || op == "mul" || op == "and" || op == "or" {
				for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
					next := p.parseExpression()
					cur = &ast.BinaryOpExpr{Op: op, Left: cur, Right: next, Line: line, Col: col}
				}
			}
			if !p.expectCur(token.RPAREN) {
				return nil
			}
			return &ast.ExprStmt{Expr: cur, Line: line, Col: col}
		}
		p.addError(fmt.Sprintf("Comando desconhecido: %q", p.curToken.Literal), "E_UNKNOWN_STATEMENT")
		return nil
	}
}

func (p *Parser) parseLetStmt(line, col int) *ast.LetStmt {
	p.nextToken()

	if !p.curTokenIs(token.IDENT) {
		p.addError("Esperado identificador no let", "E_EXPECTED_LET_IDENT")
		return nil
	}
	name := p.curToken.Literal
	p.nextToken()

	var varType ast.Type
	var val ast.Expr

	hasExplicitType := false
	if p.curTokenIs(token.IDENT) && !p.peekTokenIs(token.RPAREN) {
		hasExplicitType = true
	} else if p.curTokenIs(token.LPAREN) {
		switch p.peekToken.Literal {
		case "chan", "list", "map", "option", "result":
			hasExplicitType = true
		}
	}

	if hasExplicitType {
		varType = p.parseType()
		if varType == nil {
			return nil
		}
		val = p.parseExpression()
		if val == nil {
			return nil
		}
	} else {
		val = p.parseExpression()
		if val == nil {
			return nil
		}
	}

	if !p.expectCur(token.RPAREN) {
		return nil
	}

	return &ast.LetStmt{Name: name, Type: varType, Value: val, Line: line, Col: col}
}

func (p *Parser) parseSetStmt(line, col int) *ast.SetStmt {
	p.nextToken()

	if !p.curTokenIs(token.IDENT) {
		p.addError("Esperado identificador no set", "E_EXPECTED_SET_IDENT")
		return nil
	}
	name := p.curToken.Literal
	p.nextToken()

	val := p.parseExpression()
	if val == nil {
		return nil
	}

	if !p.expectCur(token.RPAREN) {
		return nil
	}

	return &ast.SetStmt{Name: name, Value: val, Line: line, Col: col}
}

func (p *Parser) parseReturnStmt(line, col int) *ast.ReturnStmt {
	p.nextToken()

	var val ast.Expr
	if !p.curTokenIs(token.RPAREN) {
		val = p.parseExpression()
	}

	if !p.expectCur(token.RPAREN) {
		return nil
	}

	return &ast.ReturnStmt{Value: val, Line: line, Col: col}
}

func (p *Parser) parseIfStmt(line, col int) ast.Stmt {
	p.nextToken()

	cond := p.parseExpression()
	if cond == nil {
		return nil
	}

	if !p.expectCur(token.LPAREN) {
		return nil
	}
	if !p.expectCur(token.THEN) {
		return nil
	}

	if !p.curTokenIs(token.LPAREN) {
		thenExpr := p.parseExpression()
		if !p.expectCur(token.RPAREN) {
			return nil
		}
		var elseExpr ast.Expr
		if p.curTokenIs(token.LPAREN) && p.peekTokenIs(token.ELSE) {
			p.nextToken()
			p.nextToken()
			elseExpr = p.parseExpression()
			if !p.expectCur(token.RPAREN) {
				return nil
			}
		}
		if !p.expectCur(token.RPAREN) {
			return nil
		}
		return &ast.ExprStmt{
			Expr: &ast.IfExpr{
				Condition: cond,
				Then:      thenExpr,
				Else:      elseExpr,
				Line:      line,
				Col:       col,
			},
			Line: line,
			Col:  col,
		}
	}

	var thenStmts []ast.Stmt
	for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
		st := p.parseStatement()
		if st != nil {
			thenStmts = append(thenStmts, st)
		}
	}
	if !p.expectCur(token.RPAREN) {
		return nil
	}

	var elseStmts []ast.Stmt
	if p.curTokenIs(token.LPAREN) && p.peekTokenIs(token.ELSE) {
		p.nextToken()
		p.nextToken()
		for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
			st := p.parseStatement()
			if st != nil {
				elseStmts = append(elseStmts, st)
			}
		}
		if !p.expectCur(token.RPAREN) {
			return nil
		}
	}

	if !p.expectCur(token.RPAREN) {
		return nil
	}

	return &ast.IfStmt{
		Condition: cond,
		Then:      thenStmts,
		Else:      elseStmts,
		Line:      line,
		Col:       col,
	}
}

func (p *Parser) parseSpawnStmt(line, col int) *ast.SpawnStmt {
	p.nextToken()

	expr := p.parseExpression()
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		p.addError("spawn requer uma expressão 'call'", "E_EXPECTED_CALL_IN_SPAWN")
		return nil
	}

	if !p.expectCur(token.RPAREN) {
		return nil
	}

	return &ast.SpawnStmt{Call: call, Line: line, Col: col}
}

func (p *Parser) parseSendStmt(line, col int) *ast.SendStmt {
	p.nextToken()

	ch := p.parseExpression()
	val := p.parseExpression()

	if !p.expectCur(token.RPAREN) {
		return nil
	}

	return &ast.SendStmt{Channel: ch, Value: val, Line: line, Col: col}
}

func (p *Parser) parseExprStmt(line, col int) *ast.ExprStmt {
	p.nextToken()

	expr := p.parseExpression()
	if expr == nil {
		return nil
	}

	if !p.expectCur(token.RPAREN) {
		return nil
	}

	return &ast.ExprStmt{Expr: expr, Line: line, Col: col}
}
