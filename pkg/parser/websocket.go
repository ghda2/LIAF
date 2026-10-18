package parser

import (
	"liaf/pkg/ast"
	"liaf/pkg/token"
)

// parseWSRoute le
// (ws-route "/caminho" (params ...) (effects ...) [(on-err v ...)]
//   [(on-open ...)] (on-message msg ...) [(on-close ...)])
//
// A ordem dos blocos e fixa, como o resto da gramatica da LIAF: um formato
// canonico unico e o que permite ao liafc fmt ter uma saida so e a um modelo
// gerar a declaracao sem escolher entre variantes equivalentes.
func (p *Parser) parseWSRoute() *ast.WSRouteDecl {
	line, col := p.curToken.Line, p.curToken.Col
	p.nextToken() // consome 'ws-route'

	if !p.curTokenIs(token.STRING) {
		p.addError("Esperado path em string para 'ws-route'", "E_EXPECTED_ROUTE_PATH")
		return nil
	}
	decl := &ast.WSRouteDecl{Path: p.curToken.Literal, Line: line, Col: col}
	p.nextToken()

	decl.Params = p.parseParams()
	decl.Effects = p.parseEffects()

	if p.curTokenIs(token.LPAREN) && p.peekTokenIs(token.ON_ERR) {
		p.nextToken()
		p.nextToken()
		if !p.curTokenIs(token.IDENT) {
			p.addError("Esperado identificador para a variável de erro em 'on-err'", "E_EXPECTED_ON_ERR_VAR")
			return nil
		}
		decl.OnErrVar = p.identifier()
		decl.OnErrBody = p.statements()
	}

	if p.curTokenIs(token.LPAREN) && p.peekTokenIs(token.ON_OPEN) {
		p.nextToken()
		p.nextToken()
		decl.OnOpen = p.statements()
	}

	if !p.curTokenIs(token.LPAREN) || !p.peekTokenIs(token.ON_MESSAGE) {
		p.addError("'ws-route' exige um bloco (on-message nome ...)", "E_EXPECTED_ON_MESSAGE")
		return nil
	}
	p.nextToken()
	p.nextToken()
	if !p.curTokenIs(token.IDENT) {
		p.addError("Esperado identificador para a mensagem recebida em 'on-message'", "E_EXPECTED_ON_MESSAGE_VAR")
		return nil
	}
	decl.MsgVar = p.identifier()
	decl.OnMessage = p.statements()

	if p.curTokenIs(token.LPAREN) && p.peekTokenIs(token.ON_CLOSE) {
		p.nextToken()
		p.nextToken()
		decl.OnClose = p.statements()
	}

	if !p.expectCur(token.RPAREN) {
		return nil
	}
	return decl
}
