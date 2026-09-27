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
