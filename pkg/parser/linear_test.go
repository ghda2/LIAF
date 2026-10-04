package parser

import (
	"strings"
	"testing"

	"liaf/pkg/ast"
	"liaf/pkg/lexer"
)

func TestParseLinearStructAndFunc(t *testing.T) {
	input := `
	// Tipos
	struct Item
		id int
		nome str
		preco float
	end

	// Funções
	fn somar(a int, b int) int
		a + b
	end

	fn principal() void effects(io)
		let x = somar(10, 20)
		println(x)
	end
	`

	l := lexer.New(input)
	p := New(l, "app.liaf")
	mod := p.ParseModule()

	if len(p.Diagnostics) > 0 {
		t.Fatalf("diagnósticos inesperados: %+v", p.Diagnostics)
	}

	if mod == nil {
		t.Fatalf("mod é nulo")
	}

	if len(mod.Decls) != 3 {
		t.Fatalf("esperado 3 declarações, obteve %d", len(mod.Decls))
	}

	// 1. Struct
	st, ok := mod.Decls[0].(*ast.StructDecl)
	if !ok {
		t.Fatalf("decls[0] não é StructDecl")
	}
	if st.Name != "Item" || len(st.Fields) != 3 {
		t.Errorf("struct Item incorreta: %s (%d campos)", st.Name, len(st.Fields))
	}

	// 2. Func somar com implicit return de binary op
	fn1, ok := mod.Decls[1].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("decls[1] não é FuncDecl")
	}
	if fn1.Name != "somar" || len(fn1.Params) != 2 {
		t.Errorf("func somar incorreta")
	}
	if len(fn1.Body) != 1 {
		t.Fatalf("esperado 1 stmt no corpo de somar, obteve %d", len(fn1.Body))
	}
	retStmt, ok := fn1.Body[0].(*ast.ReturnStmt)
	if !ok {
		t.Fatalf("esperado ReturnStmt implícito, obteve %T", fn1.Body[0])
	}
	binOp, ok := retStmt.Value.(*ast.BinaryOpExpr)
	if !ok || binOp.Op != "add" {
		t.Fatalf("esperado BinaryOpExpr 'add', obteve %v", retStmt.Value)
	}

	// 3. Func principal com effects(io)
	fn2, ok := mod.Decls[2].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("decls[2] não é FuncDecl")
	}
	if fn2.Name != "principal" || len(fn2.Effects) != 1 || fn2.Effects[0] != "io" {
		t.Errorf("func principal com efeitos incorretos: %v", fn2.Effects)
	}
}

func TestParseLinearRouteAndTry(t *testing.T) {
	input := `
	route GET "/api/cardapio" () Response effects(db)
		let db = try db-connect("sqlite", "cardapio.db")
		let itens = try db-query(db, "SELECT * FROM itens", Item)
		json-response(200, try json-encode(itens))
	end
	`

	l := lexer.New(input)
	p := New(l, "cardapio.liaf")
	mod := p.ParseModule()

	if len(p.Diagnostics) > 0 {
		t.Fatalf("diagnósticos inesperados: %+v", p.Diagnostics)
	}

	if len(mod.Decls) != 1 {
		t.Fatalf("esperado 1 declaração, obteve %d", len(mod.Decls))
	}

	route, ok := mod.Decls[0].(*ast.RouteDecl)
	if !ok {
		t.Fatalf("esperado RouteDecl, obteve %T", mod.Decls[0])
	}

	if route.Method != "GET" || route.Path != "/api/cardapio" {
		t.Errorf("rota incorreta: %s %s", route.Method, route.Path)
	}

	if len(route.Effects) != 1 || route.Effects[0] != "db" {
		t.Errorf("efeito de rota incorreto: %v", route.Effects)
	}

	// Último stmt deve ser return implícito do json-response
	last := route.Body[len(route.Body)-1]
	if _, ok := last.(*ast.ReturnStmt); !ok {
		t.Errorf("esperado ReturnStmt no final da rota, obteve %T", last)
	}
}

