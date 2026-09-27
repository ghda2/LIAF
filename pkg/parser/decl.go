package parser

import (
	"fmt"

	"liaf/pkg/ast"
	"liaf/pkg/token"
)

func (p *Parser) parseTopLevel() ast.TopLevel {
	if !p.curTokenIs(token.LPAREN) {
		p.addError(fmt.Sprintf("Declaração de nível superior deve iniciar com '(', obteve %q", p.curToken.Literal), "E_EXPECTED_LPAREN")
		return nil
	}
	p.nextToken() // consome '('

	switch p.curToken.Type {
	case token.IMPORT:
		return p.parseImport()
	case token.STRUCT:
		return p.parseStruct()
	case token.FN:
		return p.parseFunc()
	case token.ROUTE:
		return p.parseRoute()
	case token.WS_ROUTE:
		return p.parseWSRoute()
	default:
		p.addError(fmt.Sprintf("Declaração inválida: esperado 'import', 'struct', 'fn', 'route' ou 'ws-route', mas obteve %q", p.curToken.Literal), "E_INVALID_TOPLEVEL")
		return nil
	}
}

func (p *Parser) parseImport() *ast.ImportDecl {
	line, col := p.curToken.Line, p.curToken.Col
	p.nextToken() // consome 'import'

	if !p.curTokenIs(token.STRING) {
		p.addError("Esperada string para o caminho do import", "E_EXPECTED_IMPORT_PATH")
		return nil
	}
	path := p.curToken.Literal
	p.nextToken()

	if !p.expectCur(token.RPAREN) {
		return nil
	}
	return &ast.ImportDecl{Path: path, Line: line, Col: col}
}

func (p *Parser) parseStruct() *ast.StructDecl {
	line, col := p.curToken.Line, p.curToken.Col
	p.nextToken() // consome 'struct'

	if !p.curTokenIs(token.IDENT) {
		p.addError("Esperado identificador para o nome da struct", "E_EXPECTED_STRUCT_NAME")
		return nil
	}
	name := p.curToken.Literal
	p.nextToken()

	var fields []ast.Field

	if p.curTokenIs(token.LPAREN) && p.peekTokenIs(token.FIELDS) {
		p.nextToken() // consome '('
		p.nextToken() // consome 'fields'
		for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
			f := p.parseField()
			if f != nil {
				fields = append(fields, *f)
			}
		}
		if !p.expectCur(token.RPAREN) {
			return nil
		}
	} else {
		for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
			f := p.parseField()
			if f != nil {
				fields = append(fields, *f)
			}
		}
	}

	if !p.expectCur(token.RPAREN) {
		return nil
	}

	return &ast.StructDecl{Name: name, Fields: fields, Line: line, Col: col}
}

func (p *Parser) parseField() *ast.Field {
	if !p.expectCur(token.LPAREN) {
		return nil
	}
	line, col := p.curToken.Line, p.curToken.Col
	if !token.IsName(p.curToken) {
		p.addError("Esperado nome do campo da struct", "E_EXPECTED_FIELD_NAME")
		return nil
	}
	fieldName := p.curToken.Literal
	p.nextToken()

	fType := p.parseType()
	if fType == nil {
		return nil
	}

	if !p.expectCur(token.RPAREN) {
		return nil
	}
	return &ast.Field{Name: fieldName, Type: fType, Line: line, Col: col}
}

func (p *Parser) parseFunc() *ast.FuncDecl {
	line, col := p.curToken.Line, p.curToken.Col
	p.nextToken() // consome 'fn'

	if !p.curTokenIs(token.IDENT) {
		p.addError("Esperado identificador para o nome da função", "E_EXPECTED_FN_NAME")
		return nil
	}
	fnName := p.curToken.Literal
	p.nextToken()

	params := p.parseParams()
	retType := p.parseReturns()
	effects := p.parseEffects()

	var onErrVar string
	var onErrBody []ast.Stmt
	if p.curTokenIs(token.LPAREN) && p.peekTokenIs(token.ON_ERR) {
		p.nextToken() // consome '('
		p.nextToken() // consome 'on-err'
		if !p.curTokenIs(token.IDENT) {
			p.addError("Esperado identificador para a variável de erro em 'on-err'", "E_EXPECTED_ON_ERR_VAR")
			return nil
		}
		onErrVar = p.curToken.Literal
		p.nextToken()
		for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
			stmt := p.parseStatement()
			if stmt != nil {
				onErrBody = append(onErrBody, stmt)
			} else {
				p.nextToken()
			}
		}
		if !p.expectCur(token.RPAREN) {
			return nil
		}
	}

	var body []ast.Stmt
	if p.curTokenIs(token.LPAREN) && p.peekTokenIs(token.BODY) {
		body = p.parseBody()
	} else {
		for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
			stmt := p.parseStatement()
			if stmt != nil {
				body = append(body, stmt)
			}
		}
	}

	// Implicit return: se a função não retorna void e o último statement for uma expressão (ExprStmt)
	if retType != nil {
		if prim, ok := retType.(*ast.PrimitiveType); !ok || prim.Name != "void" {
			if len(body) > 0 {
				if es, ok := body[len(body)-1].(*ast.ExprStmt); ok {
					body[len(body)-1] = &ast.ReturnStmt{
						Value: es.Expr,
						Line:  es.Line,
						Col:   es.Col,
					}
				}
			}
		}
	}

	if !p.expectCur(token.RPAREN) {
		return nil
	}

	return &ast.FuncDecl{
		Name:       fnName,
		Params:     params,
		ReturnType: retType,
		Effects:    effects,
		OnErrVar:   onErrVar,
		OnErrBody:  onErrBody,
		Body:       body,
		Line:       line,
		Col:        col,
	}
}

