package dbdrv

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// redisConn fala RESP2 sobre TCP. RESP2 basta para GET/SET e e entendido por
// todo servidor Redis de 2.0 em diante, incluindo os que negociam RESP3.
type redisConn struct {
	conn net.Conn
	r    *bufio.Reader
	w    *bufio.Writer
}

func openRedis(dsn string) (Conn, error) {
	cfg, err := parseDSN(dsn, DriverRedis, "6379")
	if err != nil {
		return nil, err
	}
	raw, err := dial(cfg.address())
	if err != nil {
		return nil, err
	}
	c := &redisConn{conn: raw, r: bufio.NewReader(raw), w: bufio.NewWriter(raw)}

	if cfg.password != "" {
		// AUTH com dois argumentos exige Redis 6 (ACL). Sem usuario no DSN,
		// a forma de um argumento continua valendo para servidores antigos.
		args := []string{"AUTH"}
		if cfg.user != "" {
			args = append(args, cfg.user)
		}
		args = append(args, cfg.password)
		if _, err := c.command(args...); err != nil {
			c.Close()
			return nil, fmt.Errorf("redis authentication failed: %w", err)
		}
	}
	// O "banco" do Redis e o indice numerico no path do DSN: redis://host:6379/2
	if db := strings.Trim(cfg.database, "/"); db != "" && db != "0" {
		if _, err := strconv.Atoi(db); err != nil {
			c.Close()
			return nil, fmt.Errorf("redis database must be a number, received %q", db)
		}
		if _, err := c.command("SELECT", db); err != nil {
			c.Close()
			return nil, fmt.Errorf("redis SELECT failed: %w", err)
		}
	}
	return c, nil
}

func (c *redisConn) Close() error { return c.conn.Close() }

func (c *redisConn) Ping() error {
	_, err := c.command("PING")
	return err
}

func (c *redisConn) Get(key string) (string, bool, error) {
	reply, err := c.command("GET", key)
	if err != nil {
		return "", false, err
	}
	if reply == nil {
		return "", false, nil
	}
	s, ok := reply.(string)
	if !ok {
		return "", false, fmt.Errorf("redis GET returned an unexpected reply type")
	}
	return s, true, nil
}

func (c *redisConn) Set(key, value string, ttlSeconds int64) error {
	args := []string{"SET", key, value}
	if ttlSeconds > 0 {
		args = append(args, "EX", strconv.FormatInt(ttlSeconds, 10))
	}
	_, err := c.command(args...)
	return err
}

// As quatro operacoes relacionais nao existem no Redis. Falhar aqui com uma
// mensagem explicita e melhor do que deixar db-query devolver vazio.
var errRedisRelational = errors.New("the redis driver supports only redis-get and redis-set")

func (c *redisConn) Query(string, []Value) (*Rows, error) { return nil, errRedisRelational }
func (c *redisConn) Exec(string, []Value) (int64, error)  { return 0, errRedisRelational }
func (c *redisConn) Begin() error                         { return errRedisRelational }
func (c *redisConn) Commit() error                        { return errRedisRelational }
func (c *redisConn) Rollback() error                      { return errRedisRelational }

// command envia um comando como array RESP de bulk strings — a forma unificada,
// que nunca precisa de escape e por isso nao tem o equivalente de injecao.
func (c *redisConn) command(args ...string) (any, error) {
	_ = c.conn.SetDeadline(time.Now().Add(ioTimeout))
	fmt.Fprintf(c.w, "*%d\r\n", len(args))
	for _, a := range args {
		fmt.Fprintf(c.w, "$%d\r\n%s\r\n", len(a), a)
	}
	if err := c.w.Flush(); err != nil {
		return nil, err
	}
	return c.reply()
}

// reply devolve string, int64, []any ou nil. O erro do servidor (-ERR ...)
// vira erro Go.
func (c *redisConn) reply() (any, error) {
	prefix, err := c.r.ReadByte()
	if err != nil {
		return nil, err
	}
	line, err := c.line()
	if err != nil {
		return nil, err
	}
	switch prefix {
	case '+':
		return line, nil
	case '-':
		return nil, fmt.Errorf("redis: %s", line)
	case ':':
		return strconv.ParseInt(line, 10, 64)
	case '$':
		n, err := strconv.Atoi(line)
		if err != nil {
			return nil, fmt.Errorf("redis: invalid bulk length %q", line)
		}
		if n < 0 {
			return nil, nil
		}
		buf := make([]byte, n+2) // inclui o CRLF final
		if _, err := ioReadFull(c.r, buf); err != nil {
			return nil, err
		}
		return string(buf[:n]), nil
	case '*':
		n, err := strconv.Atoi(line)
		if err != nil {
			return nil, fmt.Errorf("redis: invalid array length %q", line)
		}
		if n < 0 {
			return nil, nil
		}
		items := make([]any, 0, n)
		for i := 0; i < n; i++ {
			item, err := c.reply()
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		return items, nil
	default:
		return nil, fmt.Errorf("redis: unsupported reply prefix %q", string(prefix))
	}
}

func (c *redisConn) line() (string, error) {
	s, err := c.r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(s, "\r\n"), nil
}

func ioReadFull(r *bufio.Reader, buf []byte) (int, error) {
	read := 0
	for read < len(buf) {
		n, err := r.Read(buf[read:])
		read += n
		if err != nil {
			return read, err
		}
	}
	return read, nil
}
