package runtime

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"liaf/pkg/dbdrv"
)

// A conversao de linha em struct e o pool sao a camada entre o driver e a
// linguagem. Testa-los com uma conexao falsa isola essa camada do
// enquadramento de protocolo, que ja e coberto em pkg/dbdrv.

type stubConn struct {
	mu        sync.Mutex
	rows      *dbdrv.Rows
	queryErr  error
	execErr   error
	affected  int64
	commands  []string
	closed    bool
	beginErr  error
	commitErr error
}

func (s *stubConn) record(op string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commands = append(s.commands, op)
}

func (s *stubConn) Query(string, []dbdrv.Value) (*dbdrv.Rows, error) {
	s.record("query")
	return s.rows, s.queryErr
}
func (s *stubConn) Exec(string, []dbdrv.Value) (int64, error) {
	s.record("exec")
	return s.affected, s.execErr
}
func (s *stubConn) Begin() error    { s.record("begin"); return s.beginErr }
func (s *stubConn) Commit() error   { s.record("commit"); return s.commitErr }
func (s *stubConn) Rollback() error { s.record("rollback"); return nil }
func (s *stubConn) Ping() error     { return nil }
func (s *stubConn) Close() error    { s.closed = true; return nil }

func (s *stubConn) seen() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.commands...)
}

// stubPool monta um DBConn ja abastecido, sem tocar a rede.
func stubPool(conn dbdrv.Conn) *DBConn {
	c := &DBConn{driver: "stub", dsn: "stub://", sem: make(chan struct{}, 4)}
	c.idle = append(c.idle, conn)
	return c
}

type user struct {
	ID     int64   `json:"id"`
	Name   string  `json:"name"`
	Score  float64 `json:"score"`
	Active bool    `json:"active"`
	NextID int64   `json:"next-id"`
}

func TestDBQueryMapsColumnsToStructFields(t *testing.T) {
	stub := &stubConn{rows: &dbdrv.Rows{
		Columns: []string{"id", "name", "score", "active", "next_id"},
		Values: [][]dbdrv.Value{
			{int64(1), "alice", 9.5, true, int64(2)},
			{int64(2), "bruno", 7.25, false, int64(3)},
		},
	}}
	result := DBQuery[user](stubPool(stub), "SELECT ...")
	if !result.OK {
		t.Fatalf("query falhou: %s", result.Error)
	}
	items := result.Value.Items
	if len(items) != 2 {
		t.Fatalf("esperadas 2 linhas, recebidas %d", len(items))
	}
	if items[0] != (user{ID: 1, Name: "alice", Score: 9.5, Active: true, NextID: 2}) {
		t.Errorf("linha 0: %+v", items[0])
	}
	// A coluna next_id precisa alimentar o campo LIAF next-id: snake_case no
	// banco e kebab-case na linguagem sao a combinacao normal.
	if items[1].NextID != 3 {
		t.Errorf("apelido snake_case/kebab-case nao funcionou: %+v", items[1])
	}
}

func TestDBQueryHandlesNullAndEmpty(t *testing.T) {
	stub := &stubConn{rows: &dbdrv.Rows{
		Columns: []string{"id", "name"},
		Values:  [][]dbdrv.Value{{int64(1), nil}},
	}}
	result := DBQuery[user](stubPool(stub), "SELECT ...")
	if !result.OK {
		t.Fatalf("NULL quebrou a conversao: %s", result.Error)
	}
	if result.Value.Items[0].Name != "" {
		t.Errorf("NULL devia virar o zero do campo: %q", result.Value.Items[0].Name)
	}

	empty := DBQuery[user](stubPool(&stubConn{rows: &dbdrv.Rows{Columns: []string{"id"}}}), "SELECT ...")
	if !empty.OK {
		t.Fatalf("resultado vazio virou erro: %s", empty.Error)
	}
	// Lista vazia, e nao nil: json-encode dela precisa produzir [].
	if empty.Value.Items == nil || len(empty.Value.Items) != 0 {
		t.Errorf("esperada lista vazia, recebido %#v", empty.Value.Items)
	}
}

func TestDBQueryExplainsColumnMismatch(t *testing.T) {
	stub := &stubConn{rows: &dbdrv.Rows{
		Columns: []string{"id"},
		Values:  [][]dbdrv.Value{{"nao-e-numero"}},
	}}
	result := DBQuery[user](stubPool(stub), "SELECT ...")
	if result.OK {
		t.Fatal("coluna incompativel foi aceita")
	}
	if !strings.Contains(result.Error, "select the columns") {
		t.Errorf("mensagem nao orienta a correcao: %s", result.Error)
	}
}

func TestDBExecReturnsAffectedRows(t *testing.T) {
	stub := &stubConn{affected: 3}
	result := DBExec(stubPool(stub), "UPDATE ...", int64(1))
	if !result.OK || result.Value != 3 {
		t.Fatalf("resultado: %+v", result)
	}
}

