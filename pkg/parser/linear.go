package parser

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"liaf/pkg/ast"
	"liaf/pkg/token"
)

func (p *Parser) parseLinearModule() *ast.Module {
	startLine, startCol := p.curToken.Line, p.curToken.Col
	modName := "main"
	if p.file != "" {
		base := filepath.Base(p.file)
		ext := filepath.Ext(base)
		if ext != "" {
			base = strings.TrimSuffix(base, ext)
		}
		if base != "" && base != "." {
			modName = base
		}
	}

	if p.curTokenIs(token.MODULE) {
		startLine, startCol = p.curToken.Line, p.curToken.Col
		p.nextToken() // consome 'module'
		if !p.curTokenIs(token.IDENT) {
			p.addError("Esperado identificador para o nome do módulo", "E_EXPECTED_MODULE_NAME")
			return nil
		}
		modName = p.curToken.Literal
		p.nextToken()
	}

	mod := &ast.Module{
		Name: modName,
		Line: startLine,
		Col:  startCol,
	}

	for !p.curTokenIs(token.EOF) {
		decl := p.parseLinearTopLevel()
		if decl != nil {
			mod.Decls = append(mod.Decls, decl)
		} else {
			p.nextToken()
		}
	}

	return mod
}

func (p *Parser) parseLinearTopLevel() ast.TopLevel {
	switch p.curToken.Type {
	case token.IMPORT:
		return p.parseLinearImport()
	case token.STRUCT:
		return p.parseLinearStruct()
	case token.FN:
		return p.parseLinearFunc()
	case token.ROUTE:
		return p.parseLinearRoute()
	case token.WS_ROUTE:
		return p.parseLinearWSRoute()
	case token.LPAREN:
		return p.parseTopLevel()
	default:
		p.addError(fmt.Sprintf("Declaração inválida: esperado 'import', 'struct', 'fn', 'route' ou 'ws-route', mas obteve %q", p.curToken.Literal), "E_INVALID_TOPLEVEL")
		return nil
	}
}

func (p *Parser) parseLinearImport() *ast.ImportDecl {
	line, col := p.curToken.Line, p.curToken.Col
	p.nextToken() // consome 'import'

	if !p.curTokenIs(token.STRING) {
		p.addError("Esperada string para o caminho do import", "E_EXPECTED_IMPORT_PATH")
		return nil
	}
	path := p.curToken.Literal
	p.nextToken()

	return &ast.ImportDecl{Path: path, Line: line, Col: col}
}

func (p *Parser) parseLinearStruct() *ast.StructDecl {
	line, col := p.curToken.Line, p.curToken.Col
	p.nextToken() // consome 'struct'

	if !p.curTokenIs(token.IDENT) {
		p.addError("Esperado identificador para o nome da struct", "E_EXPECTED_STRUCT_NAME")
		return nil
	}
	name := p.curToken.Literal
	p.nextToken()

	var fields []ast.Field
	for !p.curTokenIs(token.END) && !p.curTokenIs(token.EOF) {
		fLine, fCol := p.curToken.Line, p.curToken.Col
		if !token.IsName(p.curToken) {
			p.addError("Esperado nome do campo da struct", "E_EXPECTED_FIELD_NAME")
			return nil
		}
		fName := p.curToken.Literal
		p.nextToken()

		if p.curTokenIs(token.COLON) {
			p.nextToken()
		}

		fType := p.parseType()
		if fType == nil {
			return nil
		}

		if p.curTokenIs(token.COMMA) {
			p.nextToken()
		}

		fields = append(fields, ast.Field{
			Name: fName,
			Type: fType,
			Line: fLine,
			Col:  fCol,
		})
	}

	if !p.expectCur(token.END) {
		return nil
	}

	return &ast.StructDecl{
		Name:   name,
		Fields: fields,
		Line:   line,
		Col:    col,
	}
}

func (p *Parser) parseLinearParams() []ast.Param {
	if !p.expectCur(token.LPAREN) {
		return nil
	}

	var params []ast.Param
	for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
		pLine, pCol := p.curToken.Line, p.curToken.Col
		if !p.curTokenIs(token.IDENT) {
			p.addError("Esperado nome do parâmetro", "E_EXPECTED_PARAM_NAME")
			return nil
		}
		pName := p.curToken.Literal
		p.nextToken()

		if p.curTokenIs(token.COLON) {
			p.nextToken()
		}

		pType := p.parseType()
		if pType == nil {
			return nil
		}

		params = append(params, ast.Param{
			Name: pName,
			Type: pType,
			Line: pLine,
			Col:  pCol,
		})

		if p.curTokenIs(token.COMMA) {
			p.nextToken()
		}
	}

	p.expectCur(token.RPAREN)
	return params
}

