package dbdrv

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

// Protocolo de mensagens do PostgreSQL, versao 3.0.
// Referencia: https://www.postgresql.org/docs/current/protocol.html

type pgConn struct {
	conn net.Conn
	r    *bufio.Reader
	w    *bufio.Writer
	buf  []byte // reaproveitado na montagem de mensagens de saida
}

func openPostgres(dsn string) (Conn, error) {
	cfg, err := parseDSN(dsn, DriverPostgres, "5432")
	if err != nil {
		return nil, err
	}
	if cfg.user == "" {
		return nil, fmt.Errorf("postgres DSN requires a user")
	}
	raw, err := dial(cfg.address())
	if err != nil {
		return nil, err
	}
	raw, err = pgNegotiateTLS(raw, cfg.host, cfg.param("sslmode", "prefer"))
	if err != nil {
		return nil, err
	}

	c := &pgConn{conn: raw, r: bufio.NewReader(raw), w: bufio.NewWriter(raw)}
	if err := c.startup(cfg); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

func (c *pgConn) Close() error { return c.conn.Close() }

func (c *pgConn) Ping() error {
	_, err := c.simple("SELECT 1")
	return err
}

func (c *pgConn) Begin() error    { _, err := c.simple("BEGIN"); return err }
func (c *pgConn) Commit() error   { _, err := c.simple("COMMIT"); return err }
func (c *pgConn) Rollback() error { _, err := c.simple("ROLLBACK"); return err }

func (c *pgConn) Query(sql string, args []Value) (*Rows, error) {
	rows, _, err := c.extended(sql, args)
	return rows, err
}

func (c *pgConn) Exec(sql string, args []Value) (int64, error) {
	_, affected, err := c.extended(sql, args)
	return affected, err
}

// --- Consultas --------------------------------------------------------------

// simple usa o Simple Query ('Q'), reservado a comandos sem parametro que a
// propria LIAF emite (BEGIN/COMMIT/ROLLBACK). Toda consulta escrita pelo autor
// passa por extended, que envia os parametros fora do texto do comando.
func (c *pgConn) simple(sql string) (int64, error) {
	_ = c.conn.SetDeadline(time.Now().Add(ioTimeout))
	if err := c.send('Q', pgAppendString(nil, sql)); err != nil {
		return 0, err
	}
	var affected int64
	var failure error
	for {
		kind, payload, err := c.receive()
		if err != nil {
			return 0, err
		}
		switch kind {
		case 'C':
			affected = pgRowsAffected(payload)
		case 'E':
			failure = pgError(payload)
		case 'Z':
			return affected, failure
		}
	}
}

// extended usa Parse/Bind/Describe/Execute/Sync. Os parametros viajam como
// campos proprios da mensagem Bind, nunca concatenados ao SQL — e por isso
// que nenhum valor consegue virar sintaxe.
func (c *pgConn) extended(sql string, args []Value) (*Rows, int64, error) {
	_ = c.conn.SetDeadline(time.Now().Add(ioTimeout))

	// Parse: statement sem nome, sem tipos declarados (0) para que o
	// servidor infira o tipo de cada parametro pelo contexto da consulta.
	parse := pgAppendString(nil, "")
	parse = pgAppendString(parse, sql)
	parse = binary.BigEndian.AppendUint16(parse, 0)
	if err := c.send('P', parse); err != nil {
		return nil, 0, err
	}

	bind := pgAppendString(nil, "") // portal
	bind = pgAppendString(bind, "") // statement
	bind = binary.BigEndian.AppendUint16(bind, 0)
	bind = binary.BigEndian.AppendUint16(bind, uint16(len(args)))
	for _, a := range args {
		text, ok := textValue(a)
		if !ok {
			bind = binary.BigEndian.AppendUint32(bind, ^uint32(0)) // -1 = NULL
			continue
		}
		bind = binary.BigEndian.AppendUint32(bind, uint32(len(text)))
		bind = append(bind, text...)
	}
	bind = binary.BigEndian.AppendUint16(bind, 0) // resultados em formato texto
	if err := c.send('B', bind); err != nil {
		return nil, 0, err
	}
	if err := c.send('D', append([]byte{'P'}, pgAppendString(nil, "")...)); err != nil {
		return nil, 0, err
	}
	execute := pgAppendString(nil, "")
	execute = binary.BigEndian.AppendUint32(execute, 0) // sem limite de linhas
	if err := c.send('E', execute); err != nil {
		return nil, 0, err
	}
	if err := c.send('S', nil); err != nil {
		return nil, 0, err
	}

	var columns []pgColumn
	rows := &Rows{}
	var affected int64
	var failure error
	for {
		kind, payload, err := c.receive()
		if err != nil {
			return nil, 0, err
		}
		switch kind {
		case 'T':
			columns = pgRowDescription(payload)
			rows.Columns = make([]string, len(columns))
			for i, col := range columns {
				rows.Columns[i] = col.name
			}
		case 'D':
			values, err := pgDataRow(payload, columns)
			if err != nil {
				failure = err
				continue
			}
			rows.Values = append(rows.Values, values)
		case 'C':
			affected = pgRowsAffected(payload)
		case 'E':
			failure = pgError(payload)
		case 'Z':
			if failure != nil {
				return nil, 0, failure
			}
			return rows, affected, nil
		}
	}
}
