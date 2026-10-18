// Package dbdrv implementa, sobre TCP puro, os protocolos de rede do
// PostgreSQL, do MySQL e do Redis.
//
// A LIAF compila para um binario unico e nao quer depender de driver externo
// (issues #010 e #015), entao o protocolo vive aqui em vez de vir de pgx,
// go-sql-driver/mysql ou go-redis. O escopo e deliberadamente o que a
// biblioteca da linguagem expoe: conectar, consultar com parametros, executar
// e transacionar. Nao e um driver de proposito geral.
package dbdrv

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Value e o universo de valores que atravessa a fronteira do driver:
// nil, bool, int64, uint64, float64 ou string. Tudo o mais e convertido
// para string pelo driver antes de chegar aqui, para que a conversao de
// linha em struct seja a mesma nos tres bancos.
//
// E um alias, e nao um tipo proprio, para que o codigo gerado possa passar
// um []any direto, sem copia elemento a elemento.
type Value = any

// ErrConnLost marca a falha de transporte, por oposicao ao erro que o banco
// devolve pelo proprio protocolo. So a primeira invalida a conexao: um erro
// de sintaxe ou de restricao deixa a sessao utilizavel, e descartar a conexao
// nesse caso esvaziaria o pool a cada consulta ruim.
var ErrConnLost = errors.New("database connection lost")

// Rows e um resultado ja materializado. Os bancos suportados devolvem
// conjuntos pequenos nas consultas que a LIAF sabe expressar, e materializar
// evita expor um cursor — que precisaria de um tipo com ciclo de vida
// proprio na linguagem.
type Rows struct {
	Columns []string
	Values  [][]Value
}

// Conn e uma conexao unica, nao concorrente. O pool de pkg/runtime garante
// que so uma goroutine a usa por vez.
type Conn interface {
	Query(sql string, args []Value) (*Rows, error)
	Exec(sql string, args []Value) (int64, error)
	Begin() error
	Commit() error
	Rollback() error
	Ping() error
	Close() error
}

// KV e a face chave-valor, implementada so pelo Redis. db-query e db-exec
// nao fazem sentido ali, e redis-get/redis-set nao fazem sentido num banco
// relacional; separar as interfaces deixa o erro claro em vez de virar um
// metodo que sempre falha.
type KV interface {
	Get(key string) (string, bool, error)
	Set(key, value string, ttlSeconds int64) error
}

// Driver identifica o protocolo. A string vem direto do fonte LIAF, em
// (db-connect "postgres" dsn).
const (
	DriverPostgres = "postgres"
	DriverMySQL    = "mysql"
	DriverRedis    = "redis"
	DriverSQLite   = "sqlite"
)

// Drivers lista os protocolos aceitos, para mensagens de erro e para o
// checker manter a mesma lista sem duplicar literais.
var Drivers = []string{DriverPostgres, DriverMySQL, DriverRedis, DriverSQLite}

const (
	dialTimeout = 10 * time.Second
	ioTimeout   = 30 * time.Second
)

// Open conecta e autentica. O DSN aceito e a forma URL
// (postgres://user:senha@host:porta/banco?sslmode=require); o PostgreSQL
// tambem aceita a forma de palavras-chave (host=... user=...), que e o que
// a maioria dos tutoriais mostra.
func Open(driver, dsn string) (Conn, error) {
	switch driver {
	case DriverPostgres:
		return openPostgres(dsn)
	case DriverMySQL:
		return openMySQL(dsn)
	case DriverRedis:
		return openRedis(dsn)
	case DriverSQLite:
		return openSQLite(dsn)
	default:
		return nil, fmt.Errorf("unknown driver %q, expected one of %s", driver, strings.Join(Drivers, ", "))
	}
}

// config e o DSN ja normalizado, qualquer que seja a forma de origem.
type config struct {
	host     string
	port     string
	user     string
	password string
	database string
	params   map[string]string
}

func (c *config) param(name, fallback string) string {
	if v, ok := c.params[name]; ok && v != "" {
		return v
	}
	return fallback
}

func (c *config) address() string { return net.JoinHostPort(c.host, c.port) }

// parseDSN aceita a forma URL e, para o PostgreSQL, a forma de palavras-chave.
// defaultPort e usado quando o DSN nao traz porta.
func parseDSN(dsn, scheme, defaultPort string) (*config, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil, fmt.Errorf("empty DSN")
	}
	cfg := &config{host: "localhost", port: defaultPort, params: map[string]string{}}

	if !strings.Contains(dsn, "://") {
		if scheme != DriverPostgres {
			return nil, fmt.Errorf("DSN must be a URL such as %s://user:password@host:%s/database", scheme, defaultPort)
		}
		return parseKeywordDSN(dsn, cfg)
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid DSN: %w", err)
	}
	if h := u.Hostname(); h != "" {
		cfg.host = h
	}
	if p := u.Port(); p != "" {
		cfg.port = p
	}
	if u.User != nil {
		cfg.user = u.User.Username()
		if pw, ok := u.User.Password(); ok {
			cfg.password = pw
		}
	}
	cfg.database = strings.TrimPrefix(u.Path, "/")
	for k, vs := range u.Query() {
		if len(vs) > 0 {
			cfg.params[k] = vs[0]
		}
	}
	return cfg, nil
}

// parseKeywordDSN le a forma "host=localhost user=app password=s3cr3t dbname=app".
// Valores com espaco vao entre aspas simples, como no libpq.
func parseKeywordDSN(dsn string, cfg *config) (*config, error) {
	for _, field := range splitKeywordFields(dsn) {
		key, value, ok := strings.Cut(field, "=")
		if !ok {
			return nil, fmt.Errorf("invalid DSN fragment %q, expected key=value", field)
		}
		value = strings.Trim(value, "'")
		switch strings.TrimSpace(key) {
		case "host":
			cfg.host = value
		case "port":
			cfg.port = value
		case "user":
			cfg.user = value
		case "password":
			cfg.password = value
		case "dbname", "database":
			cfg.database = value
		default:
			cfg.params[strings.TrimSpace(key)] = value
		}
	}
	return cfg, nil
}

func splitKeywordFields(dsn string) []string {
	var fields []string
	var cur strings.Builder
	quoted := false
	for _, r := range dsn {
		switch {
		case r == '\'':
			quoted = !quoted
			cur.WriteRune(r)
		case (r == ' ' || r == '\t') && !quoted:
			if cur.Len() > 0 {
				fields = append(fields, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		fields = append(fields, cur.String())
	}
	return fields
}

func dial(address string) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", address, dialTimeout)
	if err != nil {
		return nil, fmt.Errorf("%w: could not connect to %s: %v", ErrConnLost, address, err)
	}
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.SetKeepAlive(true)
	}
	return conn, nil
}

// textValue converte um parametro LIAF na representacao textual usada pelos
// protocolos que enviam parametros como texto (PostgreSQL) e pelo Redis.
func textValue(v Value) (string, bool) {
	switch t := v.(type) {
	case nil:
		return "", false
	case bool:
		if t {
			return "t", true
		}
		return "f", true
	case int64:
		return strconv.FormatInt(t, 10), true
	case int:
		// Um literal inteiro do fonte LIAF chega como constante Go sem tipo,
		// que o ...any materializa como int e nao como int64.
		return strconv.Itoa(t), true
	case uint64:
		return strconv.FormatUint(t, 10), true
	case float64:
		return strconv.FormatFloat(t, 'g', -1, 64), true
	case string:
		return t, true
	default:
		return fmt.Sprint(t), true
	}
}
