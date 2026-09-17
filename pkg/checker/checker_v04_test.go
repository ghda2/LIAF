package checker_test

import "testing"

// Regras da v0.4: banco de dados (issue #015) e WebSocket (issue #016).
// O que se testa aqui e o que o compilador recusa — a garantia de que um
// modelo nao consegue escrever a forma insegura sem ser barrado.

const dbPreamble = `(module db
(struct User (fields (id int) (name str)))
`

const dbEpilogue = `
(fn main (params) (returns void) (effects) (body)))`

func dbModule(body string) string { return dbPreamble + body + dbEpilogue }

// --- Anti-injecao -----------------------------------------------------------

func TestCheckV04RejectsInterpolatedSQL(t *testing.T) {
	for _, tc := range []struct{ name, call string }{
		{
			// A forma mais curta e mais tentadora, e a que abre a porta.
			name: "concat na query",
			call: `(db-query db (concat "SELECT id, name FROM users WHERE name = '" nome "'") User)`,
		},
		{
			name: "query vinda de variavel",
			call: `(db-query db sql User)`,
		},
		{
			name: "concat no db-exec",
			call: `(db-exec db (concat "DELETE FROM users WHERE name = '" nome "'"))`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			codes := diagCodes(t, dbModule(`
(fn busca (params (db DBConnection) (nome str) (sql str)) (returns (result bool str)) (effects db)
  (body
    (try `+tc.call+`)
    (return (ok true))))`))
			if !hasCode(codes, "E_SQL_INTERPOLATION") {
				t.Fatalf("esperado E_SQL_INTERPOLATION, recebido %v", codes)
			}
		})
	}
}

func TestCheckV04AcceptsParameterizedSQL(t *testing.T) {
	codes := diagCodes(t, dbModule(`
(fn busca (params (db DBConnection) (nome str)) (returns (result bool str)) (effects db)
  (body
    (let achados (list User)
      (try (db-query db "SELECT id, name FROM users WHERE name = $1" User nome)))
    (try (db-exec db "DELETE FROM users WHERE name = $1" nome))
    (return (ok true))))`))
	if len(codes) > 0 {
		t.Fatalf("consulta parametrizada foi recusada: %v", codes)
	}
}

func TestCheckV04CountsPlaceholders(t *testing.T) {
	for _, tc := range []struct {
		name  string
		call  string
		valid bool
	}{
		{"faltando um argumento", `(db-exec db "INSERT INTO users (id, name) VALUES ($1, $2)" 1)`, false},
		{"argumento a mais", `(db-exec db "DELETE FROM users WHERE id = $1" 1 2)`, false},
		{"argumento sem marcador", `(db-exec db "DELETE FROM users" 1)`, false},
		{"estilos misturados", `(db-exec db "DELETE FROM users WHERE id = $1 AND name = ?" 1 "a")`, false},
		{"contagem certa com $n", `(db-exec db "INSERT INTO users (id, name) VALUES ($1, $2)" 1 "alice")`, true},
		{"contagem certa com ?", `(db-exec db "DELETE FROM users WHERE id = ?" 1)`, true},
		{"$1 repetido conta uma vez", `(db-exec db "UPDATE users SET name = $1 WHERE name = $1" "a")`, true},
		// Um ? dentro de literal nao e marcador.
		{"interrogacao dentro de literal", `(db-exec db "UPDATE users SET name = 'quem?' WHERE id = $1" 1)`, true},
		{"marcador em comentario ignorado", `(db-exec db "DELETE FROM users -- era $9\n WHERE id = $1" 1)`, true},
		{"sem marcador e sem argumento", `(db-exec db "DELETE FROM users")`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			codes := diagCodes(t, dbModule(`
(fn roda (params (db DBConnection)) (returns (result bool str)) (effects db)
  (body
    (try `+tc.call+`)
    (return (ok true))))`))
			got := hasCode(codes, "E_SQL_PARAM_COUNT")
			if tc.valid && got {
				t.Fatalf("forma valida foi recusada: %v", codes)
			}
			if !tc.valid && !got {
				t.Fatalf("esperado E_SQL_PARAM_COUNT, recebido %v", codes)
			}
		})
	}
}

