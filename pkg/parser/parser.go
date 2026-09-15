package parser

import (
	"fmt"
	"strconv"

	"liaf/pkg/ast"
	"liaf/pkg/diagnostic"
	"liaf/pkg/lexer"
)

type Parser struct {
	l         *lexer.Lexer
	curToken  lexer.Token
	peekToken lexer.Token
	errors    []diagnostic.Diagnostic
	filename  string
}

func New(l *lexer.Lexer, filename string) *Parser {
	p := &Parser{
		l:        l,
		filename: filename,
	}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) Errors() []diagnostic.Diagnostic {
	return p.errors
}

func (p *Parser) addDiagnostic(code, node, msg, expected, received, patch string, line, col int) {
	p.errors = append(p.errors, diagnostic.Diagnostic{
		Code:           code,
		File:           p.filename,
		Line:           line,
		Col:            col,
		Node:           node,
		Message:        msg,
		Expected:       expected,
		Received:       received,
		SuggestedPatch: patch,
	})
}

func (p *Parser) ParseProgram() *ast.Program {
	prog := &ast.Program{Decls: []ast.TopLevelDecl{}}

	for p.curToken.Type != lexer.TOKEN_EOF {
		if p.curToken.Type == lexer.TOKEN_LBRACKET {
			switch p.peekToken.Type {
			case lexer.TOKEN_FN:
				fn := p.parseFuncDecl()
				if fn != nil {
					prog.Decls = append(prog.Decls, fn)
				}
			case lexer.TOKEN_STRUCT:
				st := p.parseStructDecl()
				if st != nil {
					prog.Decls = append(prog.Decls, st)
				}
			default:
				p.addDiagnostic(
					"INVALID_TOP_LEVEL_DECL",
					"Program",
					fmt.Sprintf("Esperado 'fn' ou 'struct' após '[', recebido '%s'", p.peekToken.Literal),
					"fn | struct",
					p.peekToken.Literal,
					"[fn ...]",
					p.curToken.Line,
					p.curToken.Col,
				)
				p.nextToken()
			}
		} else {
			p.addDiagnostic(
				"UNEXPECTED_TOKEN",
				"Program",
				fmt.Sprintf("Declaração de nível superior deve começar com '[', recebido '%s'", p.curToken.Literal),
				"[",
				p.curToken.Literal,
				"[fn ...",
				p.curToken.Line,
				p.curToken.Col,
			)
			p.nextToken()
		}
	}

	return prog
}

// [fn name (p1: t1 p2: t2) -> (ret_t) ... /fn name]
func (p *Parser) parseFuncDecl() *ast.FuncDecl {
	line, col := p.curToken.Line, p.curToken.Col
	p.nextToken() // consome '['
	p.nextToken() // consome 'fn'

	if p.curToken.Type != lexer.TOKEN_IDENT {
		p.addDiagnostic("EXPECTED_IDENTIFIER", "FuncDecl", "Nome da função esperado após 'fn'", "Identifier", p.curToken.Literal, "my_func", p.curToken.Line, p.curToken.Col)
		return nil
	}
	fnName := p.curToken.Literal
	p.nextToken() // consome o nome

	// Parâmetros: (p1: t1 p2: t2)
	params := p.parseParams()

	// Retorno opcional -> (ret_t)
	retType := "void"
	if p.curToken.Type == lexer.TOKEN_ARROW {
		p.nextToken() // consome '->'
		if p.curToken.Type == lexer.TOKEN_LPAREN {
			p.nextToken() // consome '('
			retType = p.parseType()
			if p.curToken.Type == lexer.TOKEN_RPAREN {
				p.nextToken()
			}
		}
	}

	// Corpo da função
	var body []ast.Stmt
	for {
		if p.curToken.Type == lexer.TOKEN_EOF {
			p.addDiagnostic("UNCLOSED_BLOCK", "FuncDecl", fmt.Sprintf("Bloco da função '%s' não foi fechado antes do fim do arquivo", fnName), fmt.Sprintf("/fn %s]", fnName), "EOF", fmt.Sprintf("/fn %s]", fnName), p.curToken.Line, p.curToken.Col)
			break
		}

		// Checar se é o fechamento: /fn name]
		if p.curToken.Type == lexer.TOKEN_SLASH {
			p.nextToken() // consome '/'
			if p.curToken.Type == lexer.TOKEN_FN {
				p.nextToken() // consome 'fn'
				closedName := p.curToken.Literal
				p.nextToken() // consome nome
				if closedName != fnName {
					p.addDiagnostic(
						"TAG_NAME_MISMATCH",
						"FuncDecl",
						fmt.Sprintf("Nome no fechamento '/fn %s' não corresponde a '/fn %s'", closedName, fnName),
						fnName,
						closedName,
						fmt.Sprintf("/fn %s]", fnName),
						p.curToken.Line,
						p.curToken.Col,
					)
				}
				if p.curToken.Type == lexer.TOKEN_RBRACKET {
					p.nextToken() // consome ']'
				} else {
					p.addDiagnostic("EXPECTED_RBRACKET", "FuncDecl", "Esperado ']' ao fechar bloco de função", "]", p.curToken.Literal, "]", p.curToken.Line, p.curToken.Col)
				}
				break
			}
		}

		stmt := p.parseStmt()
		if stmt != nil {
			body = append(body, stmt)
		}
	}

	return &ast.FuncDecl{
		Name:       fnName,
		Params:     params,
		ReturnType: retType,
		Body:       body,
		Line:       line,
		Col:        col,
	}
}

