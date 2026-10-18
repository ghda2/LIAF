package parser

import (
	"fmt"
	"liaf/pkg/ast"
	"liaf/pkg/token"
)

func (p *Parser) statements() []ast.Stmt {
	var body []ast.Stmt
	for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
		body = append(body, p.parseStatement())
	}
	p.expectCur(token.RPAREN)
	return body
}

func (p *Parser) identifier() string {
	name := p.curToken.Literal
	p.expectCur(token.IDENT)
	return name
}

func (p *Parser) parseLoop(line, col int) ast.Stmt {
	s := &ast.LoopStmt{Kind: p.curToken.Literal, Line: line, Col: col}
	p.nextToken()
	switch s.Kind {
	case "while":
		s.Condition = p.parseExpression()
	case "for-range":
		s.Name = p.identifier()
		s.Start, s.End = p.parseExpression(), p.parseExpression()
	case "for-each":
		s.Name = p.identifier()
		s.Collection = p.parseExpression()
	}
	s.Body = p.statements()
	return s
}

func (p *Parser) parseMatchExpr(line, col int) *ast.MatchExpr {
	p.nextToken() // consome 'match'
	val := p.parseExpression()
	if !p.expectCur(token.LPAREN) {
		return nil
	}
	b1 := p.curToken.Literal
	p.nextToken()

	s := &ast.MatchExpr{Value: val, Line: line, Col: col}

	if b1 == "some" {
		s.IsOption = true
		s.OKName = p.identifier()
		s.OK = p.parseExpression()
		if !p.expectCur(token.RPAREN) {
			return nil
		}
		if !p.expectCur(token.LPAREN) {
			return nil
		}
		if p.curToken.Literal != "none" {
			p.addError("match de option exige ramo none", "E_NONEXHAUSTIVE_MATCH")
			return nil
		}
		p.nextToken()
		if p.curTokenIs(token.IDENT) {
			s.ErrName = p.curToken.Literal
			p.nextToken()
		}
		s.Err = p.parseExpression()
		if !p.expectCur(token.RPAREN) {
			return nil
		}
	} else if b1 == "ok" {
		s.OKName = p.identifier()
		s.OK = p.parseExpression()
		if !p.expectCur(token.RPAREN) {
			return nil
		}
		if !p.expectCur(token.LPAREN) {
			return nil
		}
		if p.curToken.Literal != "err" {
			p.addError("match exige ramos ok e err, nessa ordem", "E_NONEXHAUSTIVE_MATCH")
			return nil
		}
		p.nextToken()
		s.ErrName = p.identifier()
		s.Err = p.parseExpression()
		if !p.expectCur(token.RPAREN) {
			return nil
		}
	} else if b1 == "case" {
		c := p.parseSingleCase()
		if c != nil {
			s.Cases = append(s.Cases, *c)
		}
		for p.curTokenIs(token.LPAREN) {
			p.nextToken() // consome '('
			if !p.curTokenIs(token.CASE) {
				p.addError(fmt.Sprintf("esperado 'case' no match, obteve %q", p.curToken.Literal), "E_EXPECTED_CASE")
				return nil
			}
			p.nextToken() // consome 'case'
			c2 := p.parseSingleCase()
			if c2 != nil {
				s.Cases = append(s.Cases, *c2)
			}
		}
	} else {
		p.addError(fmt.Sprintf("match exige ramos ok/err, some/none ou case, mas obteve %s", b1), "E_NONEXHAUSTIVE_MATCH")
		return nil
	}
	if !p.expectCur(token.RPAREN) {
		return nil
	}
	return s
}

func (p *Parser) parseMatch(line, col int) ast.Stmt {
	p.nextToken()
	val := p.parseExpression()
	if !p.expectCur(token.LPAREN) {
		return nil
	}
	b1 := p.curToken.Literal
	p.nextToken()

	switch b1 {
	case "some":
		return p.parseMatchSome(line, col, val)
	case "ok":
		return p.parseMatchOk(line, col, val)
	case "case":
		return p.parseMatchCases(line, col, val)
	default:
		p.addError(fmt.Sprintf("match exige ramos ok/err, some/none ou case, mas obteve %s", b1), "E_NONEXHAUSTIVE_MATCH")
		return nil
	}
}

func (p *Parser) parseMatchCases(line, col int, val ast.Expr) ast.Stmt {
	var cases []ast.MatchCase
	firstCase := p.parseSingleCase()
	if firstCase != nil {
		cases = append(cases, *firstCase)
	}

	for p.curTokenIs(token.LPAREN) {
		p.nextToken() // consome '('
		if !p.curTokenIs(token.CASE) {
			p.addError(fmt.Sprintf("esperado 'case' no match, obteve %q", p.curToken.Literal), "E_EXPECTED_CASE")
			return nil
		}
		p.nextToken() // consome 'case'
		c := p.parseSingleCase()
		if c != nil {
			cases = append(cases, *c)
		}
	}

	if !p.expectCur(token.RPAREN) {
		return nil
	}

	return &ast.MatchStmt{
		Value: val,
		Cases: cases,
		Line:  line,
		Col:   col,
	}
}

