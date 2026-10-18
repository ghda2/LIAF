package parser

import (
	"fmt"

	"liaf/pkg/ast"
	"liaf/pkg/token"
)

func (p *Parser) parseType() ast.Type {
	line, col := p.curToken.Line, p.curToken.Col

	if p.curTokenIs(token.IDENT) {
		name := p.curToken.Literal
		p.nextToken()
		switch name {
		case "int", "float", "str", "bool", "void":
			return &ast.PrimitiveType{Name: name, Line: line, Col: col}
		default:
			return &ast.NamedType{Name: name, Line: line, Col: col}
		}
	}

	if p.curTokenIs(token.LPAREN) {
		p.nextToken()
		if !p.curTokenIs(token.IDENT) {
			p.addError("Esperado construtor de tipo (chan, list, map, option, result)", "E_EXPECTED_TYPE_CONSTRUCTOR")
			return nil
		}
		constructor := p.curToken.Literal
		switch constructor {
		case "chan", "list", "map", "option", "result":
			// valido
		default:
			p.addError(fmt.Sprintf("Construtor de tipo desconhecido %q. Esperado chan, list, map, option ou result", constructor), "E_INVALID_TYPE_CONSTRUCTOR")
			return nil
		}
		p.nextToken()

		var args []ast.Type
		for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
			arg := p.parseType()
			if arg != nil {
				args = append(args, arg)
			}
		}

		if !p.expectCur(token.RPAREN) {
			return nil
		}

		return &ast.AppliedType{Constructor: constructor, Args: args, Line: line, Col: col}
	}

	p.addError(fmt.Sprintf("Tipo inválido: %q", p.curToken.Literal), "E_INVALID_TYPE")
	return nil
}