func TestParseLinearWebSocket(t *testing.T) {
	input := `
	ws-route "/ws/chat" (req Request, conn WSConn) effects(io, net)
		on-open
			ws-join(conn, "chat_geral")
			println("Conectado")
		end

		on-message text
			try ws-broadcast("chat_geral", text)
		end

		on-close
			println("Desconectado")
		end
	end
	`

	l := lexer.New(input)
	p := New(l, "ws.liaf")
	mod := p.ParseModule()

	if len(p.Diagnostics) > 0 {
		t.Fatalf("diagnósticos inesperados: %+v", p.Diagnostics)
	}

	if len(mod.Decls) != 1 {
		t.Fatalf("esperado 1 declaração, obteve %d", len(mod.Decls))
	}

	ws, ok := mod.Decls[0].(*ast.WSRouteDecl)
	if !ok {
		t.Fatalf("esperado WSRouteDecl, obteve %T", mod.Decls[0])
	}

	if ws.Path != "/ws/chat" || len(ws.Params) != 2 || len(ws.Effects) != 2 {
		t.Errorf("ws-route incorreto")
	}

	if len(ws.OnOpen) != 2 {
		t.Errorf("esperado 2 stmts em on-open, obteve %d", len(ws.OnOpen))
	}

	if ws.MsgVar != "text" || len(ws.OnMessage) != 1 {
		t.Errorf("on-message incorreto: msgVar=%s, len=%d", ws.MsgVar, len(ws.OnMessage))
	}

	if len(ws.OnClose) != 1 {
		t.Errorf("esperado 1 stmt em on-close, obteve %d", len(ws.OnClose))
	}
}

func TestParseLinearControlFlow(t *testing.T) {
	input := `
	fn testarFluxo(n int) int
		if n > 10
			let dobro = n * 2
			return dobro
		else
			let triplo = n * 3
			return triplo
		end

		while n < 100
			set n = n + 1
			if n == 50
				break
			end
		end

		for-range i 0 10
			println(i)
		end

		for-each item itens
			println(item)
		end

		n
	end
	`

	l := lexer.New(input)
	p := New(l, "fluxo.liaf")
	mod := p.ParseModule()

	if len(p.Diagnostics) > 0 {
		for _, d := range p.Diagnostics {
			t.Logf("diag: [%s] %s:%d:%d %s", d.Code, d.File, d.Line, d.Col, d.Message)
		}
		t.Fatalf("diagnósticos inesperados: %d", len(p.Diagnostics))
	}

	fn, ok := mod.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("esperado FuncDecl")
	}

	if len(fn.Body) != 5 {
		t.Fatalf("esperado 5 stmts, obteve %d", len(fn.Body))
	}

	// 1: if
	ifStmt, ok := fn.Body[0].(*ast.IfStmt)
	if !ok || len(ifStmt.Then) != 2 || len(ifStmt.Else) != 2 {
		t.Errorf("ifStmt inválido: then=%d, else=%d", len(ifStmt.Then), len(ifStmt.Else))
	}

	// 2: while
	whileStmt, ok := fn.Body[1].(*ast.LoopStmt)
	if !ok || whileStmt.Kind != "while" {
		t.Errorf("whileStmt inválido: %v", fn.Body[1])
	}

	// 3: for-range
	frStmt, ok := fn.Body[2].(*ast.LoopStmt)
	if !ok || frStmt.Kind != "for-range" || frStmt.Name != "i" {
		t.Errorf("for-range inválido: %v", fn.Body[2])
	}

	// 4: for-each
	feStmt, ok := fn.Body[3].(*ast.LoopStmt)
	if !ok || feStmt.Kind != "for-each" || feStmt.Name != "item" {
		t.Errorf("for-each inválido: %v", fn.Body[3])
	}

	// 5: return implícito
	ret, ok := fn.Body[4].(*ast.ReturnStmt)
	if !ok {
		t.Errorf("retorno implícito inválido")
	}
	ident, ok := ret.Value.(*ast.IdentExpr)
	if !ok || ident.Name != "n" {
		t.Errorf("esperado retorno de n")
	}
}

func TestParseLinearOperatorPrecedence(t *testing.T) {
	input := `
	fn calc(a int, b int, c int) int
		a + b * c
	end
	`
	l := lexer.New(input)
	p := New(l, "calc.liaf")
	mod := p.ParseModule()

	if len(p.Diagnostics) > 0 {
		t.Fatalf("diagnósticos: %+v", p.Diagnostics)
	}

	fn := mod.Decls[0].(*ast.FuncDecl)
	ret := fn.Body[0].(*ast.ReturnStmt)
	bin := ret.Value.(*ast.BinaryOpExpr)

	if bin.Op != "add" {
		t.Fatalf("raiz deveria ser 'add', obteve %s", bin.Op)
	}

	rightBin, ok := bin.Right.(*ast.BinaryOpExpr)
	if !ok || rightBin.Op != "mul" {
		t.Fatalf("ramo direito deveria ser 'mul', obteve %v", bin.Right)
	}
}

