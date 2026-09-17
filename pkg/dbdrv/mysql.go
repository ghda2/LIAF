package dbdrv

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io"
	"math"
	"net"
	"strconv"
	"strings"
	"time"
)

// Protocolo cliente/servidor do MySQL, versao 4.1 (a unica em uso desde 2004),
// com prepared statements binarios.
// Referencia: https://dev.mysql.com/doc/dev/mysql-server/latest/PAGE_PROTOCOL.html

const (
	myCapLongPassword     = 0x00000001
	myCapFoundRows        = 0x00000002
	myCapLongFlag         = 0x00000004
	myCapConnectWithDB    = 0x00000008
	myCapLocalFiles       = 0x00000080
	myCapProtocol41       = 0x00000200
	myCapSSL              = 0x00000800
	myCapTransactions     = 0x00002000
	myCapSecureConnection = 0x00008000
	myCapPluginAuth       = 0x00080000
	myCapPluginAuthLenenc = 0x00200000
	myCapDeprecateEOF     = 0x01000000

	myComQuery      = 0x03
	myComPing       = 0x0e
	myComStmtPrep   = 0x16
	myComStmtExec   = 0x17
	myComStmtClose  = 0x19

	myOK           = 0x00
	myEOF          = 0xfe
	myErr          = 0xff
	myAuthMoreData = 0x01

	myMaxPacketSize = 1<<24 - 1
	myCharsetUTF8   = 45 // utf8mb4_general_ci
)

// Tipos de coluna do protocolo binario.
const (
	myTypeDecimal    = 0x00
	myTypeTiny       = 0x01
	myTypeShort      = 0x02
	myTypeLong       = 0x03
	myTypeFloat      = 0x04
	myTypeDouble     = 0x05
	myTypeNull       = 0x06
	myTypeTimestamp  = 0x07
	myTypeLongLong   = 0x08
	myTypeInt24      = 0x09
	myTypeDate       = 0x0a
	myTypeTime       = 0x0b
	myTypeDatetime   = 0x0c
	myTypeYear       = 0x0d
	myTypeNewDecimal = 0xf6
	myTypeVarString  = 0xfd
	myTypeString     = 0xfe

	myFlagUnsigned = 0x0020
)

type myColumn struct {
	name     string
	kind     byte
	unsigned bool
	length   uint32
}

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

// --- Handshake --------------------------------------------------------------

func (c *mysqlConn) handshake(cfg *config) error {
	_ = c.conn.SetDeadline(time.Now().Add(ioTimeout))

	greeting, err := c.read()
	if err != nil {
		return fmt.Errorf("could not read the MySQL greeting: %w", err)
	}
	if len(greeting) > 0 && greeting[0] == myErr {
		return myError(greeting)
	}
	salt, plugin, serverCaps, err := parseGreeting(greeting)
	if err != nil {
		return err
	}

	c.caps = myCapLongPassword | myCapLongFlag | myCapProtocol41 | myCapTransactions |
		myCapSecureConnection | myCapPluginAuth | myCapPluginAuthLenenc | myCapFoundRows
	c.caps &^= myCapLocalFiles // LOAD DATA LOCAL le arquivos do cliente a pedido do servidor
	if serverCaps&myCapDeprecateEOF != 0 {
		c.caps |= myCapDeprecateEOF
	}
	if cfg.database != "" {
		c.caps |= myCapConnectWithDB
	}

	secure := false
	tlsMode := cfg.param("tls", "preferred")
	if tlsMode != "false" && tlsMode != "disable" && serverCaps&myCapSSL != 0 {
		c.caps |= myCapSSL
		if err := c.write(c.authHeader()); err != nil {
			return err
		}
		conf := &tls.Config{ServerName: cfg.host}
		if tlsMode != "verify" && tlsMode != "true" {
			conf.InsecureSkipVerify = true
		}
		tlsConn := tls.Client(c.conn, conf)
		if err := tlsConn.Handshake(); err != nil {
			return fmt.Errorf("TLS handshake failed: %w", err)
		}
		c.conn = tlsConn
		c.r = bufio.NewReader(tlsConn)
		c.w = bufio.NewWriter(tlsConn)
		secure = true
	} else if tlsMode == "true" || tlsMode == "verify" {
		return fmt.Errorf("server does not offer TLS but tls=%s requires it", tlsMode)
	}

	if plugin == "" {
		plugin = "mysql_native_password"
	}
	response, err := myAuthResponse(plugin, cfg.password, salt)
	if err != nil {
		return err
	}
	if err := c.write(c.authResponse(cfg, plugin, response)); err != nil {
		return err
	}
	return c.finishAuth(cfg, plugin, salt, secure)
}

