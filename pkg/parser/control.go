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
	} else {
		p.addError(fmt.Sprintf("match exige ramos ok/err ou some/none, mas obteve %s", b1), "E_NONEXHAUSTIVE_MATCH")
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

	if b1 == "some" {
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
	} else if b1 == "ok" {
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
	} else {
		p.addError(fmt.Sprintf("match exige ramos ok/err ou some/none, mas obteve %s", b1), "E_NONEXHAUSTIVE_MATCH")
		return nil
	}
}