// [struct Name f1: t1 f2: t2 /struct Name]
func (p *Parser) parseStructDecl() *ast.StructDecl {
	line, col := p.curToken.Line, p.curToken.Col
	p.nextToken() // consome '['
	p.nextToken() // consome 'struct'

	if p.curToken.Type != lexer.TOKEN_IDENT {
		p.addDiagnostic("EXPECTED_IDENTIFIER", "StructDecl", "Nome da struct esperado", "Identifier", p.curToken.Literal, "MyStruct", p.curToken.Line, p.curToken.Col)
		return nil
	}
	stName := p.curToken.Literal
	p.nextToken()

	var fields []ast.Param
	for {
		if p.curToken.Type == lexer.TOKEN_EOF {
			break
		}
		if p.curToken.Type == lexer.TOKEN_SLASH {
			p.nextToken() // '/'
			if p.curToken.Type == lexer.TOKEN_STRUCT {
				p.nextToken() // 'struct'
				closedName := p.curToken.Literal
				p.nextToken() // nome
				if closedName != stName {
					p.addDiagnostic("TAG_NAME_MISMATCH", "StructDecl", fmt.Sprintf("Nome de fechamento '/struct %s' não corresponde a '/struct %s'", closedName, stName), stName, closedName, fmt.Sprintf("/struct %s]", stName), p.curToken.Line, p.curToken.Col)
				}
				if p.curToken.Type == lexer.TOKEN_RBRACKET {
					p.nextToken()
				}
				break
			}
		}

		if p.curToken.Type == lexer.TOKEN_IDENT && p.peekToken.Type == lexer.TOKEN_COLON {
			fName := p.curToken.Literal
			p.nextToken() // ident
			p.nextToken() // ':'
			fType := p.parseType()
			fields = append(fields, ast.Param{Name: fName, Type: fType})
		} else {
			p.nextToken()
		}
	}

	return &ast.StructDecl{
		Name:   stName,
		Fields: fields,
		Line:   line,
		Col:    col,
	}
}

func (p *Parser) parseParams() []ast.Param {
	var params []ast.Param
	if p.curToken.Type != lexer.TOKEN_LPAREN {
		return params
	}
	p.nextToken() // consome '('

	for p.curToken.Type != lexer.TOKEN_RPAREN && p.curToken.Type != lexer.TOKEN_EOF {
		if p.curToken.Type == lexer.TOKEN_IDENT {
			pName := p.curToken.Literal
			p.nextToken()
			if p.curToken.Type == lexer.TOKEN_COLON {
				p.nextToken()
				pType := p.parseType()
				params = append(params, ast.Param{Name: pName, Type: pType})
			}
		} else {
			p.nextToken()
		}
	}

	if p.curToken.Type == lexer.TOKEN_RPAREN {
		p.nextToken() // consome ')'
	}
	return params
}