// parseGreeting le o Initial Handshake Packet v10.
func parseGreeting(p []byte) (salt []byte, plugin string, caps uint32, err error) {
	if len(p) < 1 || p[0] != 10 {
		return nil, "", 0, fmt.Errorf("unsupported MySQL handshake (only protocol 10 is implemented)")
	}
	end := indexZero(p[1:])
	if end < 0 {
		return nil, "", 0, fmt.Errorf("malformed MySQL handshake")
	}
	rest := p[1+end+1:]
	if len(rest) < 8+1+2+1+2 {
		return nil, "", 0, fmt.Errorf("truncated MySQL handshake")
	}
	rest = rest[4:] // connection id
	salt = append(salt, rest[:8]...)
	rest = rest[8+1:] // scramble parte 1 + filler
	caps = uint32(binary.LittleEndian.Uint16(rest[:2]))
	rest = rest[2:]
	if len(rest) < 1 {
		return salt, "", caps, nil
	}
	if len(rest) < 1+2+2+1+10 {
		return nil, "", 0, fmt.Errorf("truncated MySQL handshake")
	}
	rest = rest[1+2:] // charset + status flags
	caps |= uint32(binary.LittleEndian.Uint16(rest[:2])) << 16
	rest = rest[2:]
	saltLen := int(rest[0])
	rest = rest[1+10:] // tamanho do scramble + reservado
	if saltLen > 8 {
		take := saltLen - 8
		if take > len(rest) {
			take = len(rest)
		}
		// O ultimo byte do scramble e um NUL terminador, nao entropia.
		salt = append(salt, bytes.TrimRight(rest[:take], "\x00")...)
		rest = rest[take:]
	}
	if caps&myCapPluginAuth != 0 && len(rest) > 0 {
		plugin = string(bytes.TrimRight(rest, "\x00"))
	}
	return salt, plugin, caps, nil
}

// authHeader e o prefixo de 32 bytes comum ao SSLRequest e ao
// HandshakeResponse41. Enviado sozinho, e o pedido de upgrade para TLS.
func (c *mysqlConn) authHeader() []byte {
	out := make([]byte, 0, 32)
	out = binary.LittleEndian.AppendUint32(out, c.caps)
	out = binary.LittleEndian.AppendUint32(out, myMaxPacketSize)
	out = append(out, myCharsetUTF8)
	return append(out, make([]byte, 23)...)
}

func (c *mysqlConn) authResponse(cfg *config, plugin string, auth []byte) []byte {
	out := c.authHeader()
	out = append(append(out, cfg.user...), 0)
	out = append(myAppendLenenc(out, uint64(len(auth))), auth...)
	if c.caps&myCapConnectWithDB != 0 {
		out = append(append(out, cfg.database...), 0)
	}
	if c.caps&myCapPluginAuth != 0 {
		out = append(append(out, plugin...), 0)
	}
	return out
}

// myAuthResponse calcula a prova de conhecimento da senha sem transmiti-la.
func myAuthResponse(plugin, password string, salt []byte) ([]byte, error) {
	if password == "" {
		return nil, nil
	}
	switch plugin {
	case "mysql_native_password":
		// SHA1(senha) XOR SHA1(salt + SHA1(SHA1(senha)))
		first := sha1.Sum([]byte(password))
		second := sha1.Sum(first[:])
		mac := sha1.New()
		mac.Write(salt)
		mac.Write(second[:])
		mask := mac.Sum(nil)
		out := make([]byte, len(first))
		for i := range first {
			out[i] = first[i] ^ mask[i]
		}
		return out, nil
	case "caching_sha2_password":
		// XOR(SHA256(senha), SHA256(SHA256(SHA256(senha)) + salt))
		first := sha256.Sum256([]byte(password))
		second := sha256.Sum256(first[:])
		mac := sha256.New()
		mac.Write(second[:])
		mac.Write(salt)
		mask := mac.Sum(nil)
		out := make([]byte, len(first))
		for i := range first {
			out[i] = first[i] ^ mask[i]
		}
		return out, nil
	case "mysql_clear_password":
		return append([]byte(password), 0), nil
	default:
		return nil, fmt.Errorf("unsupported MySQL authentication plugin %q", plugin)
	}
}