func TestCanonicalFormattingFromLinearAST(t *testing.T) {
	input := `
	fn somar(a int, b int) int
		a + b
	end
	`
	l := lexer.New(input)
	p := New(l, "calc.liaf")
	mod := p.ParseModule()

	if len(p.Diagnostics) > 0 {
		t.Fatalf("diagnósticos: %+v", p.Diagnostics)
	}

	formatted := strings.TrimSpace(ast.Format(mod))
	expected := strings.TrimSpace(`(module calc
  (fn somar ((a int) (b int)) int (effects)
    (add a b)))`)

	if formatted != expected {
		t.Errorf("formatação divergente:\nEsperado:\n%s\nObtido:\n%s", expected, formatted)
	}
}

func TestParseLinearEnumAndMatch(t *testing.T) {
	input := `
	enum Status
		Pendente
		Pago
		Cancelado
	end

	fn checar(s Status) str
		match s
			case Status.Pendente
				"Aguardando"
			case Status.Pago
				"Confirmado"
			case Status.Cancelado
				"Cancelado"
		end
	end
	`
	l := lexer.New(input)
	p := New(l, "enum_test.liaf")
	mod := p.ParseModule()

	if len(p.Diagnostics) > 0 {
		t.Fatalf("diagnósticos inesperados: %+v", p.Diagnostics)
	}

	if len(mod.Decls) != 2 {
		t.Fatalf("esperado 2 declarações, obteve %d", len(mod.Decls))
	}

	enumDecl, ok := mod.Decls[0].(*ast.EnumDecl)
	if !ok {
		t.Fatalf("decls[0] não é *ast.EnumDecl")
	}
	if enumDecl.Name != "Status" || len(enumDecl.Variants) != 3 {
		t.Fatalf("esperado enum Status com 3 variantes, obteve %+v", enumDecl)
	}
	if enumDecl.Variants[0].Name != "Pendente" || enumDecl.Variants[1].Name != "Pago" || enumDecl.Variants[2].Name != "Cancelado" {
		t.Fatalf("variantes incorretas: %+v", enumDecl.Variants)
	}

	fnDecl, ok := mod.Decls[1].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("decls[1] não é *ast.FuncDecl")
	}
	if len(fnDecl.Body) != 1 {
		t.Fatalf("esperado 1 stmt no body, obteve %d", len(fnDecl.Body))
	}
	matchStmt, ok := fnDecl.Body[0].(*ast.MatchStmt)
	if !ok {
		t.Fatalf("stmt não é *ast.MatchStmt")
	}
	if len(matchStmt.Cases) != 3 {
		t.Fatalf("esperado 3 cases, obteve %d", len(matchStmt.Cases))
	}
	if matchStmt.Cases[0].Pattern != "Status.Pendente" {
		t.Fatalf("case[0] pattern incorreto: %s", matchStmt.Cases[0].Pattern)
	}
}

