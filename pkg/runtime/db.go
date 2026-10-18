package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"liaf/pkg/dbdrv"
)

// DBConn e o valor por tras do tipo opaco DBConnection da LIAF. Um mesmo
// handle representa duas coisas: uma conexao ao pool, quando vem de
// db-connect, ou uma transacao fixada numa conexao, quando vem de
// db-transaction. Os dois casos tem o mesmo tipo na linguagem porque
// (db-query conn ...) precisa funcionar dentro e fora da transacao, sem o
// autor reescrever a chamada.
type DBConn struct {
	driver string
	dsn    string
	pool   *dbPool

	// closed e deste handle, nao do pool: um db-close fecha a conexao de quem
	// o chamou sem derrubar os outros handlers que dividem o mesmo pool.
	closed atomic.Bool

	// tx != nil marca este handle como transacao.
	tx     dbdrv.Conn
	txDone bool
}

// dbPool e compartilhado por todos os db-connect com o mesmo driver e DSN.
// A LIAF nao tem variavel global, entao o jeito natural de escrever um
// handler e chamar db-connect a cada requisicao; sem o compartilhamento,
// cada chamada abriria conexoes novas que ninguem fecharia.
type dbPool struct {
	key    string
	driver string
	dsn    string

	// sem limita conexoes simultaneas. Um pico de requisicoes nao deve abrir
	// mil conexoes e derrubar o banco; aqui ele espera por uma vaga.
	sem chan struct{}

	mu   sync.Mutex
	idle []dbdrv.Conn

	// refs conta os handles abertos. closed e escrito com poolsMu e mu
	// seguros, entao basta um dos dois para le-lo. A ordem de travamento e
	// sempre poolsMu antes de mu.
	refs   int
	closed bool
}

const (
	defaultMaxOpen = 16
	maxIdleConns   = 8
)

var (
	poolsMu sync.Mutex
	pools   = map[string]*dbPool{}
)

// DBConnect abre o pool e ja valida credenciais numa conexao real: um DSN
// errado deve falhar em db-connect, nao na primeira consulta. Com o mesmo
// driver e DSN, reaproveita o pool ja aberto e devolve um handle novo.
func DBConnect(driver, dsn string) Result[*DBConn, string] {
	key := driver + "::" + dsn
	if c := shareOpenPool(key); c != nil {
		return Ok[*DBConn, string](c)
	}

	// Um dial por DSN de cada vez: quem chega enquanto outro disca espera e
	// reaproveita o pool dele. Sem isso, a primeira rajada de requisicoes
	// abriria varias conexoes ao mesmo tempo e, num SQLite recem-criado, todas
	// disputariam a troca para WAL e falhariam com SQLITE_BUSY. O lock e por
	// chave, e nao poolsMu, para que um banco lento nao trave os outros.
	dialMu := dialLock(key)
	dialMu.Lock()
	defer dialMu.Unlock()
	if c := shareOpenPool(key); c != nil {
		return Ok[*DBConn, string](c)
	}

	conn, err := dbdrv.Open(driver, dsn)
	if err != nil {
		return Err[*DBConn, string](err.Error())
	}

	maxOpen := defaultMaxOpen
	if n := dsnPoolSize(dsn); n > 0 {
		maxOpen = n
	}
	p := &dbPool{key: key, driver: driver, dsn: dsn, sem: make(chan struct{}, maxOpen), refs: 1}
	p.idle = append(p.idle, conn)
	poolsMu.Lock()
	pools[key] = p
	poolsMu.Unlock()
	return Ok[*DBConn, string](newHandle(p))
}

// dialLocks guarda um mutex por DSN. Nao ha remocao: e um por DSN distinto
// usado pelo programa, e programas LIAF abrem poucos.
var dialLocks sync.Map

func dialLock(key string) *sync.Mutex {
	mu, _ := dialLocks.LoadOrStore(key, &sync.Mutex{})
	return mu.(*sync.Mutex)
}

func shareOpenPool(key string) *DBConn {
	poolsMu.Lock()
	defer poolsMu.Unlock()
	p := pools[key]
	if p == nil || p.closed {
		return nil
	}
	p.refs++
	return newHandle(p)
}

func newHandle(p *dbPool) *DBConn {
	return &DBConn{driver: p.driver, dsn: p.dsn, pool: p}
}

// dsnPoolSize le ?pool_max=N sem reabrir o parser de DSN do driver: e um
// ajuste opcional, e um valor ilegivel simplesmente cai no padrao.
func dsnPoolSize(dsn string) int {
	_, query, ok := strings.Cut(dsn, "?")
	if !ok {
		return 0
	}
	for _, pair := range strings.Split(query, "&") {
		if key, value, ok := strings.Cut(pair, "="); ok && key == "pool_max" {
			if n, err := strconv.Atoi(value); err == nil && n > 0 {
				return n
			}
		}
	}
	return 0
}