// Os operadores JSONB do postgres (?, ?| e ?&) sao indistinguiveis de
// marcadores numa varredura lexica, entao a contagem e abandonada em vez de
// acusar um falso positivo do qual nao ha escapatoria.
func TestCheckV04SkipsCountForJSONBOperators(t *testing.T) {
	codes := diagCodes(t, dbModule(`
(fn roda (params (db DBConnection)) (returns (result bool str)) (effects db)
  (body
    (try (db-exec db "DELETE FROM users WHERE meta ?| array['a','b'] AND id = $1" 1))
    (return (ok true))))`))
	if hasCode(codes, "E_SQL_PARAM_COUNT") {
		t.Fatalf("operador JSONB gerou falso positivo: %v", codes)
	}
}

// --- Tipos e efeitos --------------------------------------------------------

func TestCheckV04DriverName(t *testing.T) {
	for _, tc := range []struct {
		name   string
		driver string
		valid  bool
	}{
		{"postgres", `"postgres"`, true},
		{"mysql", `"mysql"`, true},
		{"redis", `"redis"`, true},
		// "postgresql" e "pg" sao os erros de digitacao mais comuns e so
		// apareceriam em execucao se o nome fosse dinamico.
		{"nome errado", `"postgresql"`, false},
		{"nome dinamico", `dsn`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			codes := diagCodes(t, dbModule(`
(fn abre (params (dsn str)) (returns (result DBConnection str)) (effects db)
  (body (return (db-connect `+tc.driver+` dsn))))`))
			got := hasCode(codes, "E_UNKNOWN_DRIVER")
			if tc.valid && got {
				t.Fatalf("driver valido recusado: %v", codes)
			}
			if !tc.valid && !got {
				t.Fatalf("esperado E_UNKNOWN_DRIVER, recebido %v", codes)
			}
		})
	}
}

func TestCheckV04RequiresDBEffect(t *testing.T) {
	codes := diagCodes(t, dbModule(`
(fn abre (params (dsn str)) (returns (result DBConnection str)) (effects)
  (body (return (db-connect "postgres" dsn))))`))
	if !hasCode(codes, "E_UNDECLARED_EFFECT") {
		t.Fatalf("esperado E_UNDECLARED_EFFECT, recebido %v", codes)
	}
}

func TestCheckV04QueryRowTypeMustBeStruct(t *testing.T) {
	codes := diagCodes(t, dbModule(`
(fn roda (params (db DBConnection)) (returns (result bool str)) (effects db)
  (body
    (let nomes (list str) (try (db-query db "SELECT name FROM users" str)))
    (return (ok true))))`))
	if !hasCode(codes, "E_INVALID_TYPE") {
		t.Fatalf("esperado E_INVALID_TYPE para tipo de linha escalar, recebido %v", codes)
	}
}

func TestCheckV04QueryArgumentsMustBeScalar(t *testing.T) {
	codes := diagCodes(t, dbModule(`
(fn roda (params (db DBConnection) (u User)) (returns (result bool str)) (effects db)
  (body
    (try (db-exec db "DELETE FROM users WHERE id = $1" u))
    (return (ok true))))`))
	if !hasCode(codes, "E_TYPE_MISMATCH") {
		t.Fatalf("esperado E_TYPE_MISMATCH para struct como parametro, recebido %v", codes)
	}
}

func TestCheckV04ConnectionIsOpaque(t *testing.T) {
	// DBConnection nao tem campos; ler um seria um erro silencioso.
	codes := diagCodes(t, dbModule(`
(fn roda (params (db DBConnection)) (returns (result bool str)) (effects db)
  (body
    (try (db-exec db "DELETE FROM users"))
    (println (field db host))
    (return (ok true))))`))
	if !hasCode(codes, "E_UNKNOWN_FIELD") {
		t.Fatalf("esperado E_UNKNOWN_FIELD, recebido %v", codes)
	}
}

func TestCheckV04ConnectResultMustBeHandled(t *testing.T) {
	codes := diagCodes(t, dbModule(`
(fn roda (params (dsn str)) (returns void) (effects db)
  (body
    (let db (result DBConnection str) (db-connect "postgres" dsn))))`))
	if !hasCode(codes, "E_UNHANDLED_RESULT") {
		t.Fatalf("esperado E_UNHANDLED_RESULT, recebido %v", codes)
	}
}

// --- Transacoes -------------------------------------------------------------

func TestCheckV04TransactionRequiresErrorPath(t *testing.T) {
	// Abrir e confirmar podem falhar; sem (on-err ...) nem retorno result
	// nao ha para onde desviar o erro.
	codes := diagCodes(t, dbModule(`
(fn roda (params (db DBConnection)) (returns void) (effects db)
  (body
    (db-transaction db
      (try (db-exec db "DELETE FROM users")))))`))
	if !hasCode(codes, "E_UNHANDLED_RESULT") {
		t.Fatalf("esperado E_UNHANDLED_RESULT, recebido %v", codes)
	}
}

