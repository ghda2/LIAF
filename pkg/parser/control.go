package parser

import (
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

func (p *Parser) parseMatch(line, col int) ast.Stmt {
	p.nextToken()
	s := &ast.MatchStmt{Value: p.parseExpression(), Line: line, Col: col}
	for _, kind := range []string{"ok", "err"} {
		p.expectCur(token.LPAREN)
		if p.curToken.Literal != kind {
			p.addError("match exige ramos ok e err, nessa ordem", "E_NONEXHAUSTIVE_MATCH")
		}
		p.nextToken()
		name := p.identifier()
		body := p.statements()
		if kind == "ok" {
			s.OKName, s.OK = name, body
		} else {
			s.ErrName, s.Err = name, body
		}
	}
	p.expectCur(token.RPAREN)
	return s
}