func (p *Parser) parseRoute() *ast.RouteDecl {
	line, col := p.curToken.Line, p.curToken.Col
	p.nextToken() // consome 'route'

	if !p.curTokenIs(token.IDENT) {
		p.addError("Esperado método HTTP (GET, POST, etc.) em 'route'", "E_EXPECTED_ROUTE_METHOD")
		return nil
	}
	method := p.curToken.Literal
	p.nextToken()

	if !p.curTokenIs(token.STRING) {
		p.addError("Esperado path em string para 'route'", "E_EXPECTED_ROUTE_PATH")
		return nil
	}
	path := p.curToken.Literal
	p.nextToken()

	params := p.parseParams()
	retType := p.parseReturns()
	effects := p.parseEffects()

	var onErrVar string
	var onErrBody []ast.Stmt
	if p.curTokenIs(token.LPAREN) && p.peekTokenIs(token.ON_ERR) {
		p.nextToken() // consome '('
		p.nextToken() // consome 'on-err'
		if !p.curTokenIs(token.IDENT) {
			p.addError("Esperado identificador para a variável de erro em 'on-err'", "E_EXPECTED_ON_ERR_VAR")
			return nil
		}
		onErrVar = p.curToken.Literal
		p.nextToken()
		for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
			stmt := p.parseStatement()
			if stmt != nil {
				onErrBody = append(onErrBody, stmt)
			} else {
				p.nextToken()
			}
		}
		if !p.expectCur(token.RPAREN) {
			return nil
		}
	}

	var body []ast.Stmt
	if p.curTokenIs(token.LPAREN) && p.peekTokenIs(token.BODY) {
		body = p.parseBody()
	} else {
		for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
			stmt := p.parseStatement()
			if stmt != nil {
				body = append(body, stmt)
			}
		}
	}

	// Implicit return para rotas não-void
	if retType != nil {
		if prim, ok := retType.(*ast.PrimitiveType); !ok || prim.Name != "void" {
			if len(body) > 0 {
				if es, ok := body[len(body)-1].(*ast.ExprStmt); ok {
					body[len(body)-1] = &ast.ReturnStmt{
						Value: es.Expr,
						Line:  es.Line,
						Col:   es.Col,
					}
				}
			}
		}
	}

	if !p.expectCur(token.RPAREN) {
		return nil
	}

	return &ast.RouteDecl{
		Method:     method,
		Path:       path,
		Params:     params,
		ReturnType: retType,
		Effects:    effects,
		OnErrVar:   onErrVar,
		OnErrBody:  onErrBody,
		Body:       body,
		Line:       line,
		Col:        col,
	}
}

func (p *Parser) parseParams() []ast.Param {
	if !p.expectCur(token.LPAREN) {
		return nil
	}

	var params []ast.Param

	// Caso 1: (params ...) ou (params)
	if p.curTokenIs(token.PARAMS) {
		p.nextToken() // consome 'params'
		for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
			param := p.parseSingleParam()
			if param != nil {
				params = append(params, *param)
			} else {
				break
			}
		}
		p.expectCur(token.RPAREN)
		return params
	}

	// Caso 2: () - lista vazia compacta
	if p.curTokenIs(token.RPAREN) {
		p.nextToken() // consome ')'
		return params
	}

	// Caso 3: ((id int) (name str)) - lista compacta com parâmetros
	for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
		param := p.parseSingleParam()
		if param != nil {
			params = append(params, *param)
		} else {
			break
		}
	}
	p.expectCur(token.RPAREN)
	return params
}

func (p *Parser) parseSingleParam() *ast.Param {
	if !p.expectCur(token.LPAREN) {
		return nil
	}
	pLine, pCol := p.curToken.Line, p.curToken.Col
	if !p.curTokenIs(token.IDENT) {
		p.addError("Esperado nome do parâmetro", "E_EXPECTED_PARAM_NAME")
		return nil
	}
	pName := p.curToken.Literal
	p.nextToken()

	pType := p.parseType()
	if pType == nil {
		return nil
	}

	if !p.expectCur(token.RPAREN) {
		return nil
	}
	return &ast.Param{Name: pName, Type: pType, Line: pLine, Col: pCol}
}

func (p *Parser) parseReturns() ast.Type {
	if p.curTokenIs(token.LPAREN) && p.peekTokenIs(token.RETURNS) {
		p.nextToken() // consome '('
		p.nextToken() // consome 'returns'
		retType := p.parseType()
		p.expectCur(token.RPAREN)
		return retType
	}
	return p.parseType()
}

func (p *Parser) parseEffects() []string {
	if !p.curTokenIs(token.LPAREN) || !p.peekTokenIs(token.EFFECTS) {
		return nil
	}
	p.nextToken() // consome '('
	p.nextToken() // consome 'effects'

	var effects []string
	for p.curTokenIs(token.IDENT) || p.curTokenIs(token.SPAWN) {
		effects = append(effects, p.curToken.Literal)
		p.nextToken()
	}

	p.expectCur(token.RPAREN)
	return effects
}

func (p *Parser) parseBody() []ast.Stmt {
	if !p.expectCur(token.LPAREN) {
		return nil
	}
	if !p.expectCur(token.BODY) {
		return nil
	}

	var stmts []ast.Stmt
	for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}

	p.expectCur(token.RPAREN)
	return stmts
}