func (p *Parser) parseSingleCase() *ast.MatchCase {
	cLine, cCol := p.curToken.Line, p.curToken.Col
	pattern := ""
	if p.curTokenIs(token.PATH) || p.curTokenIs(token.IDENT) {
		pattern = p.curToken.Literal
		p.nextToken()
	} else if p.curToken.Literal == "_" {
		pattern = "_"
		p.nextToken()
	} else {
		p.addError("Esperado padrão para o case do match", "E_EXPECTED_MATCH_PATTERN")
		return nil
	}

	var vars []string
	if p.curTokenIs(token.LPAREN) {
		p.nextToken()
		for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
			if !token.IsName(p.curToken) && p.curToken.Literal != "_" {
				p.addError("Esperado nome de variável no padrão", "E_EXPECTED_IDENT")
				return nil
			}
			vars = append(vars, p.curToken.Literal)
			p.nextToken()
		}
		if !p.expectCur(token.RPAREN) {
			return nil
		}
	}

	var body []ast.Stmt
	for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.LPAREN) {
			stmt := p.parseStatement()
			if stmt != nil {
				body = append(body, stmt)
			}
		} else {
			expr := p.parseExpression()
			if expr != nil {
				body = append(body, &ast.ExprStmt{Expr: expr, Line: cLine, Col: cCol})
			}
		}
	}
	if !p.expectCur(token.RPAREN) {
		return nil
	}

	var exprBody ast.Expr
	if len(body) > 0 {
		if es, ok := body[len(body)-1].(*ast.ExprStmt); ok {
			exprBody = es.Expr
		} else if rs, ok := body[len(body)-1].(*ast.ReturnStmt); ok {
			exprBody = rs.Value
		}
	}

	return &ast.MatchCase{
		Pattern:   pattern,
		Variables: vars,
		Body:      body,
		ExprBody:  exprBody,
		Line:      cLine,
		Col:       cCol,
	}
}

func (p *Parser) parseMatchSome(line, col int, val ast.Expr) ast.Stmt {
	okName := p.identifier()
	if !p.curTokenIs(token.LPAREN) && !p.curTokenIs(token.RPAREN) {
		okExpr := p.parseExpression()
		p.expectCur(token.RPAREN)
		p.expectCur(token.LPAREN)
		if p.curToken.Literal != "none" {
			p.addError("match de option exige ramo none", "E_NONEXHAUSTIVE_MATCH")
		}
		p.nextToken()
		var errName string
		if p.curTokenIs(token.IDENT) {
			errName = p.curToken.Literal
			p.nextToken()
		}
		errExpr := p.parseExpression()
		p.expectCur(token.RPAREN)
		p.expectCur(token.RPAREN)
		return &ast.ExprStmt{
			Expr: &ast.MatchExpr{
				Value:    val,
				OKName:   okName,
				OK:       okExpr,
				ErrName:  errName,
				Err:      errExpr,
				IsOption: true,
				Line:     line,
				Col:      col,
			},
			Line: line,
			Col:  col,
		}
	}

	s := &ast.MatchStmt{Value: val, IsOption: true, OKName: okName, Line: line, Col: col}
	s.OK = p.statements()
	p.expectCur(token.LPAREN)
	if p.curToken.Literal != "none" {
		p.addError("match de option exige ramo none", "E_NONEXHAUSTIVE_MATCH")
	}
	p.nextToken()
	if p.curTokenIs(token.IDENT) {
		s.ErrName = p.curToken.Literal
		p.nextToken()
	}
	s.Err = p.statements()
	p.expectCur(token.RPAREN)
	return s
}

func (p *Parser) parseMatchOk(line, col int, val ast.Expr) ast.Stmt {
	okName := p.identifier()
	if !p.curTokenIs(token.LPAREN) && !p.curTokenIs(token.RPAREN) {
		okExpr := p.parseExpression()
		p.expectCur(token.RPAREN)
		p.expectCur(token.LPAREN)
		if p.curToken.Literal != "err" {
			p.addError("match exige ramos ok e err, nessa ordem", "E_NONEXHAUSTIVE_MATCH")
		}
		p.nextToken()
		errName := p.identifier()
		errExpr := p.parseExpression()
		p.expectCur(token.RPAREN)
		p.expectCur(token.RPAREN)
		return &ast.ExprStmt{
			Expr: &ast.MatchExpr{
				Value:    val,
				OKName:   okName,
				OK:       okExpr,
				ErrName:  errName,
				Err:      errExpr,
				IsOption: false,
				Line:     line,
				Col:      col,
			},
			Line: line,
			Col:  col,
		}
	}

	s := &ast.MatchStmt{Value: val, OKName: okName, Line: line, Col: col}
	s.OK = p.statements()
	p.expectCur(token.LPAREN)
	if p.curToken.Literal != "err" {
		p.addError("match exige ramos ok e err, nessa ordem", "E_NONEXHAUSTIVE_MATCH")
	}
	p.nextToken()
	s.ErrName = p.identifier()
	s.Err = p.statements()
	p.expectCur(token.RPAREN)
	return s
}