// finishAuth trata a troca extra que caching_sha2_password pode pedir e o
// AuthSwitchRequest, que o servidor manda quando o plugin adivinhado nao e o
// configurado para o usuario.
func (c *mysqlConn) finishAuth(cfg *config, plugin string, salt []byte, secure bool) error {
	for {
		p, err := c.read()
		if err != nil {
			return err
		}
		if len(p) == 0 {
			return fmt.Errorf("empty authentication packet")
		}
		switch p[0] {
		case myOK:
			return nil
		case myErr:
			return myError(p)
		case myEOF:
			// AuthSwitchRequest: plugin\0 + salt
			rest := p[1:]
			end := indexZero(rest)
			if end < 0 {
				return fmt.Errorf("malformed authentication switch request")
			}
			plugin = string(rest[:end])
			salt = bytes.TrimRight(rest[end+1:], "\x00")
			response, err := myAuthResponse(plugin, cfg.password, salt)
			if err != nil {
				return err
			}
			if err := c.write(response); err != nil {
				return err
			}
		case myAuthMoreData:
			if plugin != "caching_sha2_password" || len(p) < 2 {
				return fmt.Errorf("unexpected AuthMoreData from server")
			}
			switch p[1] {
			case 0x03: // fast auth: a senha ja estava no cache do servidor
			case 0x04: // full auth: o servidor precisa da senha em claro
				if err := c.fullSHA256Auth(cfg.password, salt, secure); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported caching_sha2_password state 0x%02x", p[1])
			}
		default:
			return fmt.Errorf("unexpected authentication packet 0x%02x", p[0])
		}
	}
}

// fullSHA256Auth executa o caminho lento do caching_sha2_password. Sobre TLS
// a senha vai em claro dentro do tunel; sem TLS ela e cifrada com a chave
// publica RSA do servidor, que nunca deve trafegar em claro.
func (c *mysqlConn) fullSHA256Auth(password string, salt []byte, secure bool) error {
	if secure {
		return c.write(append([]byte(password), 0))
	}
	if err := c.write([]byte{0x02}); err != nil { // request public key
		return err
	}
	p, err := c.read()
	if err != nil {
		return err
	}
	if len(p) < 2 || p[0] != myAuthMoreData {
		return fmt.Errorf("server did not send its RSA public key")
	}
	block, _ := pem.Decode(p[1:])
	if block == nil {
		return fmt.Errorf("could not decode the server RSA public key")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("invalid server RSA public key: %w", err)
	}
	pub, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("server public key is not RSA")
	}
	plain := append([]byte(password), 0)
	for i := range plain {
		plain[i] ^= salt[i%len(salt)]
	}
	cipher, err := rsa.EncryptOAEP(sha1.New(), rand.Reader, pub, plain, nil)
	if err != nil {
		return fmt.Errorf("could not encrypt the password: %w", err)
	}
	return c.write(cipher)
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

func (c *mysqlConn) readColumns(n int) ([]myColumn, error) {
	columns := make([]myColumn, 0, n)
	for i := 0; i < n; i++ {
		p, err := c.read()
		if err != nil {
			return nil, err
		}
		if len(p) > 0 && p[0] == myErr {
			return nil, myError(p)
		}
		col, err := parseColumn(p)
		if err != nil {
			return nil, err
		}
		columns = append(columns, col)
	}
	if n > 0 && c.caps&myCapDeprecateEOF == 0 {
		if _, err := c.read(); err != nil { // EOF packet
			return nil, err
		}
	}
	return columns, nil
}

// parseColumn le o ColumnDefinition41: seis strings de tamanho variavel
// (catalog, schema, tabela, tabela original, nome, nome original) seguidas
// dos metadados de tipo.
func parseColumn(p []byte) (myColumn, error) {
	rest := p
	var name string
	for i := 0; i < 6; i++ {
		s, tail, ok := myReadLenencString(rest)
		if !ok {
			return myColumn{}, fmt.Errorf("malformed column definition")
		}
		if i == 4 {
			name = s
		}
		rest = tail
	}
	if len(rest) < 1+2+4+1+2 {
		return myColumn{}, fmt.Errorf("truncated column definition")
	}
	rest = rest[1+2:] // tamanho dos campos fixos + charset
	length := binary.LittleEndian.Uint32(rest[:4])
	kind := rest[4]
	flags := binary.LittleEndian.Uint16(rest[5:7])
	return myColumn{name: name, kind: kind, unsigned: flags&myFlagUnsigned != 0, length: length}, nil
}

