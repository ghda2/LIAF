package parser

import (
	"fmt"

	"liaf/pkg/ast"
	"liaf/pkg/lexer"
	"liaf/pkg/token"
)

type Diagnostic struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Col     int    `json:"col"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

type Parser struct {
	l           *lexer.Lexer
	file        string
	curToken    token.Token
	peekToken   token.Token
	Diagnostics []Diagnostic
}

func New(l *lexer.Lexer, file string) *Parser {
	p := &Parser{
		l:    l,
		file: file,
	}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectCur(t token.TokenType) bool {
	if p.curTokenIs(t) {
		p.nextToken()
		return true
	}
	p.addError(fmt.Sprintf("Esperado token %q, mas obteve %q (%s)", t, p.curToken.Literal, p.curToken.Type), "E_UNEXPECTED_TOKEN")
	return false
}

func (p *Parser) addError(msg string, code string) {
	p.Diagnostics = append(p.Diagnostics, Diagnostic{
		File:    p.file,
		Line:    p.curToken.Line,
		Col:     p.curToken.Col,
		Message: msg,
		Code:    code,
	})
	panic(parseAbort{})
}

type parseAbort struct{}

// ParseModule analisa o arquivo completo, suportando tanto S-expressions (v0.2-v0.5) quanto sintaxe linear (v0.6)
func (p *Parser) ParseModule() (result *ast.Module) {
	defer func() {
		if v := recover(); v != nil {
			if _, ok := v.(parseAbort); !ok {
				panic(v)
			}
			result = nil
		}
	}()

	if p.curTokenIs(token.LPAREN) && p.peekTokenIs(token.MODULE) {
		return p.parseSExprModule()
	}

	return p.parseLinearModule()
}

func (p *Parser) parseSExprModule() *ast.Module {
	startLine, startCol := p.curToken.Line, p.curToken.Col
	p.nextToken() // consome '('

	if !p.curTokenIs(token.MODULE) {
		p.addError(fmt.Sprintf("Esperado 'module' no início do programa, mas obteve %q", p.curToken.Literal), "E_EXPECTED_MODULE")
		return nil
	}
	p.nextToken() // consome 'module'

	if !p.curTokenIs(token.IDENT) {
		p.addError("Esperado identificador para o nome do módulo", "E_EXPECTED_MODULE_NAME")
		return nil
	}
	modName := p.curToken.Literal
	p.nextToken() // consome nome do modulo

	mod := &ast.Module{
		Name: modName,
		Line: startLine,
		Col:  startCol,
	}

	for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
		decl := p.parseTopLevel()
		if decl != nil {
			mod.Decls = append(mod.Decls, decl)
		} else {
			p.nextToken()
		}
	}

	if !p.expectCur(token.RPAREN) {
		return nil
	}

	return mod
}