// DBClose fecha este handle e e idempotente. O pool so e fechado, com suas
// conexoes ociosas, quando o ultimo handle que o divide for fechado.
func DBClose(c *DBConn) Result[bool, string] {
	if c == nil {
		return Err[bool, string]("db-close received a connection that was never opened")
	}
	if c.tx != nil {
		return Err[bool, string]("db-close cannot be used on a transaction handle; the block commits or rolls back on its own")
	}
	if !c.closed.CompareAndSwap(false, true) {
		return Ok[bool, string](true)
	}

	p := c.pool
	poolsMu.Lock()
	p.refs--
	if p.refs > 0 {
		poolsMu.Unlock()
		return Ok[bool, string](true)
	}
	if pools[p.key] == p {
		delete(pools, p.key)
	}
	p.mu.Lock()
	p.closed = true
	idle := p.idle
	p.idle = nil
	p.mu.Unlock()
	poolsMu.Unlock()

	var failure error
	for _, conn := range idle {
		if err := conn.Close(); err != nil && failure == nil {
			failure = err
		}
	}
	if failure != nil {
		return Err[bool, string](failure.Error())
	}
	return Ok[bool, string](true)
}

// acquire devolve uma conexao utilizavel. Num handle de transacao devolve
// sempre a mesma conexao fixada, para que as consultas do bloco caiam dentro
// da transacao em vez de numa conexao paralela.
func (c *DBConn) acquire() (dbdrv.Conn, error) {
	if c == nil {
		return nil, errors.New("connection was never opened; match the result of db-connect first")
	}
	if c.tx != nil {
		if c.txDone {
			return nil, errors.New("this transaction has already finished")
		}
		return c.tx, nil
	}
	if c.closed.Load() {
		return nil, errors.New("connection is closed")
	}
	return c.pool.acquire()
}

func (p *dbPool) acquire() (dbdrv.Conn, error) {
	p.sem <- struct{}{}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		<-p.sem
		return nil, errors.New("connection is closed")
	}
	if n := len(p.idle); n > 0 {
		conn := p.idle[n-1]
		p.idle = p.idle[:n-1]
		p.mu.Unlock()
		return conn, nil
	}
	p.mu.Unlock()

	conn, err := dbdrv.Open(p.driver, p.dsn)
	if err != nil {
		<-p.sem
		return nil, err
	}
	return conn, nil
}

// release devolve a conexao ao pool. Uma conexao cuja falha foi de rede nao
// volta para o pool: o proximo uso herdaria o estado quebrado.
func (c *DBConn) release(conn dbdrv.Conn, err error) {
	if c.tx != nil {
		return
	}
	c.pool.release(conn, err)
}

func (p *dbPool) release(conn dbdrv.Conn, err error) {
	broken := errors.Is(err, dbdrv.ErrConnLost)
	p.mu.Lock()
	if !broken && !p.closed && len(p.idle) < maxIdleConns {
		p.idle = append(p.idle, conn)
		p.mu.Unlock()
		<-p.sem
		return
	}
	p.mu.Unlock()
	_ = conn.Close()
	<-p.sem
}

// DBQuery executa a consulta e converte cada linha na struct T. O SQL e os
// parametros viajam separados ate o banco: o driver nunca costura valor
// dentro do texto do comando.
func DBQuery[T any](c *DBConn, sql string, args ...any) Result[*List[T], string] {
	conn, err := c.acquire()
	if err != nil {
		return Err[*List[T], string](err.Error())
	}
	rows, err := conn.Query(sql, args)
	c.release(conn, err)
	if err != nil {
		return Err[*List[T], string](err.Error())
	}

	items, err := decodeRows[T](rows)
	if err != nil {
		return Err[*List[T], string](err.Error())
	}
	return Ok[*List[T], string](&List[T]{Items: items})
}

// DBExec executa um comando que nao devolve linhas e informa quantas foram
// afetadas.
func DBExec(c *DBConn, sql string, args ...any) Result[int64, string] {
	conn, err := c.acquire()
	if err != nil {
		return Err[int64, string](err.Error())
	}
	affected, err := conn.Exec(sql, args)
	c.release(conn, err)
	if err != nil {
		return Err[int64, string](err.Error())
	}
	return Ok[int64, string](affected)
}