// myExecutePacket monta o COM_STMT_EXECUTE. Os parametros vao num bloco
// tipado, separado do texto preparado: nenhum valor pode virar sintaxe.
func myExecutePacket(stmtID uint32, args []Value) []byte {
	out := binary.LittleEndian.AppendUint32([]byte{myComStmtExec}, stmtID)
	out = append(out, 0)                                    // CURSOR_TYPE_NO_CURSOR
	out = binary.LittleEndian.AppendUint32(out, 1)          // iteration count
	if len(args) == 0 {
		return out
	}

	nullMask := make([]byte, (len(args)+7)/8)
	for i, a := range args {
		if a == nil {
			nullMask[i/8] |= 1 << (i % 8)
		}
	}
	out = append(out, nullMask...)
	out = append(out, 1) // new-params-bound-flag

	var values []byte
	for _, a := range args {
		switch v := a.(type) {
		case nil:
			out = append(out, myTypeNull, 0)
		case bool:
			out = append(out, myTypeTiny, 0)
			if v {
				values = append(values, 1)
			} else {
				values = append(values, 0)
			}
		case int64:
			out = append(out, myTypeLongLong, 0)
			values = binary.LittleEndian.AppendUint64(values, uint64(v))
		case int:
			// Literal inteiro do fonte LIAF: chega como int, nao como int64.
			out = append(out, myTypeLongLong, 0)
			values = binary.LittleEndian.AppendUint64(values, uint64(v))
		case uint64:
			out = append(out, myTypeLongLong, 0x80)
			values = binary.LittleEndian.AppendUint64(values, v)
		case float64:
			out = append(out, myTypeDouble, 0)
			values = binary.LittleEndian.AppendUint64(values, math.Float64bits(v))
		default:
			text, _ := textValue(v)
			out = append(out, myTypeVarString, 0)
			values = append(myAppendLenenc(values, uint64(len(text))), text...)
		}
	}
	return append(out, values...)
}

func (c *mysqlConn) readBinaryRows(columns []myColumn) (*Rows, error) {
	rows := &Rows{Columns: make([]string, len(columns))}
	for i, col := range columns {
		rows.Columns[i] = col.name
	}
	for {
		p, err := c.read()
		if err != nil {
			return nil, err
		}
		if len(p) == 0 {
			return nil, fmt.Errorf("empty row packet")
		}
		if p[0] == myErr {
			return nil, myError(p)
		}
		// Um pacote curto comecando em 0xfe encerra o result set — em 0x00
		// quando o servidor negociou DEPRECATE_EOF, e ai o OK vem no lugar.
		if p[0] == myEOF && len(p) < 9 {
			return rows, nil
		}
		if p[0] == myOK && c.caps&myCapDeprecateEOF != 0 && len(p) < 9 && len(columns) == 0 {
			return rows, nil
		}
		values, err := myDecodeBinaryRow(p, columns)
		if err != nil {
			return nil, err
		}
		rows.Values = append(rows.Values, values)
	}
}

// myDecodeBinaryRow le uma linha do protocolo binario: cabecalho 0x00, mapa
// de nulos deslocado em 2 bits, e os valores em sequencia.
func myDecodeBinaryRow(p []byte, columns []myColumn) ([]Value, error) {
	maskLen := (len(columns) + 9) / 8
	if len(p) < 1+maskLen {
		return nil, fmt.Errorf("truncated binary row")
	}
	mask := p[1 : 1+maskLen]
	rest := p[1+maskLen:]
	values := make([]Value, 0, len(columns))
	for i, col := range columns {
		offset := i + 2 // os dois primeiros bits do mapa sao reservados
		if mask[offset/8]&(1<<(offset%8)) != 0 {
			values = append(values, nil)
			continue
		}
		v, tail, err := myDecodeBinaryValue(col, rest)
		if err != nil {
			return nil, err
		}
		values = append(values, v)
		rest = tail
	}
	return values, nil
}