func (p *Parser) parseStmt() ast.Stmt {
	if p.curToken.Type == lexer.TOKEN_LBRACKET {
		switch p.peekToken.Type {
		case lexer.TOKEN_LET:
			return p.parseLetStmt()
		case lexer.TOKEN_RETURN:
			return p.parseReturnStmt()
		case lexer.TOKEN_SPAWN:
			return p.parseSpawnStmt()
		case lexer.TOKEN_SEND:
			return p.parseSendStmt()
		case lexer.TOKEN_IF:
			return p.parseIfStmt()
		case lexer.TOKEN_RECV:
			// [recv ch] as a statement
			line, col := p.curToken.Line, p.curToken.Col
			p.nextToken() // '['
			p.nextToken() // 'recv'
			chExpr := p.parseExpr()
			if p.curToken.Type == lexer.TOKEN_RBRACKET {
				p.nextToken()
			}
			return &ast.RecvExpr{Channel: chExpr, Line: line, Col: col}
		default:
			// Pode ser expressão dentro de colchetes ou erro
		}
	} else if p.curToken.Type == lexer.TOKEN_LPAREN {
		expr := p.parseExpr()
		line, col := expr.Pos()
		return &ast.ExprStmt{Expression: expr, Line: line, Col: col}
	}

	p.nextToken()
	return nil
}

// [let x: int 10]
func (p *Parser) parseLetStmt() *ast.LetStmt {
	line, col := p.curToken.Line, p.curToken.Col
	p.nextToken() // '['
	p.nextToken() // 'let'

	name := p.curToken.Literal
	p.nextToken() // name

	var typeName string
	if p.curToken.Type == lexer.TOKEN_COLON {
		p.nextToken() // ':'
		typeName = p.parseType()
	}

	val := p.parseExpr()

	if p.curToken.Type == lexer.TOKEN_RBRACKET {
		p.nextToken() // ']'
	} else {
		p.addDiagnostic("EXPECTED_RBRACKET", "LetStmt", "Esperado ']' ao final de [let ...]", "]", p.curToken.Literal, "]", p.curToken.Line, p.curToken.Col)
	}

	return &ast.LetStmt{
		Name:  name,
		Type:  typeName,
		Value: val,
		Line:  line,
		Col:   col,
	}
}

// [return expr]
func (p *Parser) parseReturnStmt() *ast.ReturnStmt {
	line, col := p.curToken.Line, p.curToken.Col
	p.nextToken() // '['
	p.nextToken() // 'return'

	var val ast.Expr
	if p.curToken.Type != lexer.TOKEN_RBRACKET {
		val = p.parseExpr()
	}

	if p.curToken.Type == lexer.TOKEN_RBRACKET {
		p.nextToken() // ']'
	}

	return &ast.ReturnStmt{
		Value: val,
		Line:  line,
		Col:   col,
	}
}

// [spawn (fn_name args...)]
func (p *Parser) parseSpawnStmt() *ast.SpawnStmt {
	line, col := p.curToken.Line, p.curToken.Col
	p.nextToken() // '['
	p.nextToken() // 'spawn'

	expr := p.parseExpr()
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		p.addDiagnostic("INVALID_SPAWN", "SpawnStmt", "'spawn' requer uma chamada de função entre parênteses", "(func args...)", p.curToken.Literal, "(worker ch)", line, col)
	}

	if p.curToken.Type == lexer.TOKEN_RBRACKET {
		p.nextToken() // ']'
	}

	return &ast.SpawnStmt{
		Call: call,
		Line: line,
		Col:  col,
	}
}

// [send ch val]
func (p *Parser) parseSendStmt() *ast.SendStmt {
	line, col := p.curToken.Line, p.curToken.Col
	p.nextToken() // '['
	p.nextToken() // 'send'

	ch := p.parseExpr()
	val := p.parseExpr()

	if p.curToken.Type == lexer.TOKEN_RBRACKET {
		p.nextToken() // ']'
	}

	return &ast.SendStmt{
		Channel: ch,
		Value:   val,
		Line:    line,
		Col:     col,
	}
}

