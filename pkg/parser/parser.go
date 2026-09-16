package parser

import (
	"fmt"
	"strconv"

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

// ParseModule analisa o arquivo completo, exigindo a declaração do módulo
func (p *Parser) ParseModule() (result *ast.Module) {
	defer func() {
		if v := recover(); v != nil {
			if _, ok := v.(parseAbort); !ok {
				panic(v)
			}
			result = nil
		}
	}()
	if !p.curTokenIs(token.LPAREN) {
		p.addError("Todo programa LIAF v0.2 deve iniciar com '('", "E_EXPECTED_LPAREN")
		return nil
	}
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

	if !p.curTokenIs(token.EOF) {
		p.addError(fmt.Sprintf("Tokens excedentes após o fechamento do módulo: %q", p.curToken.Literal), "E_TRAILING_TOKENS")
	}

	return mod
}

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
	default:
		p.addError(fmt.Sprintf("Declaração inválida: esperado 'import', 'struct', 'fn' ou 'route', mas obteve %q", p.curToken.Literal), "E_INVALID_TOPLEVEL")
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

	if !p.expectCur(token.LPAREN) {
		return nil
	}
	if !p.expectCur(token.FIELDS) {
		return nil
	}

	var fields []ast.Field
	for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
		f := p.parseField()
		if f != nil {
			fields = append(fields, *f)
		}
	}
	if !p.expectCur(token.RPAREN) {
		return nil
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
	if !p.curTokenIs(token.IDENT) {
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

	body := p.parseBody()

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

	body := p.parseBody()

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
	if !p.expectCur(token.PARAMS) {
		return nil
	}

	var params []ast.Param
	for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
		if !p.expectCur(token.LPAREN) {
			break
		}
		pLine, pCol := p.curToken.Line, p.curToken.Col
		if !p.curTokenIs(token.IDENT) {
			p.addError("Esperado nome do parâmetro", "E_EXPECTED_PARAM_NAME")
			break
		}
		pName := p.curToken.Literal
		p.nextToken()

		pType := p.parseType()
		if pType == nil {
			break
		}

		if !p.expectCur(token.RPAREN) {
			break
		}
		params = append(params, ast.Param{Name: pName, Type: pType, Line: pLine, Col: pCol})
	}

	p.expectCur(token.RPAREN)
	return params
}

func (p *Parser) parseReturns() ast.Type {
	if !p.expectCur(token.LPAREN) {
		return nil
	}
	if !p.expectCur(token.RETURNS) {
		return nil
	}

	retType := p.parseType()

	p.expectCur(token.RPAREN)
	return retType
}

func (p *Parser) parseEffects() []string {
	if !p.expectCur(token.LPAREN) {
		return nil
	}
	if !p.expectCur(token.EFFECTS) {
		return nil
	}

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

func (p *Parser) parseStatement() ast.Stmt {
	if !p.expectCur(token.LPAREN) {
		return nil
	}
	line, col := p.curToken.Line, p.curToken.Col

	switch p.curToken.Type {
	case token.WHILE, token.FOR_RANGE, token.FOR_EACH:
		return p.parseLoop(line, col)
	case token.BREAK, token.CONTINUE:
		kind := p.curToken.Literal
		p.nextToken()
		p.expectCur(token.RPAREN)
		return &ast.LoopControl{Kind: kind, Line: line, Col: col}
	case token.MATCH:
		return p.parseMatch(line, col)
	case token.LET:
		return p.parseLetStmt(line, col)
	case token.SET:
		return p.parseSetStmt(line, col)
	case token.RETURN:
		return p.parseReturnStmt(line, col)
	case token.IF:
		return p.parseIfStmt(line, col)
	case token.SPAWN:
		return p.parseSpawnStmt(line, col)
	case token.SEND:
		return p.parseSendStmt(line, col)
	case token.DO:
		return p.parseExprStmt(line, col)
	case token.TRY:
		p.nextToken()
		sub := p.parseExpression()
		if !p.expectCur(token.RPAREN) {
			return nil
		}
		return &ast.ExprStmt{Expr: &ast.TryExpr{Expr: sub, Line: line, Col: col}, Line: line, Col: col}
	case token.CALL:
		p.nextToken()
		if !p.curTokenIs(token.IDENT) {
			p.addError("Esperado identificador de função em call", "E_EXPECTED_CALL_IDENT")
			return nil
		}
		fnName := p.curToken.Literal
		p.nextToken()
		var args []ast.Expr
		for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
			arg := p.parseExpression()
			if arg != nil {
				args = append(args, arg)
			}
		}
		if !p.expectCur(token.RPAREN) {
			return nil
		}
		return &ast.ExprStmt{Expr: &ast.CallExpr{Func: fnName, Args: args, HasCall: true, Line: line, Col: col}, Line: line, Col: col}
	case token.IDENT:
		fnName := p.curToken.Literal
		p.nextToken()
		var args []ast.Expr
		for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
			arg := p.parseExpression()
			if arg != nil {
				args = append(args, arg)
			}
		}
		if !p.expectCur(token.RPAREN) {
			return nil
		}
		return &ast.ExprStmt{Expr: &ast.CallExpr{Func: fnName, Args: args, HasCall: false, Line: line, Col: col}, Line: line, Col: col}
	default:
		p.addError(fmt.Sprintf("Comando desconhecido: %q", p.curToken.Literal), "E_UNKNOWN_STATEMENT")
		return nil
	}
}