func (p *Parser) parseLinearEffects() []string {
	if !p.curTokenIs(token.EFFECTS) && !(p.curTokenIs(token.IDENT) && p.curToken.Literal == "effects") {
		return nil
	}
	p.nextToken() // consome 'effects'
	if !p.expectCur(token.LPAREN) {
		return nil
	}

	var effects []string
	for p.curTokenIs(token.IDENT) || p.curTokenIs(token.SPAWN) {
		effects = append(effects, p.curToken.Literal)
		p.nextToken()
		if p.curTokenIs(token.COMMA) {
			p.nextToken()
		}
	}

	p.expectCur(token.RPAREN)
	return effects
}

func (p *Parser) isTypeStart() bool {
	if p.curTokenIs(token.LPAREN) {
		switch p.peekToken.Literal {
		case "chan", "list", "map", "option", "result":
			return true
		}
		return false
	}
	if p.curTokenIs(token.IDENT) {
		switch p.curToken.Literal {
		case "effects", "on-err", "let", "set", "return", "if", "while", "for-range", "for-each", "break", "continue", "end", "db-transaction":
			return false
		default:
			return true
		}
	}
	return false
}

func (p *Parser) parseLinearFunc() *ast.FuncDecl {
	line, col := p.curToken.Line, p.curToken.Col
	p.nextToken() // consome 'fn'

	if !p.curTokenIs(token.IDENT) {
		p.addError("Esperado identificador para o nome da função", "E_EXPECTED_FN_NAME")
		return nil
	}
	fnName := p.curToken.Literal
	p.nextToken()

	params := p.parseLinearParams()

	var retType ast.Type
	if p.isTypeStart() {
		retType = p.parseType()
	} else {
		retType = &ast.PrimitiveType{Name: "void", Line: line, Col: col}
	}

	effects := p.parseLinearEffects()

	var onErrVar string
	var onErrBody []ast.Stmt
	if p.curTokenIs(token.ON_ERR) {
		p.nextToken() // consome 'on-err'
		if !p.curTokenIs(token.IDENT) {
			p.addError("Esperado identificador para a variável de erro em 'on-err'", "E_EXPECTED_ON_ERR_VAR")
			return nil
		}
		onErrVar = p.curToken.Literal
		p.nextToken()
		for !p.curTokenIs(token.END) && !p.curTokenIs(token.EOF) {
			stmt := p.parseLinearStatement()
			if stmt != nil {
				onErrBody = append(onErrBody, stmt)
			}
		}
		if retType != nil {
			if prim, ok := retType.(*ast.PrimitiveType); !ok || prim.Name != "void" {
				if len(onErrBody) > 0 {
					if es, ok := onErrBody[len(onErrBody)-1].(*ast.ExprStmt); ok {
						onErrBody[len(onErrBody)-1] = &ast.ReturnStmt{
							Value: es.Expr,
							Line:  es.Line,
							Col:   es.Col,
						}
					}
				}
			}
		}
		p.expectCur(token.END)
	}

	var body []ast.Stmt
	for !p.curTokenIs(token.END) && !p.curTokenIs(token.EOF) {
		stmt := p.parseLinearStatement()
		if stmt != nil {
			body = append(body, stmt)
		}
	}

	// Implicit return para tipo não-void
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

	p.expectCur(token.END)

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

func (p *Parser) parseLinearRoute() *ast.RouteDecl {
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

	params := p.parseLinearParams()

	var retType ast.Type
	if p.isTypeStart() {
		retType = p.parseType()
	} else {
		retType = &ast.PrimitiveType{Name: "void", Line: line, Col: col}
	}

	effects := p.parseLinearEffects()

	var onErrVar string
	var onErrBody []ast.Stmt
	if p.curTokenIs(token.ON_ERR) {
		p.nextToken() // consome 'on-err'
		if !p.curTokenIs(token.IDENT) {
			p.addError("Esperado identificador para a variável de erro em 'on-err'", "E_EXPECTED_ON_ERR_VAR")
			return nil
		}
		onErrVar = p.curToken.Literal
		p.nextToken()
		for !p.curTokenIs(token.END) && !p.curTokenIs(token.EOF) {
			stmt := p.parseLinearStatement()
			if stmt != nil {
				onErrBody = append(onErrBody, stmt)
			}
		}
		if retType != nil {
			if prim, ok := retType.(*ast.PrimitiveType); !ok || prim.Name != "void" {
				if len(onErrBody) > 0 {
					if es, ok := onErrBody[len(onErrBody)-1].(*ast.ExprStmt); ok {
						onErrBody[len(onErrBody)-1] = &ast.ReturnStmt{
							Value: es.Expr,
							Line:  es.Line,
							Col:   es.Col,
						}
					}
				}
			}
		}
		p.expectCur(token.END)
	}

	var body []ast.Stmt
	for !p.curTokenIs(token.END) && !p.curTokenIs(token.EOF) {
		stmt := p.parseLinearStatement()
		if stmt != nil {
			body = append(body, stmt)
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

	p.expectCur(token.END)

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

func (p *Parser) parseLinearWSRoute() *ast.WSRouteDecl {
	line, col := p.curToken.Line, p.curToken.Col
	p.nextToken() // consome 'ws-route'

	if !p.curTokenIs(token.STRING) {
		p.addError("Esperado path em string para 'ws-route'", "E_EXPECTED_ROUTE_PATH")
		return nil
	}
	path := p.curToken.Literal
	p.nextToken()

	params := p.parseLinearParams()
	effects := p.parseLinearEffects()

	decl := &ast.WSRouteDecl{
		Path:    path,
		Params:  params,
		Effects: effects,
		Line:    line,
		Col:     col,
	}

	for !p.curTokenIs(token.END) && !p.curTokenIs(token.EOF) {
		switch p.curToken.Type {
		case token.ON_ERR:
			p.nextToken()
			decl.OnErrVar = p.identifier()
			for !p.curTokenIs(token.END) && !p.curTokenIs(token.EOF) {
				s := p.parseLinearStatement()
				if s != nil {
					decl.OnErrBody = append(decl.OnErrBody, s)
				}
			}
			p.expectCur(token.END)
		case token.ON_OPEN:
			p.nextToken()
			for !p.curTokenIs(token.END) && !p.curTokenIs(token.EOF) {
				s := p.parseLinearStatement()
				if s != nil {
					decl.OnOpen = append(decl.OnOpen, s)
				}
			}
			p.expectCur(token.END)
		case token.ON_MESSAGE:
			p.nextToken()
			decl.MsgVar = p.identifier()
			for !p.curTokenIs(token.END) && !p.curTokenIs(token.EOF) {
				s := p.parseLinearStatement()
				if s != nil {
					decl.OnMessage = append(decl.OnMessage, s)
				}
			}
			p.expectCur(token.END)
		case token.ON_CLOSE:
			p.nextToken()
			for !p.curTokenIs(token.END) && !p.curTokenIs(token.EOF) {
				s := p.parseLinearStatement()
				if s != nil {
					decl.OnClose = append(decl.OnClose, s)
				}
			}
			p.expectCur(token.END)
		default:
			p.addError(fmt.Sprintf("Bloco inesperado em ws-route: %q", p.curToken.Literal), "E_UNEXPECTED_TOKEN")
			return nil
		}
	}

	p.expectCur(token.END)
	return decl
}

func (p *Parser) parseLinearStatement() ast.Stmt {
	line, col := p.curToken.Line, p.curToken.Col

	// Se começar com parênteses, pode ser uma S-expression statement legada/embutida
	if p.curTokenIs(token.LPAREN) {
		return p.parseStatement()
	}

	switch p.curToken.Type {
	case token.LET:
		return p.parseLinearLet(line, col)
	case token.SET:
		return p.parseLinearSet(line, col)
	case token.RETURN:
		return p.parseLinearReturn(line, col)
	case token.IF:
		return p.parseLinearIf(line, col)
	case token.WHILE:
		return p.parseLinearWhile(line, col)
	case token.FOR_RANGE:
		return p.parseLinearForRange(line, col)
	case token.FOR_EACH:
		return p.parseLinearForEach(line, col)
	case token.BREAK, token.CONTINUE:
		kind := p.curToken.Literal
		p.nextToken()
		return &ast.LoopControl{Kind: kind, Line: line, Col: col}
	case token.DB_TRANSACTION:
		return p.parseLinearDBTransaction(line, col)
	default:
		expr := p.parseLinearExpression(0)
		if expr != nil {
			return &ast.ExprStmt{Expr: expr, Line: line, Col: col}
		}
		return nil
	}
}

func (p *Parser) parseLinearLet(line, col int) ast.Stmt {
	p.nextToken() // consome 'let'

	if !p.curTokenIs(token.IDENT) {
		p.addError("Esperado identificador para a variável em 'let'", "E_EXPECTED_LET_NAME")
		return nil
	}
	name := p.curToken.Literal
	p.nextToken()

	var typ ast.Type
	if !p.curTokenIs(token.ASSIGN) {
		if p.curTokenIs(token.COLON) {
			p.nextToken()
		}
		typ = p.parseType()
	}

	if p.curTokenIs(token.ASSIGN) {
		p.nextToken() // consome '='
	}

	val := p.parseLinearExpression(0)
	return &ast.LetStmt{
		Name:  name,
		Type:  typ,
		Value: val,
		Line:  line,
		Col:   col,
	}
}

func (p *Parser) parseLinearSet(line, col int) ast.Stmt {
	p.nextToken() // consome 'set'

	if !p.curTokenIs(token.IDENT) {
		p.addError("Esperado identificador para a variável em 'set'", "E_EXPECTED_SET_NAME")
		return nil
	}
	name := p.curToken.Literal
	p.nextToken()

	if p.curTokenIs(token.ASSIGN) {
		p.nextToken() // consome '='
	}

	val := p.parseLinearExpression(0)
	return &ast.SetStmt{
		Name:  name,
		Value: val,
		Line:  line,
		Col:   col,
	}
}

func (p *Parser) parseLinearReturn(line, col int) ast.Stmt {
	p.nextToken() // consome 'return'

	if p.curTokenIs(token.END) || p.curTokenIs(token.ELSE) || p.curTokenIs(token.EOF) {
		return &ast.ReturnStmt{Value: nil, Line: line, Col: col}
	}

	val := p.parseLinearExpression(0)
	return &ast.ReturnStmt{Value: val, Line: line, Col: col}
}

func (p *Parser) parseLinearIf(line, col int) ast.Stmt {
	p.nextToken() // consome 'if'

	cond := p.parseLinearExpression(0)
	if p.curTokenIs(token.THEN) {
		p.nextToken()
	}

	var thenStmts []ast.Stmt
	for !p.curTokenIs(token.ELSE) && !p.curTokenIs(token.END) && !p.curTokenIs(token.EOF) {
		stmt := p.parseLinearStatement()
		if stmt != nil {
			thenStmts = append(thenStmts, stmt)
		}
	}

	var elseStmts []ast.Stmt
	if p.curTokenIs(token.ELSE) {
		p.nextToken()
		for !p.curTokenIs(token.END) && !p.curTokenIs(token.EOF) {
			stmt := p.parseLinearStatement()
			if stmt != nil {
				elseStmts = append(elseStmts, stmt)
			}
		}
	}

	p.expectCur(token.END)

	return &ast.IfStmt{
		Condition: cond,
		Then:      thenStmts,
		Else:      elseStmts,
		Line:      line,
		Col:       col,
	}
}

func (p *Parser) parseLinearWhile(line, col int) ast.Stmt {
	p.nextToken() // consome 'while'
	cond := p.parseLinearExpression(0)

	var body []ast.Stmt
	for !p.curTokenIs(token.END) && !p.curTokenIs(token.EOF) {
		s := p.parseLinearStatement()
		if s != nil {
			body = append(body, s)
		}
	}

	p.expectCur(token.END)
	return &ast.LoopStmt{
		Kind:      "while",
		Condition: cond,
		Body:      body,
		Line:      line,
		Col:       col,
	}
}

func (p *Parser) parseLinearForRange(line, col int) ast.Stmt {
	p.nextToken() // consome 'for-range'
	if !p.curTokenIs(token.IDENT) {
		p.addError("Esperado nome da variável em for-range", "E_EXPECTED_LOOP_VAR")
		return nil
	}
	name := p.curToken.Literal
	p.nextToken()

	start := p.parseLinearExpression(0)
	endExpr := p.parseLinearExpression(0)

	var body []ast.Stmt
	for !p.curTokenIs(token.END) && !p.curTokenIs(token.EOF) {
		s := p.parseLinearStatement()
		if s != nil {
			body = append(body, s)
		}
	}

	p.expectCur(token.END)
	return &ast.LoopStmt{
		Kind:  "for-range",
		Name:  name,
		Start: start,
		End:   endExpr,
		Body:  body,
		Line:  line,
		Col:   col,
	}
}

func (p *Parser) parseLinearForEach(line, col int) ast.Stmt {
	p.nextToken() // consome 'for-each'
	if !p.curTokenIs(token.IDENT) {
		p.addError("Esperado nome da variável em for-each", "E_EXPECTED_LOOP_VAR")
		return nil
	}
	name := p.curToken.Literal
	p.nextToken()

	coll := p.parseLinearExpression(0)

	var body []ast.Stmt
	for !p.curTokenIs(token.END) && !p.curTokenIs(token.EOF) {
		s := p.parseLinearStatement()
		if s != nil {
			body = append(body, s)
		}
	}

	p.expectCur(token.END)
	return &ast.LoopStmt{
		Kind:       "for-each",
		Name:       name,
		Collection: coll,
		Body:       body,
		Line:       line,
		Col:        col,
	}
}

func (p *Parser) parseLinearDBTransaction(line, col int) ast.Stmt {
	p.nextToken() // consome 'db-transaction'
	if !p.curTokenIs(token.IDENT) {
		p.addError("Esperado identificador da conexão em db-transaction", "E_EXPECTED_CONN_NAME")
		return nil
	}
	conn := p.curToken.Literal
	p.nextToken()

	var body []ast.Stmt
	for !p.curTokenIs(token.END) && !p.curTokenIs(token.EOF) {
		s := p.parseLinearStatement()
		if s != nil {
			body = append(body, s)
		}
	}

	p.expectCur(token.END)
	return &ast.DBTransactionStmt{
		Conn: conn,
		Body: body,
		Line: line,
		Col:  col,
	}
}

func linearPrecedence(tok token.TokenType) int {
	switch tok {
	case token.OR:
		return 1
	case token.AND:
		return 2
	case token.EQ, token.NEQ, token.LT, token.LTE, token.GT, token.GTE:
		return 3
	case token.ADD, token.SUB:
		return 4
	case token.MUL, token.DIV:
		return 5
	default:
		return 0
	}
}

func (p *Parser) parseLinearExpression(minPrec int) ast.Expr {
	left := p.parseLinearPrimary()
	if left == nil {
		return nil
	}

	for {
		prec := linearPrecedence(p.curToken.Type)
		if prec <= minPrec {
			break
		}

		opTok := p.curToken
		p.nextToken()

		right := p.parseLinearExpression(prec)
		if right == nil {
			return nil
		}

		left = &ast.BinaryOpExpr{
			Op:    opTok.Literal,
			Left:  left,
			Right: right,
			Line:  opTok.Line,
			Col:   opTok.Col,
		}
	}

	return left
}

func (p *Parser) parseLinearPrimary() ast.Expr {
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

	case token.PATH:
		expr := p.parsePath()
		p.nextToken()
		return expr

	case token.TRY:
		p.nextToken() // consome 'try'
		sub := p.parseLinearPrimary()
		return &ast.TryExpr{Expr: sub, Line: line, Col: col}

	case token.NOT:
		p.nextToken() // consome 'not' ou '!'
		sub := p.parseLinearPrimary()
		return &ast.CallExpr{Func: "not", Args: []ast.Expr{sub}, Line: line, Col: col}

	case token.IDENT:
		name := p.curToken.Literal
		p.nextToken()

		// Construtor: new Tipo(arg1, arg2, ...)
		if name == "new" && p.curTokenIs(token.IDENT) {
			typeName := p.curToken.Literal
			p.nextToken()
			var args []ast.Expr
			args = append(args, &ast.IdentExpr{Name: typeName, Line: line, Col: col})
			if p.curTokenIs(token.LPAREN) {
				p.nextToken() // consome '('
				for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
					arg := p.parseLinearExpression(0)
					if arg != nil {
						args = append(args, arg)
					}
					if p.curTokenIs(token.COMMA) {
						p.nextToken()
					}
				}
				p.expectCur(token.RPAREN)
			}
			return &ast.CallExpr{Func: "new", Args: args, Line: line, Col: col}
		}

		// Chamada de função: nome(arg1, arg2, ...)
		if p.curTokenIs(token.LPAREN) {
			p.nextToken() // consome '('
			var args []ast.Expr
			for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
				arg := p.parseLinearExpression(0)
				if arg != nil {
					args = append(args, arg)
				}
				if p.curTokenIs(token.COMMA) {
					p.nextToken()
				}
			}
			p.expectCur(token.RPAREN)
			return &ast.CallExpr{Func: name, Args: args, Line: line, Col: col}
		}

		return &ast.IdentExpr{Name: name, Line: line, Col: col}

	case token.LPAREN:
		p.nextToken() // consome '('
		// Verifica se é uma S-expression interna
		if token.IsOperator(p.curToken.Type) || p.curTokenIs(token.TRY) || p.curTokenIs(token.CALL) {
			p.curToken.Line = line
			p.curToken.Col = col
			// Re-parseia como S-expr
		}
		expr := p.parseLinearExpression(0)
		p.expectCur(token.RPAREN)
		return expr

	default:
		p.addError(fmt.Sprintf("Token inesperado ao iniciar expressão linear: %q (%s)", p.curToken.Literal, p.curToken.Type), "E_UNEXPECTED_TOKEN")
		return nil
	}
}