func TestDBTransactionCommitAndRollback(t *testing.T) {
	t.Run("commit", func(t *testing.T) {
		stub := &stubConn{}
		pool := stubPool(stub)
		tx := DBBegin(pool)
		if !tx.OK {
			t.Fatalf("begin: %s", tx.Error)
		}
		DBExec(tx.Value, "INSERT ...")
		if c := DBCommit(tx.Value); !c.OK {
			t.Fatalf("commit: %s", c.Error)
		}
		// O rollback agendado com defer precisa virar no-op depois do commit.
		DBRollback(tx.Value)
		if got := strings.Join(stub.seen(), ","); got != "begin,exec,commit" {
			t.Fatalf("sequencia enviada: %s", got)
		}
	})

	t.Run("rollback em saida antecipada", func(t *testing.T) {
		stub := &stubConn{}
		tx := DBBegin(stubPool(stub))
		if !tx.OK {
			t.Fatalf("begin: %s", tx.Error)
		}
		DBRollback(tx.Value) // como se um try tivesse falhado no meio do bloco
		DBRollback(tx.Value) // idempotente
		if got := strings.Join(stub.seen(), ","); got != "begin,rollback" {
			t.Fatalf("sequencia enviada: %s", got)
		}
	})
}

func TestDBTransactionPinsOneConnection(t *testing.T) {
	stub := &stubConn{}
	pool := stubPool(stub)
	tx := DBBegin(pool)
	if !tx.OK {
		t.Fatal(tx.Error)
	}
	// Dentro da transacao toda consulta tem de cair na conexao fixada; se o
	// pool entregasse outra, o INSERT ficaria fora da transacao.
	for i := 0; i < 3; i++ {
		DBExec(tx.Value, "INSERT ...")
	}
	if len(pool.idle) != 0 {
		t.Error("a conexao da transacao voltou para o pool antes do commit")
	}
	DBCommit(tx.Value)
	if len(pool.idle) != 1 {
		t.Error("a conexao nao voltou ao pool depois do commit")
	}
}

func TestDBTransactionRefusesNesting(t *testing.T) {
	tx := DBBegin(stubPool(&stubConn{}))
	if !tx.OK {
		t.Fatal(tx.Error)
	}
	if nested := DBBegin(tx.Value); nested.OK {
		t.Fatal("transacao aninhada foi aceita")
	}
}

func TestPoolDiscardsBrokenConnections(t *testing.T) {
	broken := &stubConn{queryErr: errWrap(dbdrv.ErrConnLost)}
	pool := stubPool(broken)
	DBQuery[user](pool, "SELECT ...")
	// Uma conexao cuja falha foi de transporte nao pode voltar para o pool:
	// o proximo uso herdaria o socket morto.
	if len(pool.idle) != 0 {
		t.Error("conexao quebrada voltou ao pool")
	}
	if !broken.closed {
		t.Error("conexao quebrada nao foi fechada")
	}
}

func TestPoolKeepsConnectionAfterSQLError(t *testing.T) {
	// Erro de sintaxe ou de restricao deixa a sessao utilizavel; descartar a
	// conexao nesse caso esvaziaria o pool a cada consulta ruim.
	failing := &stubConn{queryErr: errors.New(`relation "users" does not exist`)}
	pool := stubPool(failing)
	DBQuery[user](pool, "SELECT ...")
	if len(pool.idle) != 1 {
		t.Error("conexao saudavel foi descartada apos erro do banco")
	}
}

func errWrap(err error) error { return wrapped{err} }

type wrapped struct{ err error }

func (w wrapped) Error() string { return "transport: " + w.err.Error() }
func (w wrapped) Unwrap() error { return w.err }

func TestDBCloseIsIdempotentAndBlocksReuse(t *testing.T) {
	stub := &stubConn{}
	pool := stubPool(stub)
	if r := DBClose(pool); !r.OK {
		t.Fatalf("close: %s", r.Error)
	}
	if r := DBClose(pool); !r.OK {
		t.Fatalf("close repetido devia ser no-op: %s", r.Error)
	}
	if !stub.closed {
		t.Error("a conexao ociosa nao foi fechada")
	}
	if r := DBQuery[user](pool, "SELECT ..."); r.OK {
		t.Error("consulta apos db-close foi aceita")
	}
}

func TestRedisOperationsRequireRedisDriver(t *testing.T) {
	pool := stubPool(&stubConn{}) // stubConn nao implementa dbdrv.KV
	if r := RedisGet(pool, "k"); r.OK || !strings.Contains(r.Error, "redis") {
		t.Errorf("redis-get num driver relacional: %+v", r)
	}
	if r := RedisSet(pool, "k", "v", 0); r.OK {
		t.Error("redis-set num driver relacional foi aceito")
	}
}

func TestDSNPoolSize(t *testing.T) {
	for _, tc := range []struct {
		dsn  string
		want int
	}{
		{"postgres://u@h/db?pool_max=4", 4},
		{"postgres://u@h/db?sslmode=require&pool_max=32", 32},
		{"postgres://u@h/db", 0},
		{"postgres://u@h/db?pool_max=abc", 0},
		{"postgres://u@h/db?pool_max=0", 0},
	} {
		if got := dsnPoolSize(tc.dsn); got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.dsn, got, tc.want)
		}
	}
}
