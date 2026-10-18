package dbdrv

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

// Protocolo cliente/servidor do MySQL, versao 4.1 (a unica em uso desde 2004),
// com prepared statements binarios.
// Referencia: https://dev.mysql.com/doc/dev/mysql-server/latest/PAGE_PROTOCOL.html

const (
	myComQuery     = 0x03
	myComPing      = 0x0e
	myComStmtPrep  = 0x16
	myComStmtExec  = 0x17
	myComStmtClose = 0x19
)

type mysqlConn struct {
	conn     net.Conn
	r        *bufio.Reader
	w        *bufio.Writer
	sequence uint8
	caps     uint32
}

func openMySQL(dsn string) (Conn, error) {
	cfg, err := parseDSN(dsn, DriverMySQL, "3306")
	if err != nil {
		return nil, err
	}
	if cfg.user == "" {
		return nil, fmt.Errorf("mysql DSN requires a user")
	}
	raw, err := dial(cfg.address())
	if err != nil {
		return nil, err
	}
	c := &mysqlConn{conn: raw, r: bufio.NewReader(raw), w: bufio.NewWriter(raw)}
	if err := c.handshake(cfg); err != nil {
		raw.Close()
		return nil, err
	}
	return c, nil
}

func (c *mysqlConn) Close() error {
	return c.conn.Close()
}

func (c *mysqlConn) Ping() error {
	c.sequence = 0
	if err := c.write([]byte{myComPing}); err != nil {
		return err
	}
	_, err := c.readOK()
	return err
}

func (c *mysqlConn) Begin() error    { return c.command("START TRANSACTION") }
func (c *mysqlConn) Commit() error   { return c.command("COMMIT") }
func (c *mysqlConn) Rollback() error { return c.command("ROLLBACK") }

// command envia um COM_QUERY sem parametros. Usado so para os comandos de
// transacao que a propria LIAF emite.
func (c *mysqlConn) command(sql string) error {
	c.sequence = 0
	if err := c.write(append([]byte{myComQuery}, sql...)); err != nil {
		return err
	}
	_, err := c.readOK()
	return err
}

func (c *mysqlConn) Query(sql string, args []Value) (*Rows, error) {
	rows, _, err := c.statement(sql, args)
	return rows, err
}

func (c *mysqlConn) Exec(sql string, args []Value) (int64, error) {
	_, affected, err := c.statement(sql, args)
	return affected, err
}

// --- Prepared statements ----------------------------------------------------

// statement prepara, executa e fecha. Sem cache: a LIAF nao expoe o ciclo de
// vida de um statement, e manter um cache exigiria invalida-lo a cada DDL.
func (c *mysqlConn) statement(sql string, args []Value) (*Rows, int64, error) {
	_ = c.conn.SetDeadline(time.Now().Add(ioTimeout))

	c.sequence = 0
	if err := c.write(append([]byte{myComStmtPrep}, sql...)); err != nil {
		return nil, 0, err
	}
	p, err := c.read()
	if err != nil {
		return nil, 0, err
	}
	if len(p) == 0 || p[0] == myErr {
		return nil, 0, myError(p)
	}
	if len(p) < 12 {
		return nil, 0, fmt.Errorf("malformed COM_STMT_PREPARE response")
	}
	stmtID := binary.LittleEndian.Uint32(p[1:5])
	numColumns := int(binary.LittleEndian.Uint16(p[5:7]))
	numParams := int(binary.LittleEndian.Uint16(p[7:9]))
	defer c.closeStatement(stmtID)

	if numParams != len(args) {
		return nil, 0, fmt.Errorf("the statement takes %d parameters but %d were supplied", numParams, len(args))
	}
	if _, err := c.readColumns(numParams); err != nil {
		return nil, 0, err
	}
	columns, err := c.readColumns(numColumns)
	if err != nil {
		return nil, 0, err
	}

	c.sequence = 0
	if err := c.write(myExecutePacket(stmtID, args)); err != nil {
		return nil, 0, err
	}
	p, err = c.read()
	if err != nil {
		return nil, 0, err
	}
	if len(p) == 0 {
		return nil, 0, fmt.Errorf("empty COM_STMT_EXECUTE response")
	}
	if p[0] == myErr {
		return nil, 0, myError(p)
	}
	if p[0] == myOK {
		affected, _ := myReadLenenc(p[1:])
		return &Rows{}, int64(affected), nil
	}

	// Um result set reenvia as definicoes de coluna depois do EXECUTE.
	count, _ := myReadLenenc(p)
	columns, err = c.readColumns(int(count))
	if err != nil {
		return nil, 0, err
	}
	rows, err := c.readBinaryRows(columns)
	if err != nil {
		return nil, 0, err
	}
	return rows, int64(len(rows.Values)), nil
}

func (c *mysqlConn) closeStatement(stmtID uint32) {
	c.sequence = 0
	_ = c.write(binary.LittleEndian.AppendUint32([]byte{myComStmtClose}, stmtID))
}
