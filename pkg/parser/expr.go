package parser

import (
	"fmt"
	"strconv"
	"strings"

	"liaf/pkg/ast"
	"liaf/pkg/token"
)

func (p *Parser) parseIfExpr(line, col int) *ast.IfExpr {
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

	return &ast.IfExpr{
		Condition: cond,
		Then:      thenExpr,
		Else:      elseExpr,
		Line:      line,
		Col:       col,
	}
}

func (p *Parser) parseExpression() ast.Expr {
	line, col := p.curToken.Line, p.curToken.Col

	switch p.curToken.Type {
	case token.INT:
		val, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
		if err != nil {
			p.addError("Inteiro fora do intervalo int64", "E_INVALID_NUMBER")
		}
		lit := &ast.IntLiteral{Value: val, Raw: p.curToken.Literal, Line: line, Col: col}
		p.nextToken()
		return lit

	case token.FLOAT:
		val, err := strconv.ParseFloat(p.curToken.Literal, 64)
		if err != nil {
			p.addError("Float fora do intervalo float64", "E_INVALID_NUMBER")
		}
		lit := &ast.FloatLiteral{Value: val, Raw: p.curToken.Literal, Line: line, Col: col}
		p.nextToken()
		return lit

	case token.BOOL:
		val := p.curToken.Literal == "true"
		lit := &ast.BoolLiteral{Value: val, Line: line, Col: col}
		p.nextToken()
		return lit

	case token.STRING:
		lit := &ast.StringLiteral{Value: p.curToken.Literal, Line: line, Col: col}
		p.nextToken()
		return lit

	case token.IDENT:
		ident := &ast.IdentExpr{Name: p.curToken.Literal, Line: line, Col: col}
		p.nextToken()
		return ident

	case token.PATH:
		expr := p.parsePath()
		p.nextToken()
		return expr

	case token.LPAREN:
		p.nextToken()
		innerLine, innerCol := p.curToken.Line, p.curToken.Col

		if p.curTokenIs(token.CALL) {
			p.nextToken()
			if !p.curTokenIs(token.IDENT) {
				p.addError("Esperado identificador de função em call", "E_EXPECTED_CALL_IDENT")
				return nil
			}
			fnName := p.curToken.Literal
			p.nextToken()

			args := p.parseCallArgs(fnName)
			if !p.expectCur(token.RPAREN) {
				return nil
			}
			return &ast.CallExpr{Func: fnName, Args: args, HasCall: true, Line: innerLine, Col: innerCol}
		}

		if p.curTokenIs(token.TRY) {
			p.nextToken()
			subExpr := p.parseExpression()
			if !p.expectCur(token.RPAREN) {
				return nil
			}
			return &ast.TryExpr{Expr: subExpr, Line: innerLine, Col: innerCol}
		}

		if token.IsOperator(p.curToken.Type) {
			op := p.curToken.Literal
			p.nextToken()

			left := p.parseExpression()
			right := p.parseExpression()

			var cur ast.Expr = &ast.BinaryOpExpr{Op: op, Left: left, Right: right, Line: innerLine, Col: innerCol}

			if op == "add" || op == "mul" || op == "and" || op == "or" {
				for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
					next := p.parseExpression()
					cur = &ast.BinaryOpExpr{Op: op, Left: cur, Right: next, Line: innerLine, Col: innerCol}
				}
			}

			if !p.expectCur(token.RPAREN) {
				return nil
			}
			return cur
		}

		if p.curTokenIs(token.RECV) {
			p.nextToken()
			ch := p.parseExpression()
			if !p.expectCur(token.RPAREN) {
				return nil
			}
			return &ast.RecvExpr{Channel: ch, Line: innerLine, Col: innerCol}
		}

		if p.curTokenIs(token.IF) {
			return p.parseIfExpr(innerLine, innerCol)
		}

		if p.curTokenIs(token.MATCH) {
			return p.parseMatchExpr(innerLine, innerCol)
		}

		// Chamada direta de função: (fn-name arg1 arg2 ...)
		if p.curTokenIs(token.IDENT) {
			fnName := p.curToken.Literal
			p.nextToken()

			args := p.parseCallArgs(fnName)
			if !p.expectCur(token.RPAREN) {
				return nil
			}
			return &ast.CallExpr{Func: fnName, Args: args, HasCall: false, Line: innerLine, Col: innerCol}
		}

		p.addError(fmt.Sprintf("Expressão composta inválida iniciada por %q", p.curToken.Literal), "E_INVALID_EXPR")
		return nil

	default:
		p.addError(fmt.Sprintf("Token inesperado ao iniciar expressão: %q", p.curToken.Literal), "E_UNEXPECTED_TOKEN")
		return nil
	}
}

// parsePath expande o token PATH (issue #012) na mesma AST de (field ...):
// a.b.c vira (field (field a b) c). Checker e codegen nao sabem que o ponto
// existiu, e liafc fmt reimprime a cadeia com ponto.
func (p *Parser) parsePath() ast.Expr {
	tok := p.curToken
	segs := strings.Split(tok.Literal, ".")
	if token.LookupIdent(segs[0]) != token.IDENT {
		p.addError(fmt.Sprintf("%q e palavra reservada e nao pode ser a base de %s; use um nome de variavel", segs[0], tok.Literal), "E_INVALID_EXPR")
		return nil
	}
	var cur ast.Expr = &ast.IdentExpr{Name: segs[0], Line: tok.Line, Col: tok.Col}
	for _, seg := range segs[1:] {
		if seg == "true" || seg == "false" {
			p.addError(fmt.Sprintf("%q nao e nome de campo em %s", seg, tok.Literal), "E_EXPECTED_FIELD_NAME")
			return nil
		}
		name := &ast.IdentExpr{Name: seg, Line: tok.Line, Col: tok.Col}
		cur = &ast.CallExpr{Func: "field", Args: []ast.Expr{cur, name}, Line: tok.Line, Col: tok.Col}
	}
	return cur
}

// parseCallArgs le os argumentos ate o ')'. O segundo argumento de field e
// um nome de campo, nao uma expressao: aceita palavras reservadas, para que
// (field claims sub) leia o campo declarado como (sub str).
func (p *Parser) parseCallArgs(fnName string) []ast.Expr {
	var args []ast.Expr
	for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
		if fnName == "field" && len(args) == 1 && token.IsName(p.curToken) {
			args = append(args, &ast.IdentExpr{Name: p.curToken.Literal, Line: p.curToken.Line, Col: p.curToken.Col})
			p.nextToken()
			continue
		}
		arg := p.parseExpression()
		if arg != nil {
			args = append(args, arg)
		}
	}
	return args
}