func (p *Parser) parseLetStmt(line, col int) *ast.LetStmt {
	p.nextToken()

	if !p.curTokenIs(token.IDENT) {
		p.addError("Esperado identificador no let", "E_EXPECTED_LET_IDENT")
		return nil
	}
	name := p.curToken.Literal
	p.nextToken()

	varType := p.parseType()
	if varType == nil {
		return nil
	}

	val := p.parseExpression()
	if val == nil {
		return nil
	}

	if !p.expectCur(token.RPAREN) {
		return nil
	}

	return &ast.LetStmt{Name: name, Type: varType, Value: val, Line: line, Col: col}
}

func (p *Parser) parseSetStmt(line, col int) *ast.SetStmt {
	p.nextToken()

	if !p.curTokenIs(token.IDENT) {
		p.addError("Esperado identificador no set", "E_EXPECTED_SET_IDENT")
		return nil
	}
	name := p.curToken.Literal
	p.nextToken()

	val := p.parseExpression()
	if val == nil {
		return nil
	}

	if !p.expectCur(token.RPAREN) {
		return nil
	}

	return &ast.SetStmt{Name: name, Value: val, Line: line, Col: col}
}

func (p *Parser) parseReturnStmt(line, col int) *ast.ReturnStmt {
	p.nextToken()

	var val ast.Expr
	if !p.curTokenIs(token.RPAREN) {
		val = p.parseExpression()
	}

	if !p.expectCur(token.RPAREN) {
		return nil
	}

	return &ast.ReturnStmt{Value: val, Line: line, Col: col}
}

func (p *Parser) parseIfStmt(line, col int) *ast.IfStmt {
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

	var thenStmts []ast.Stmt
	for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
		st := p.parseStatement()
		if st != nil {
			thenStmts = append(thenStmts, st)
		}
	}
	if !p.expectCur(token.RPAREN) {
		return nil
	}

	var elseStmts []ast.Stmt
	if p.curTokenIs(token.LPAREN) && p.peekTokenIs(token.ELSE) {
		p.nextToken()
		p.nextToken()
		for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
			st := p.parseStatement()
			if st != nil {
				elseStmts = append(elseStmts, st)
			}
		}
		if !p.expectCur(token.RPAREN) {
			return nil
		}
	}

	if !p.expectCur(token.RPAREN) {
		return nil
	}

	return &ast.IfStmt{
		Condition: cond,
		Then:      thenStmts,
		Else:      elseStmts,
		Line:      line,
		Col:       col,
	}
}

func (p *Parser) parseSpawnStmt(line, col int) *ast.SpawnStmt {
	p.nextToken()

	expr := p.parseExpression()
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		p.addError("spawn requer uma expressão 'call'", "E_EXPECTED_CALL_IN_SPAWN")
		return nil
	}

	if !p.expectCur(token.RPAREN) {
		return nil
	}

	return &ast.SpawnStmt{Call: call, Line: line, Col: col}
}

func (p *Parser) parseSendStmt(line, col int) *ast.SendStmt {
	p.nextToken()

	ch := p.parseExpression()
	val := p.parseExpression()

	if !p.expectCur(token.RPAREN) {
		return nil
	}

	return &ast.SendStmt{Channel: ch, Value: val, Line: line, Col: col}
}

func (p *Parser) parseExprStmt(line, col int) *ast.ExprStmt {
	p.nextToken()

	expr := p.parseExpression()
	if expr == nil {
		return nil
	}

	if !p.expectCur(token.RPAREN) {
		return nil
	}

	return &ast.ExprStmt{Expr: expr, Line: line, Col: col}
}

func (p *Parser) parseExpression() ast.Expr {
	line, col := p.curToken.Line, p.curToken.Col

	switch p.curToken.Type {
	case token.INT:
		val, err := strconv.ParseInt(p.curToken.Literal, 10, 64)
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

			var args []ast.Expr
			for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
				arg := p.parseExpression()
				if arg != nil {
					args = append(args, arg)
				}
			}
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

			if !p.expectCur(token.RPAREN) {
				return nil
			}
			return &ast.BinaryOpExpr{Op: op, Left: left, Right: right, Line: innerLine, Col: innerCol}
		}

		if p.curTokenIs(token.RECV) {
			p.nextToken()
			ch := p.parseExpression()
			if !p.expectCur(token.RPAREN) {
				return nil
			}
			return &ast.RecvExpr{Channel: ch, Line: innerLine, Col: innerCol}
		}

		// Chamada direta de função: (fn-name arg1 arg2 ...)
		if p.curTokenIs(token.IDENT) {
			fnName := p.curToken.Literal
			p.nextToken()

			var args []ast.Expr
			for !p.curTokenIs(token.RPAREN) && !p.curTokenIs(token.EOF) {
				arg := p.parseExpression()
				if arg != nil {
					args = append(args, arg)
				}
			}
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