func TestCheckV04TransactionAcceptsOnErr(t *testing.T) {
	codes := diagCodes(t, dbModule(`
(fn roda (params (db DBConnection)) (returns void) (effects db io)
  (on-err message (println message))
  (body
    (db-transaction db
      (try (db-exec db "DELETE FROM users")))))`))
	if len(codes) > 0 {
		t.Fatalf("transacao com on-err foi recusada: %v", codes)
	}
}

func TestCheckV04TransactionTargetMustBeConnection(t *testing.T) {
	codes := diagCodes(t, dbModule(`
(fn roda (params (db DBConnection) (nome str)) (returns (result bool str)) (effects db)
  (body
    (db-transaction nome
      (try (db-exec db "DELETE FROM users")))
    (return (ok true))))`))
	if !hasCode(codes, "E_TYPE_MISMATCH") {
		t.Fatalf("esperado E_TYPE_MISMATCH, recebido %v", codes)
	}
}

func TestCheckV04TransactionBodyCountsAsReturn(t *testing.T) {
	// O corpo da transacao sempre executa, entao um return la dentro
	// satisfaz o retorno da funcao.
	codes := diagCodes(t, dbModule(`
(fn roda (params (db DBConnection)) (returns (result bool str)) (effects db)
  (body
    (db-transaction db
      (try (db-exec db "DELETE FROM users"))
      (return (ok true)))))`))
	if hasCode(codes, "E_MISSING_RETURN") {
		t.Fatalf("return dentro da transacao nao foi reconhecido: %v", codes)
	}
}

// --- WebSocket --------------------------------------------------------------

const wsEpilogue = "\n(fn main (params) (returns void) (effects) (body)))"

func wsModule(route string) string { return "(module ws\n" + route + wsEpilogue }

func TestCheckV04WSRouteAccepted(t *testing.T) {
	codes := diagCodes(t, wsModule(`
(ws-route "/ws/chat/{room}" (params (room str) (conn WSConn)) (effects io net)
  (on-err message (println message))
  (on-open
    (ws-join conn room))
  (on-message text
    (try (ws-broadcast room text)))
  (on-close
    (println "saiu")))`))
	if len(codes) > 0 {
		t.Fatalf("ws-route valida foi recusada: %v", codes)
	}
}

func TestCheckV04WSRouteMinimalForm(t *testing.T) {
	// on-open e on-close sao opcionais; on-message nao e.
	codes := diagCodes(t, wsModule(`
(ws-route "/ws/echo" (params (conn WSConn)) (effects net)
  (on-err message (ws-join conn message))
  (on-message text
    (try (ws-send conn text))))`))
	if len(codes) > 0 {
		t.Fatalf("forma minima foi recusada: %v", codes)
	}
}

func TestCheckV04WSRouteParamRules(t *testing.T) {
	for _, tc := range []struct{ name, route string }{
		{
			name:  "sem WSConn",
			route: `(ws-route "/ws/x" (params) (effects net) (on-message t (try (ws-broadcast "x" t))))`,
		},
		{
			name:  "dois WSConn",
			route: `(ws-route "/ws/x" (params (a WSConn) (b WSConn)) (effects net) (on-message t (try (ws-send a t))))`,
		},
		{
			// Nao ha corpo JSON num handshake: esse parametro nao teria de
			// onde ser preenchido.
			name:  "parametro fora do path",
			route: `(ws-route "/ws/x" (params (conn WSConn) (extra int)) (effects net) (on-message t (try (ws-send conn t))))`,
		},
		{
			name:  "path param nao declarado",
			route: `(ws-route "/ws/{room}" (params (conn WSConn)) (effects net) (on-message t (try (ws-send conn t))))`,
		},
		{
			name:  "path param composto",
			route: `(ws-route "/ws/{room}" (params (room (list int)) (conn WSConn)) (effects net) (on-message t (try (ws-send conn t))))`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			codes := diagCodes(t, wsModule(tc.route))
			if !hasCode(codes, "E_WS_PARAM") {
				t.Fatalf("esperado E_WS_PARAM, recebido %v", codes)
			}
		})
	}
}