// decodeRows converte o resultado em structs passando por JSON. Reaproveitar
// as tags json que o codegen ja emite em cada struct LIAF evita uma segunda
// tabela de nomes de campo, que poderia divergir da primeira.
func decodeRows[T any](rows *dbdrv.Rows) ([]T, error) {
	items := []T{}
	if rows == nil || len(rows.Values) == 0 {
		return items, nil
	}
	objects := make([]map[string]any, 0, len(rows.Values))
	for _, row := range rows.Values {
		object := make(map[string]any, len(rows.Columns)*2)
		for i, column := range rows.Columns {
			if i >= len(row) {
				break
			}
			object[column] = row[i]
			// Coluna snake_case alimenta tambem o campo kebab-case, que e
			// como a LIAF escreve nomes compostos. Sem este apelido,
			// next_id nunca encontraria o campo next-id.
			if alias := strings.ReplaceAll(column, "_", "-"); alias != column {
				if _, taken := object[alias]; !taken {
					object[alias] = row[i]
				}
			}
		}
		objects = append(objects, object)
	}
	data, err := json.Marshal(objects)
	if err != nil {
		return nil, fmt.Errorf("could not convert the result set: %w", err)
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("the columns returned do not fit the requested type (%w); select the columns whose names match the struct fields", err)
	}
	return items, nil
}

// --- Transacoes -------------------------------------------------------------

// DBBegin fixa uma conexao e abre a transacao. O handle devolvido e o que o
// corpo de (db-transaction ...) enxerga com o nome da conexao original.
func DBBegin(c *DBConn) Result[*DBConn, string] {
	if c == nil {
		return Err[*DBConn, string]("connection was never opened; match the result of db-connect first")
	}
	if c.tx != nil {
		return Err[*DBConn, string]("nested transactions are not supported")
	}
	conn, err := c.acquire()
	if err != nil {
		return Err[*DBConn, string](err.Error())
	}
	if err := conn.Begin(); err != nil {
		c.release(conn, err)
		return Err[*DBConn, string](err.Error())
	}
	return Ok[*DBConn, string](&DBConn{driver: c.driver, dsn: c.dsn, pool: c.pool, tx: conn})
}

// DBCommit confirma e devolve a conexao ao pool.
func DBCommit(tx *DBConn) Result[bool, string] {
	if tx == nil || tx.tx == nil {
		return Err[bool, string]("commit outside a transaction")
	}
	if tx.txDone {
		return Err[bool, string]("this transaction has already finished")
	}
	tx.txDone = true
	err := tx.tx.Commit()
	tx.pool.release(tx.tx, err)
	if err != nil {
		return Err[bool, string](err.Error())
	}
	return Ok[bool, string](true)
}

// DBRollback desfaz a transacao e e idempotente de proposito: o codegen a
// agenda com defer para cobrir qualquer saida antecipada do bloco, e ela
// precisa virar no-op quando o commit ja aconteceu.
func DBRollback(tx *DBConn) {
	if tx == nil || tx.tx == nil || tx.txDone {
		return
	}
	tx.txDone = true
	err := tx.tx.Rollback()
	tx.pool.release(tx.tx, err)
}

// --- Redis ------------------------------------------------------------------

func redisHandle(c *DBConn) (dbdrv.Conn, dbdrv.KV, error) {
	conn, err := c.acquire()
	if err != nil {
		return nil, nil, err
	}
	kv, ok := conn.(dbdrv.KV)
	if !ok {
		c.release(conn, nil)
		return nil, nil, fmt.Errorf("redis-get and redis-set require a connection opened with driver %q", dbdrv.DriverRedis)
	}
	return conn, kv, nil
}

// RedisGet trata chave ausente como erro porque o tipo declarado na issue e
// (result str str): nao ha um terceiro caso para representar a ausencia.
func RedisGet(c *DBConn, key string) Result[string, string] {
	conn, kv, err := redisHandle(c)
	if err != nil {
		return Err[string, string](err.Error())
	}
	value, found, err := kv.Get(key)
	c.release(conn, err)
	if err != nil {
		return Err[string, string](err.Error())
	}
	if !found {
		return Err[string, string]("redis key not found: " + key)
	}
	return Ok[string, string](value)
}

// RedisSet grava a chave. ttlSeconds menor ou igual a zero grava sem
// expiracao.
func RedisSet(c *DBConn, key, value string, ttlSeconds int64) Result[bool, string] {
	conn, kv, err := redisHandle(c)
	if err != nil {
		return Err[bool, string](err.Error())
	}
	err = kv.Set(key, value, ttlSeconds)
	c.release(conn, err)
	if err != nil {
		return Err[bool, string](err.Error())
	}
	return Ok[bool, string](true)
}