func TestParsePubVisibility(t *testing.T) {
	// 1. Linear Syntax
	linearInput := `
	module auth

	pub struct Usuario
		id int
	end

	struct TokenInterno
		hash str
	end

	pub enum Status
		Ativo
		Inativo
	end

	enum Privado
		A
		B
	end

	pub fn login(user str) bool
		return true
	end

	fn helper() void
	end
	`
	l := lexer.New(linearInput)
	p := New(l, "auth.liaf")
	mod := p.ParseModule()
	if len(p.Diagnostics) > 0 {
		t.Fatalf("linear diags: %+v", p.Diagnostics)
	}
	if len(mod.Decls) != 6 {
		t.Fatalf("esperado 6 decls, obteve %d", len(mod.Decls))
	}

	stPub := mod.Decls[0].(*ast.StructDecl)
	if !stPub.IsPub || stPub.Module != "auth" {
		t.Errorf("Usuario devia ser IsPub=true e Module=auth, obteve %+v", stPub)
	}

	stPriv := mod.Decls[1].(*ast.StructDecl)
	if stPriv.IsPub || stPriv.Module != "auth" {
		t.Errorf("TokenInterno devia ser IsPub=false, obteve %+v", stPriv)
	}

	enPub := mod.Decls[2].(*ast.EnumDecl)
	if !enPub.IsPub || enPub.Module != "auth" {
		t.Errorf("Status devia ser IsPub=true, obteve %+v", enPub)
	}

	enPriv := mod.Decls[3].(*ast.EnumDecl)
	if enPriv.IsPub || enPriv.Module != "auth" {
		t.Errorf("Privado devia ser IsPub=false, obteve %+v", enPriv)
	}

	fnPub := mod.Decls[4].(*ast.FuncDecl)
	if !fnPub.IsPub || fnPub.Module != "auth" {
		t.Errorf("login devia ser IsPub=true, obteve %+v", fnPub)
	}

	fnPriv := mod.Decls[5].(*ast.FuncDecl)
	if fnPriv.IsPub || fnPriv.Module != "auth" {
		t.Errorf("helper devia ser IsPub=false, obteve %+v", fnPriv)
	}

	// 2. S-Expressions
	sexprInput := `(module math
		(pub struct Ponto (fields (x int) (y int)))
		(struct Interno (fields (v int)))
		(pub enum Cor Vermelho Verde)
		(enum Modo A B)
		(pub fn calc (params (p Ponto)) (returns int) (effects) (body (return 1)))
		(fn aux (params) (returns void) (effects) (body))
	)`
	l2 := lexer.New(sexprInput)
	p2 := New(l2, "math.liaf")
	mod2 := p2.ParseModule()
	if len(p2.Diagnostics) > 0 {
		t.Fatalf("sexpr diags: %+v", p2.Diagnostics)
	}
	if len(mod2.Decls) != 6 {
		t.Fatalf("esperado 6 decls no sexpr, obteve %d", len(mod2.Decls))
	}
	if !mod2.Decls[0].(*ast.StructDecl).IsPub || mod2.Decls[1].(*ast.StructDecl).IsPub {
		t.Errorf("visibilidade de struct incorreta no sexpr")
	}
	if !mod2.Decls[2].(*ast.EnumDecl).IsPub || mod2.Decls[3].(*ast.EnumDecl).IsPub {
		t.Errorf("visibilidade de enum incorreta no sexpr")
	}
	if !mod2.Decls[4].(*ast.FuncDecl).IsPub || mod2.Decls[5].(*ast.FuncDecl).IsPub {
		t.Errorf("visibilidade de fn incorreta no sexpr")
	}
}

func TestParseLinearPostfixFieldAccess(t *testing.T) {
	input := `
	fn test() void
		let x = unwrap-or(list-get(l, 0), padrao).total
		let y = pegar-usuario().conta.saldo
	end
	`
	l := lexer.New(input)
	p := New(l, "test.liaf")
	mod := p.ParseModule()

	if len(p.Diagnostics) > 0 {
		t.Fatalf("diagnósticos inesperados: %+v", p.Diagnostics)
	}

	if mod == nil || len(mod.Decls) != 1 {
		t.Fatalf("esperava 1 decl, obteve %+v", mod)
	}

	fnDecl := mod.Decls[0].(*ast.FuncDecl)
	if len(fnDecl.Body) != 2 {
		t.Fatalf("esperava 2 stmts no corpo, obteve %d", len(fnDecl.Body))
	}

	// 1. unwrap-or(...).total deve virar (field (unwrap-or ...) total)
	let1 := fnDecl.Body[0].(*ast.LetStmt)
	call1, ok := let1.Value.(*ast.CallExpr)
	if !ok || call1.Func != "field" {
		t.Fatalf("esperava CallExpr 'field', obteve %+v", let1.Value)
	}
	if call1.Args[1].(*ast.IdentExpr).Name != "total" {
		t.Errorf("campo esperado 'total', obteve %s", call1.Args[1].(*ast.IdentExpr).Name)
	}

	// 2. pegar-usuario().conta.saldo deve virar (field (field (pegar-usuario) conta) saldo)
	let2 := fnDecl.Body[1].(*ast.LetStmt)
	call2, ok := let2.Value.(*ast.CallExpr)
	if !ok || call2.Func != "field" {
		t.Fatalf("esperava CallExpr 'field', obteve %+v", let2.Value)
	}
	if call2.Args[1].(*ast.IdentExpr).Name != "saldo" {
		t.Errorf("campo esperado 'saldo', obteve %s", call2.Args[1].(*ast.IdentExpr).Name)
	}
}