func TestCheckV04WSRouteRequiresNetEffect(t *testing.T) {
	// Manter a conexao aberta ja e trafego de rede, mesmo que o corpo so logue.
	codes := diagCodes(t, wsModule(`
(ws-route "/ws/x" (params (conn WSConn)) (effects io)
  (on-message text (println text)))`))
	if !hasCode(codes, "E_UNDECLARED_EFFECT") {
		t.Fatalf("esperado E_UNDECLARED_EFFECT, recebido %v", codes)
	}
}

// Sem esta excecao, (effects net io) de um handler que so loga cairia em
// E_UNUSED_EFFECT, e tirar net tornaria ws-send impossivel.
func TestCheckV04WSRouteNetIsAlwaysUsed(t *testing.T) {
	codes := diagCodes(t, wsModule(`
(ws-route "/ws/x" (params (conn WSConn)) (effects io net)
  (on-message text (println text)))`))
	if len(codes) > 0 {
		t.Fatalf("net devia contar como usado numa ws-route: %v", codes)
	}
}

func TestCheckV04WSRouteRejectsUnusedEffect(t *testing.T) {
	codes := diagCodes(t, wsModule(`
(ws-route "/ws/x" (params (conn WSConn)) (effects net fs)
  (on-message text (try (ws-send conn text))))`))
	if !hasCode(codes, "E_UNUSED_EFFECT") {
		t.Fatalf("esperado E_UNUSED_EFFECT para fs, recebido %v", codes)
	}
}

func TestCheckV04WSMessageVariableIsScopedToItsBlock(t *testing.T) {
	// text so existe dentro de on-message; cada bloco vira uma funcao propria.
	codes := diagCodes(t, wsModule(`
(ws-route "/ws/x" (params (conn WSConn)) (effects io net)
  (on-message text (println text))
  (on-close (println text)))`))
	if !hasCode(codes, "E_UNDEFINED_SYMBOL") {
		t.Fatalf("esperado E_UNDEFINED_SYMBOL, recebido %v", codes)
	}
}

func TestCheckV04WSBuiltinArgumentTypes(t *testing.T) {
	for _, tc := range []struct{ name, call string }{
		{"ws-send sem conexao", `(try (ws-send room text))`},
		{"ws-broadcast com conexao", `(try (ws-broadcast conn text))`},
		{"ws-close sem codigo", `(try (ws-close conn "1000" "tchau"))`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			codes := diagCodes(t, wsModule(`
(ws-route "/ws/{room}" (params (room str) (conn WSConn)) (effects net)
  (on-message text `+tc.call+`))`))
			if !hasCode(codes, "E_TYPE_MISMATCH") {
				t.Fatalf("esperado E_TYPE_MISMATCH, recebido %v", codes)
			}
		})
	}
}

// ws-join e ws-leave nao devolvem result: sao operacoes locais sobre um mapa
// em memoria e nao tem caminho de falha.
func TestCheckV04JoinAndLeaveAreNotResults(t *testing.T) {
	codes := diagCodes(t, wsModule(`
(ws-route "/ws/{room}" (params (room str) (conn WSConn)) (effects io net)
  (on-err message (println message))
  (on-open (ws-join conn room))
  (on-message text (try (ws-send conn text)))
  (on-close (ws-leave conn room)))`))
	if len(codes) > 0 {
		t.Fatalf("ws-join/ws-leave como instrucao foram recusados: %v", codes)
	}
}

// Os blocos de uma ws-route devolvem void, entao um try sem (on-err ...) nao
// tem para onde desviar — a mesma regra de uma funcao void.
func TestCheckV04WSTryRequiresOnErr(t *testing.T) {
	codes := diagCodes(t, wsModule(`
(ws-route "/ws/x" (params (conn WSConn)) (effects net)
  (on-message text (try (ws-send conn text))))`))
	if !hasCode(codes, "E_UNHANDLED_RESULT") {
		t.Fatalf("esperado E_UNHANDLED_RESULT, recebido %v", codes)
	}
}

func TestCheckV04UnknownEffectStillRejected(t *testing.T) {
	codes := diagCodes(t, dbModule(`
(fn roda (params (db DBConnection)) (returns (result bool str)) (effects database)
  (body
    (try (db-exec db "DELETE FROM users"))
    (return (ok true))))`))
	if !hasCode(codes, "E_UNKNOWN_EFFECT") {
		t.Fatalf("esperado E_UNKNOWN_EFFECT para 'database', recebido %v", codes)
	}
}