func myDecodeBinaryValue(col myColumn, b []byte) (Value, []byte, error) {
	need := func(n int) error {
		if len(b) < n {
			return fmt.Errorf("truncated value for column %q", col.name)
		}
		return nil
	}
	switch col.kind {
	case myTypeTiny:
		if err := need(1); err != nil {
			return nil, nil, err
		}
		// TINYINT(1) e como o MySQL guarda BOOLEAN.
		if col.length == 1 && !col.unsigned {
			return b[0] != 0, b[1:], nil
		}
		if col.unsigned {
			return uint64(b[0]), b[1:], nil
		}
		return int64(int8(b[0])), b[1:], nil
	case myTypeShort, myTypeYear:
		if err := need(2); err != nil {
			return nil, nil, err
		}
		raw := binary.LittleEndian.Uint16(b[:2])
		if col.unsigned {
			return uint64(raw), b[2:], nil
		}
		return int64(int16(raw)), b[2:], nil
	case myTypeLong, myTypeInt24:
		if err := need(4); err != nil {
			return nil, nil, err
		}
		raw := binary.LittleEndian.Uint32(b[:4])
		if col.unsigned {
			return uint64(raw), b[4:], nil
		}
		return int64(int32(raw)), b[4:], nil
	case myTypeLongLong:
		if err := need(8); err != nil {
			return nil, nil, err
		}
		raw := binary.LittleEndian.Uint64(b[:8])
		if col.unsigned {
			return raw, b[8:], nil
		}
		return int64(raw), b[8:], nil
	case myTypeFloat:
		if err := need(4); err != nil {
			return nil, nil, err
		}
		return float64(math.Float32frombits(binary.LittleEndian.Uint32(b[:4]))), b[4:], nil
	case myTypeDouble:
		if err := need(8); err != nil {
			return nil, nil, err
		}
		return math.Float64frombits(binary.LittleEndian.Uint64(b[:8])), b[8:], nil
	case myTypeDate, myTypeDatetime, myTypeTimestamp:
		return myDecodeDate(b)
	case myTypeTime:
		return myDecodeTime(b)
	case myTypeDecimal, myTypeNewDecimal:
		s, tail, ok := myReadLenencString(b)
		if !ok {
			return nil, nil, fmt.Errorf("truncated decimal for column %q", col.name)
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, tail, nil
		}
		return s, tail, nil
	default:
		s, tail, ok := myReadLenencString(b)
		if !ok {
			return nil, nil, fmt.Errorf("truncated value for column %q", col.name)
		}
		return s, tail, nil
	}
}

// myDecodeDate devolve a data em ISO-8601, que e o que json.Unmarshal aceita
// num campo str da struct LIAF.
func myDecodeDate(b []byte) (Value, []byte, error) {
	if len(b) < 1 {
		return nil, nil, fmt.Errorf("truncated date")
	}
	n := int(b[0])
	if len(b) < 1+n {
		return nil, nil, fmt.Errorf("truncated date")
	}
	body, tail := b[1:1+n], b[1+n:]
	switch {
	case n == 0:
		return "", tail, nil
	case n >= 11:
		micro := binary.LittleEndian.Uint32(body[7:11])
		return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d.%06d",
			binary.LittleEndian.Uint16(body[:2]), body[2], body[3], body[4], body[5], body[6], micro), tail, nil
	case n >= 7:
		return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d",
			binary.LittleEndian.Uint16(body[:2]), body[2], body[3], body[4], body[5], body[6]), tail, nil
	default:
		return fmt.Sprintf("%04d-%02d-%02d", binary.LittleEndian.Uint16(body[:2]), body[2], body[3]), tail, nil
	}
}

func myDecodeTime(b []byte) (Value, []byte, error) {
	if len(b) < 1 {
		return nil, nil, fmt.Errorf("truncated time")
	}
	n := int(b[0])
	if len(b) < 1+n {
		return nil, nil, fmt.Errorf("truncated time")
	}
	body, tail := b[1:1+n], b[1+n:]
	if n == 0 {
		return "00:00:00", tail, nil
	}
	sign := ""
	if body[0] == 1 {
		sign = "-"
	}
	days := binary.LittleEndian.Uint32(body[1:5])
	hours := uint32(body[5]) + days*24
	return fmt.Sprintf("%s%02d:%02d:%02d", sign, hours, body[6], body[7]), tail, nil
}