// [if cond ... [else ... /else] /if]
func (p *Parser) parseIfStmt() *ast.IfStmt {
	line, col := p.curToken.Line, p.curToken.Col
	p.nextToken() // '['
	p.nextToken() // 'if'

	cond := p.parseExpr()
	var thenBody []ast.Stmt
	var elseBody []ast.Stmt

	for {
		if p.curToken.Type == lexer.TOKEN_EOF {
			p.addDiagnostic("UNCLOSED_BLOCK", "IfStmt", "Bloco 'if' não foi fechado com /if]", "/if]", "EOF", "/if]", p.curToken.Line, p.curToken.Col)
			break
		}

		if p.curToken.Type == lexer.TOKEN_SLASH && p.peekToken.Type == lexer.TOKEN_IF {
			p.nextToken() // '/'
			p.nextToken() // 'if'
			if p.curToken.Type == lexer.TOKEN_RBRACKET {
				p.nextToken()
			}
			break
		}

		if p.curToken.Type == lexer.TOKEN_LBRACKET && p.peekToken.Type == lexer.TOKEN_ELSE {
			p.nextToken() // '['
			p.nextToken() // 'else'
			for {
				if p.curToken.Type == lexer.TOKEN_EOF {
					break
				}
				if p.curToken.Type == lexer.TOKEN_SLASH && p.peekToken.Type == lexer.TOKEN_ELSE {
					p.nextToken() // '/'
					p.nextToken() // 'else'
					if p.curToken.Type == lexer.TOKEN_RBRACKET {
						p.nextToken()
					}
					break
				}
				stmt := p.parseStmt()
				if stmt != nil {
					elseBody = append(elseBody, stmt)
				}
			}
			continue
		}

		stmt := p.parseStmt()
		if stmt != nil {
			thenBody = append(thenBody, stmt)
		}
	}

	return &ast.IfStmt{
		Condition: cond,
		ThenBody:  thenBody,
		ElseBody:  elseBody,
		Line:      line,
		Col:       col,
	}
}

func (p *Parser) parseExpr() ast.Expr {
	line, col := p.curToken.Line, p.curToken.Col

	switch p.curToken.Type {
	case lexer.TOKEN_INT:
		val, _ := strconv.ParseInt(p.curToken.Literal, 10, 64)
		p.nextToken()
		return &ast.IntLiteral{Value: val, Line: line, Col: col}
	case lexer.TOKEN_FLOAT:
		val, _ := strconv.ParseFloat(p.curToken.Literal, 64)
		p.nextToken()
		return &ast.FloatLiteral{Value: val, Line: line, Col: col}
	case lexer.TOKEN_STRING:
		val := p.curToken.Literal
		p.nextToken()
		return &ast.StringLiteral{Value: val, Line: line, Col: col}
	case lexer.TOKEN_BOOL:
		val := p.curToken.Literal == "true"
		p.nextToken()
		return &ast.BoolLiteral{Value: val, Line: line, Col: col}
	case lexer.TOKEN_IDENT:
		name := p.curToken.Literal
		p.nextToken()
		return &ast.IdentifierExpr{Name: name, Line: line, Col: col}
	case lexer.TOKEN_LBRACKET:
		if p.peekToken.Type == lexer.TOKEN_RECV {
			p.nextToken() // '['
			p.nextToken() // 'recv'
			chExpr := p.parseExpr()
			if p.curToken.Type == lexer.TOKEN_RBRACKET {
				p.nextToken()
			}
			return &ast.RecvExpr{Channel: chExpr, Line: line, Col: col}
		}
	case lexer.TOKEN_LPAREN:
		p.nextToken() // consome '('
		opOrFn := p.curToken.Literal
		p.nextToken()

		// Operadores binários prefixados
		if isBinaryOp(opOrFn) {
			left := p.parseExpr()
			right := p.parseExpr()
			if p.curToken.Type == lexer.TOKEN_RPAREN {
				p.nextToken() // ')'
			}
			return &ast.BinaryOpExpr{
				Op:    opOrFn,
				Left:  left,
				Right: right,
				Line:  line,
				Col:   col,
			}
		}

		// Chamada de função genérica
		var args []ast.Expr
		for p.curToken.Type != lexer.TOKEN_RPAREN && p.curToken.Type != lexer.TOKEN_EOF {
			args = append(args, p.parseExpr())
		}
		if p.curToken.Type == lexer.TOKEN_RPAREN {
			p.nextToken() // ')'
		}
		return &ast.CallExpr{
			Fn:   opOrFn,
			Args: args,
			Line: line,
			Col:  col,
		}
	}

	p.nextToken()
	return nil
}

func isBinaryOp(op string) bool {
	switch op {
	case "add", "sub", "mul", "div", "eq", "neq", "gt", "lt", "gte", "lte", "and", "or":
		return true
	default:
		return false
	}
}

func (p *Parser) parseType() string {
	if p.curToken.Type != lexer.TOKEN_IDENT {
		return ""
	}
	t := p.curToken.Literal
	p.nextToken()
	if t == "chan" && p.curToken.Type == lexer.TOKEN_LBRACKET {
		p.nextToken() // consome '['
		inner := p.parseType()
		if p.curToken.Type == lexer.TOKEN_RBRACKET {
			p.nextToken() // consome ']'
		}
		return fmt.Sprintf("%s[%s]", t, inner)
	}
	return t
}
