package parser

import (
	"liaf/pkg/ast"
	"liaf/pkg/token"
)

// Middlewares HTTP (issue #045). `middleware` e `with` sao palavras contextuais,
// nao reservadas: so tem esse papel no inicio de uma declaracao de topo e logo
// depois do path de uma rota, entao continuam valendo como identificadores.
//
// Um middleware e uma funcao comum com outra palavra na frente; a assinatura
// fixa (req Request) (result Request Response) fica a cargo do checker.

func isMiddlewareKeyword(t token.Token) bool {
	return t.Type == token.IDENT && t.Literal == "middleware"
}

// parseLinearMiddleware le [pub] middleware nome(req Request) (result Request Response) ... end.
func (p *Parser) parseLinearMiddleware() *ast.FuncDecl {
	isPub := p.curTokenIs(token.PUB)
	if isPub {
		p.nextToken() // consome 'pub'
	}
	f := p.parseLinearFunc() // consome 'middleware' no lugar de 'fn'
	if f != nil {
		f.IsPub = isPub
		f.IsMiddleware = true
	}
	return f
}

// parseMiddleware le ([pub] middleware nome ((req Request)) (result Request Response) ...),
// com o '(' de abertura ja consumido.
func (p *Parser) parseMiddleware() *ast.FuncDecl {
	isPub := p.curTokenIs(token.PUB)
	if isPub {
		p.nextToken() // consome 'pub'
	}
	f := p.parseFunc()
	if f != nil {
		f.IsPub = isPub
		f.IsMiddleware = true
	}
	return f
}

// parseLinearWith le o opcional with(a, b) de uma rota linear.
func (p *Parser) parseLinearWith() []string {
	if !p.curTokenIs(token.IDENT) || p.curToken.Literal != "with" || !p.peekTokenIs(token.LPAREN) {
		return nil
	}
	p.nextToken() // consome 'with'
	p.nextToken() // consome '('
	names := p.parseMiddlewareNames(true)
	p.expectCur(token.RPAREN)
	return names
}

// parseWith le o opcional (with a b) de uma rota em S-expression.
func (p *Parser) parseWith() []string {
	if !p.curTokenIs(token.LPAREN) || !p.peekTokenIs(token.IDENT) || p.peekToken.Literal != "with" {
		return nil
	}
	p.nextToken() // consome '('
	p.nextToken() // consome 'with'
	names := p.parseMiddlewareNames(false)
	p.expectCur(token.RPAREN)
	return names
}

func (p *Parser) parseMiddlewareNames(commas bool) []string {
	var names []string
	for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
		if !p.curTokenIs(token.IDENT) {
			p.addError("Esperado nome de middleware em 'with'", "E_EXPECTED_MIDDLEWARE_NAME")
			return names
		}
		names = append(names, p.curToken.Literal)
		p.nextToken()
		if commas && p.curTokenIs(token.COMMA) {
			p.nextToken()
		}
	}
	if len(names) == 0 {
		p.addError("'with' exige ao menos um middleware", "E_EXPECTED_MIDDLEWARE_NAME")
	}
	return names
}