// --- Enquadramento ----------------------------------------------------------

func (c *mysqlConn) readOK() ([]byte, error) {
	p, err := c.read()
	if err != nil {
		return nil, err
	}
	if len(p) > 0 && p[0] == myErr {
		return nil, myError(p)
	}
	return p, nil
}

// write divide a carga em pacotes de no maximo 16 MiB - 1, como manda o
// protocolo, e mantem o numero de sequencia.
func (c *mysqlConn) write(payload []byte) error {
	for {
		chunk := payload
		if len(chunk) > myMaxPacketSize {
			chunk = chunk[:myMaxPacketSize]
		}
		header := []byte{byte(len(chunk)), byte(len(chunk) >> 8), byte(len(chunk) >> 16), c.sequence}
		c.sequence++
		if _, err := c.w.Write(header); err != nil {
			return err
		}
		if _, err := c.w.Write(chunk); err != nil {
			return err
		}
		payload = payload[len(chunk):]
		// Uma carga de exatamente 16 MiB - 1 exige um pacote vazio a seguir
		// para o servidor saber que acabou.
		if len(chunk) < myMaxPacketSize {
			break
		}
	}
	return c.w.Flush()
}

func (c *mysqlConn) read() ([]byte, error) {
	var payload []byte
	for {
		header := make([]byte, 4)
		if _, err := io.ReadFull(c.r, header); err != nil {
			return nil, fmt.Errorf("%w: reading from mysql: %v", ErrConnLost, err)
		}
		size := int(header[0]) | int(header[1])<<8 | int(header[2])<<16
		c.sequence = header[3] + 1
		chunk := make([]byte, size)
		if _, err := io.ReadFull(c.r, chunk); err != nil {
			return nil, fmt.Errorf("%w: reading from mysql: %v", ErrConnLost, err)
		}
		payload = append(payload, chunk...)
		if size < myMaxPacketSize {
			return payload, nil
		}
	}
}

func myAppendLenenc(dst []byte, n uint64) []byte {
	switch {
	case n < 251:
		return append(dst, byte(n))
	case n < 1<<16:
		return binary.LittleEndian.AppendUint16(append(dst, 0xfc), uint16(n))
	case n < 1<<24:
		return append(append(dst, 0xfd), byte(n), byte(n>>8), byte(n>>16))
	default:
		return binary.LittleEndian.AppendUint64(append(dst, 0xfe), n)
	}
}

func myReadLenenc(b []byte) (uint64, []byte) {
	if len(b) == 0 {
		return 0, b
	}
	switch b[0] {
	case 0xfc:
		if len(b) < 3 {
			return 0, nil
		}
		return uint64(binary.LittleEndian.Uint16(b[1:3])), b[3:]
	case 0xfd:
		if len(b) < 4 {
			return 0, nil
		}
		return uint64(b[1]) | uint64(b[2])<<8 | uint64(b[3])<<16, b[4:]
	case 0xfe:
		if len(b) < 9 {
			return 0, nil
		}
		return binary.LittleEndian.Uint64(b[1:9]), b[9:]
	default:
		return uint64(b[0]), b[1:]
	}
}

func myReadLenencString(b []byte) (string, []byte, bool) {
	if len(b) > 0 && b[0] == 0xfb { // NULL
		return "", b[1:], true
	}
	n, rest := myReadLenenc(b)
	if rest == nil || uint64(len(rest)) < n {
		return "", nil, false
	}
	return string(rest[:n]), rest[n:], true
}

// myError traduz o ERR_Packet: codigo(2), marcador '#' + SQLSTATE(5), mensagem.
func myError(p []byte) error {
	if len(p) < 3 {
		return fmt.Errorf("mysql returned an unreadable error")
	}
	code := binary.LittleEndian.Uint16(p[1:3])
	rest := p[3:]
	state := ""
	if len(rest) > 6 && rest[0] == '#' {
		state = string(rest[1:6])
		rest = rest[6:]
	}
	message := strings.TrimRight(string(rest), "\x00")
	if state != "" {
		return fmt.Errorf("mysql error %d (SQLSTATE %s): %s", code, state, message)
	}
	return fmt.Errorf("mysql error %d: %s", code, message)
}
