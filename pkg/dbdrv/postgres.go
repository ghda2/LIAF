package dbdrv

import (
	"bufio"
	"crypto/md5"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

// Protocolo de mensagens do PostgreSQL, versao 3.0.
// Referencia: https://www.postgresql.org/docs/current/protocol.html

const (
	pgProtocolVersion = 196608 // 3.0 em (major<<16 | minor)
	pgSSLRequestCode  = 80877103

	pgAuthOK                = 0
	pgAuthCleartextPassword = 3
	pgAuthMD5Password       = 5
	pgAuthSASL              = 10
	pgAuthSASLContinue      = 11
	pgAuthSASLFinal         = 12
)

// OIDs dos tipos cujo texto convem converter antes de entregar a LIAF.
// Sem isso todo campo chegaria como string e um campo int da struct
// receberia "42" em vez de 42.
const (
	oidBool    = 16
	oidInt8    = 20
	oidInt2    = 21
	oidInt4    = 23
	oidFloat4  = 700
	oidFloat8  = 701
	oidNumeric = 1700
)

type pgColumn struct {
	name string
	oid  uint32
}

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

// pgNegotiateTLS envia o SSLRequest antes do StartupMessage. O servidor
// responde com um unico byte: 'S' aceita, 'N' recusa.
func pgNegotiateTLS(raw net.Conn, host, sslmode string) (net.Conn, error) {
	switch sslmode {
	case "disable":
		return raw, nil
	case "prefer", "allow", "require", "verify-ca", "verify-full":
	default:
		raw.Close()
		return nil, fmt.Errorf("unsupported sslmode %q", sslmode)
	}

	_ = raw.SetDeadline(time.Now().Add(ioTimeout))
	request := make([]byte, 8)
	binary.BigEndian.PutUint32(request[0:4], 8)
	binary.BigEndian.PutUint32(request[4:8], pgSSLRequestCode)
	if _, err := raw.Write(request); err != nil {
		raw.Close()
		return nil, fmt.Errorf("could not send SSLRequest: %w", err)
	}
	answer := make([]byte, 1)
	if _, err := io.ReadFull(raw, answer); err != nil {
		raw.Close()
		return nil, fmt.Errorf("could not read the SSLRequest reply: %w", err)
	}
	if answer[0] != 'S' {
		if sslmode == "prefer" || sslmode == "allow" {
			return raw, nil
		}
		raw.Close()
		return nil, fmt.Errorf("server refused TLS but sslmode=%s requires it", sslmode)
	}

	// prefer e allow cifram o trafego mas nao autenticam o servidor: nesses
	// modos o libpq tambem nao verifica, e exigir verificacao aqui quebraria
	// o caso comum do certificado autoassinado em desenvolvimento.
	conf := &tls.Config{ServerName: host}
	if sslmode != "verify-ca" && sslmode != "verify-full" {
		conf.InsecureSkipVerify = true
	}
	tlsConn := tls.Client(raw, conf)
	if err := tlsConn.Handshake(); err != nil {
		raw.Close()
		return nil, fmt.Errorf("TLS handshake failed: %w", err)
	}
	return tlsConn, nil
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

// --- Handshake --------------------------------------------------------------

func (c *pgConn) startup(cfg *config) error {
	_ = c.conn.SetDeadline(time.Now().Add(ioTimeout))

	var body []byte
	body = binary.BigEndian.AppendUint32(body, pgProtocolVersion)
	body = pgAppendString(body, "user")
	body = pgAppendString(body, cfg.user)
	if cfg.database != "" {
		body = pgAppendString(body, "database")
		body = pgAppendString(body, cfg.database)
	}
	if app := cfg.param("application_name", "liaf"); app != "" {
		body = pgAppendString(body, "application_name")
		body = pgAppendString(body, app)
	}
	body = append(body, 0)

	// O StartupMessage e a unica mensagem sem byte de tipo.
	frame := binary.BigEndian.AppendUint32(nil, uint32(len(body)+4))
	if _, err := c.w.Write(append(frame, body...)); err != nil {
		return err
	}
	if err := c.w.Flush(); err != nil {
		return err
	}
	return c.authenticate(cfg)
}

func (c *pgConn) authenticate(cfg *config) error {
	var scram *scramClient
	for {
		kind, payload, err := c.receive()
		if err != nil {
			return err
		}
		switch kind {
		case 'R':
			if len(payload) < 4 {
				return fmt.Errorf("malformed authentication message")
			}
			code := binary.BigEndian.Uint32(payload[:4])
			rest := payload[4:]
			switch code {
			case pgAuthOK:
			case pgAuthCleartextPassword:
				if err := c.send('p', pgAppendString(nil, cfg.password)); err != nil {
					return err
				}
			case pgAuthMD5Password:
				if len(rest) < 4 {
					return fmt.Errorf("malformed MD5 authentication salt")
				}
				if err := c.send('p', pgAppendString(nil, pgMD5Password(cfg.user, cfg.password, rest[:4]))); err != nil {
					return err
				}
			case pgAuthSASL:
				if !pgOffersSCRAM(rest) {
					return fmt.Errorf("server offered no supported SASL mechanism (only SCRAM-SHA-256 is implemented)")
				}
				scram, err = newSCRAMClient(cfg.password)
				if err != nil {
					return err
				}
				first := scram.first()
				out := pgAppendString(nil, "SCRAM-SHA-256")
				out = binary.BigEndian.AppendUint32(out, uint32(len(first)))
				out = append(out, first...)
				if err := c.send('p', out); err != nil {
					return err
				}
			case pgAuthSASLContinue:
				if scram == nil {
					return fmt.Errorf("unexpected SASLContinue before SASL start")
				}
				final, err := scram.final(string(rest))
				if err != nil {
					return err
				}
				if err := c.send('p', []byte(final)); err != nil {
					return err
				}
			case pgAuthSASLFinal:
				if scram == nil {
					return fmt.Errorf("unexpected SASLFinal before SASL start")
				}
				if err := scram.verify(string(rest)); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported authentication method %d", code)
			}
		case 'S', 'K', 'N':
			// ParameterStatus, BackendKeyData e NoticeResponse nao mudam o
			// que a LIAF expoe; o cancelamento assincrono nao e suportado.
		case 'Z':
			return nil
		case 'E':
			return pgError(payload)
		default:
			return fmt.Errorf("unexpected message %q during handshake", string(kind))
		}
	}
}

func pgOffersSCRAM(payload []byte) bool {
	for _, mechanism := range strings.Split(string(payload), "\x00") {
		if mechanism == "SCRAM-SHA-256" {
			return true
		}
	}
	return false
}

// pgMD5Password produz "md5" + hex(md5(hex(md5(senha+usuario)) + salt)).
func pgMD5Password(user, password string, salt []byte) string {
	inner := md5.Sum([]byte(password + user))
	outer := md5.Sum(append([]byte(hex.EncodeToString(inner[:])), salt...))
	return "md5" + hex.EncodeToString(outer[:])
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

func pgRowDescription(payload []byte) []pgColumn {
	if len(payload) < 2 {
		return nil
	}
	count := int(binary.BigEndian.Uint16(payload[:2]))
	rest := payload[2:]
	columns := make([]pgColumn, 0, count)
	for i := 0; i < count; i++ {
		end := indexZero(rest)
		if end < 0 || len(rest) < end+1+18 {
			break
		}
		name := string(rest[:end])
		meta := rest[end+1:]
		// tableOID(4) columnIndex(2) typeOID(4) typeSize(2) modifier(4) format(2)
		columns = append(columns, pgColumn{name: name, oid: binary.BigEndian.Uint32(meta[6:10])})
		rest = meta[18:]
	}
	return columns
}

func pgDataRow(payload []byte, columns []pgColumn) ([]Value, error) {
	if len(payload) < 2 {
		return nil, fmt.Errorf("malformed DataRow")
	}
	count := int(binary.BigEndian.Uint16(payload[:2]))
	rest := payload[2:]
	values := make([]Value, 0, count)
	for i := 0; i < count; i++ {
		if len(rest) < 4 {
			return nil, fmt.Errorf("truncated DataRow")
		}
		size := int32(binary.BigEndian.Uint32(rest[:4]))
		rest = rest[4:]
		if size < 0 {
			values = append(values, nil)
			continue
		}
		if len(rest) < int(size) {
			return nil, fmt.Errorf("truncated DataRow field")
		}
		text := string(rest[:size])
		rest = rest[size:]
		oid := uint32(0)
		if i < len(columns) {
			oid = columns[i].oid
		}
		values = append(values, pgDecodeText(oid, text))
	}
	return values, nil
}

// pgDecodeText traduz o texto do servidor no tipo Go correspondente. Tipos
// nao listados ficam como string: e o que a struct LIAF espera para texto,
// data, json, uuid e enum.
func pgDecodeText(oid uint32, text string) Value {
	switch oid {
	case oidBool:
		return text == "t" || text == "true"
	case oidInt2, oidInt4, oidInt8:
		if v, err := strconv.ParseInt(text, 10, 64); err == nil {
			return v
		}
	case oidFloat4, oidFloat8, oidNumeric:
		if v, err := strconv.ParseFloat(text, 64); err == nil {
			return v
		}
	}
	return text
}

// pgRowsAffected le a etiqueta do CommandComplete: "INSERT 0 1", "UPDATE 3",
// "DELETE 2", "SELECT 7". O ultimo campo e sempre a contagem.
func pgRowsAffected(payload []byte) int64 {
	tag := strings.TrimRight(string(payload), "\x00")
	fields := strings.Fields(tag)
	if len(fields) == 0 {
		return 0
	}
	n, err := strconv.ParseInt(fields[len(fields)-1], 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// pgError monta a mensagem a partir dos campos do ErrorResponse, priorizando
// severidade, codigo SQLSTATE e mensagem — que e o que ajuda a corrigir.
func pgError(payload []byte) error {
	fields := map[byte]string{}
	rest := payload
	for len(rest) > 0 && rest[0] != 0 {
		code := rest[0]
		end := indexZero(rest[1:])
		if end < 0 {
			break
		}
		fields[code] = string(rest[1 : 1+end])
		rest = rest[1+end+1:]
	}
	message := fields['M']
	if message == "" {
		message = "unknown error"
	}
	parts := []string{message}
	if code := fields['C']; code != "" {
		parts = append(parts, "SQLSTATE "+code)
	}
	if detail := fields['D']; detail != "" {
		parts = append(parts, detail)
	}
	if hint := fields['H']; hint != "" {
		parts = append(parts, "hint: "+hint)
	}
	severity := fields['S']
	if severity == "" {
		severity = "ERROR"
	}
	return fmt.Errorf("postgres %s: %s", severity, strings.Join(parts, " | "))
}

// --- Enquadramento ----------------------------------------------------------

func indexZero(b []byte) int {
	for i, c := range b {
		if c == 0 {
			return i
		}
	}
	return -1
}

func pgAppendString(dst []byte, s string) []byte { return append(append(dst, s...), 0) }

// send escreve uma mensagem e ja faz flush: o protocolo e sincrono e o
// servidor so responde ao receber o Sync.
func (c *pgConn) send(kind byte, body []byte) error {
	c.buf = c.buf[:0]
	c.buf = append(c.buf, kind)
	c.buf = binary.BigEndian.AppendUint32(c.buf, uint32(len(body)+4))
	c.buf = append(c.buf, body...)
	if _, err := c.w.Write(c.buf); err != nil {
		return err
	}
	return c.w.Flush()
}

// Limite defensivo: uma mensagem corrompida nao deve virar uma alocacao de
// gigabytes antes de falhar.
const pgMaxMessageSize = 64 << 20

func (c *pgConn) receive() (byte, []byte, error) {
	header := make([]byte, 5)
	if _, err := io.ReadFull(c.r, header); err != nil {
		return 0, nil, fmt.Errorf("%w: reading from postgres: %v", ErrConnLost, err)
	}
	size := int(binary.BigEndian.Uint32(header[1:5]))
	if size < 4 || size-4 > pgMaxMessageSize {
		return 0, nil, fmt.Errorf("postgres sent an invalid message length (%d)", size)
	}
	body := make([]byte, size-4)
	if _, err := io.ReadFull(c.r, body); err != nil {
		return 0, nil, fmt.Errorf("%w: reading from postgres: %v", ErrConnLost, err)
	}
	return header[0], body, nil
}
