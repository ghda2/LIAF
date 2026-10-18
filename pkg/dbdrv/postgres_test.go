package dbdrv

import (
	"encoding/binary"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
)

// fakePostgres implementa o lado servidor do protocolo v3 no minimo necessario
// para exercitar o cliente: SSLRequest, autenticacao MD5 e o ciclo
// Parse/Bind/Describe/Execute/Sync. Sem isto o enquadramento so seria testado
// contra um PostgreSQL de verdade, que a suite nao pode exigir.
type fakePostgres struct {
	listener net.Listener
	salt     [4]byte

	mu       sync.Mutex
	lastSQL  string
	lastArgs []string
	startup  map[string]string

	rows    [][]string
	columns []struct {
		name string
		oid  uint32
	}
	tag     string
	failure string // quando != "", responde ErrorResponse no lugar das linhas
}

func startFakePostgres(t *testing.T) *fakePostgres {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakePostgres{listener: listener, salt: [4]byte{1, 2, 3, 4}, startup: map[string]string{}, tag: "SELECT 0"}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go f.serve(conn)
		}
	}()
	t.Cleanup(func() { listener.Close() })
	return f
}

func (f *fakePostgres) dsn() string {
	return "postgres://app:s3cr3t@" + f.listener.Addr().String() + "/appdb?sslmode=disable"
}

func (f *fakePostgres) column(name string, oid uint32) {
	f.columns = append(f.columns, struct {
		name string
		oid  uint32
	}{name, oid})
}

func (f *fakePostgres) serve(conn net.Conn) {
	defer conn.Close()

	// Primeira mensagem sem byte de tipo: SSLRequest ou StartupMessage.
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return
	}
	size := int(binary.BigEndian.Uint32(header))
	body := make([]byte, size-4)
	if _, err := io.ReadFull(conn, body); err != nil {
		return
	}
	if size == 8 && binary.BigEndian.Uint32(body) == pgSSLRequestCode {
		conn.Write([]byte{'N'}) // recusa TLS; sslmode=disable nunca chega aqui
		if _, err := io.ReadFull(conn, header); err != nil {
			return
		}
		size = int(binary.BigEndian.Uint32(header))
		body = make([]byte, size-4)
		if _, err := io.ReadFull(conn, body); err != nil {
			return
		}
	}

	fields := strings.Split(string(body[4:]), "\x00")
	f.mu.Lock()
	for i := 0; i+1 < len(fields); i += 2 {
		if fields[i] != "" {
			f.startup[fields[i]] = fields[i+1]
		}
	}
	user := f.startup["user"]
	f.mu.Unlock()

	// AuthenticationMD5Password
	auth := binary.BigEndian.AppendUint32(nil, pgAuthMD5Password)
	writePGMessage(conn, 'R', append(auth, f.salt[:]...))

	kind, payload, err := readPGMessage(conn)
	if err != nil || kind != 'p' {
		return
	}
	expected := pgMD5Password(user, "s3cr3t", f.salt[:])
	if strings.TrimRight(string(payload), "\x00") != expected {
		writePGMessage(conn, 'E', pgErrorPayload("28P01", "password authentication failed"))
		return
	}
	writePGMessage(conn, 'R', binary.BigEndian.AppendUint32(nil, pgAuthOK))
	writePGMessage(conn, 'Z', []byte{'I'})

	f.loop(conn)
}

func (f *fakePostgres) loop(conn net.Conn) {
	for {
		kind, payload, err := readPGMessage(conn)
		if err != nil {
			return
		}
		switch kind {
		case 'Q':
			f.mu.Lock()
			f.lastSQL = strings.TrimRight(string(payload), "\x00")
			f.mu.Unlock()
			writePGMessage(conn, 'C', append([]byte("BEGIN"), 0))
			writePGMessage(conn, 'Z', []byte{'I'})
		case 'P':
			parts := strings.SplitN(string(payload), "\x00", 3)
			f.mu.Lock()
			f.lastSQL = parts[1]
			f.mu.Unlock()
		case 'B':
			f.mu.Lock()
			f.lastArgs = decodeBindArgs(payload)
			f.mu.Unlock()
		case 'S':
			f.respond(conn)
		}
	}
}

// decodeBindArgs extrai os parametros do Bind. E o ponto central do teste de
// injecao: o valor tem de chegar aqui, fora do texto do comando.
func decodeBindArgs(payload []byte) []string {
	rest := payload
	for i := 0; i < 2; i++ { // pula portal e statement, ambos strings com NUL
		end := indexZero(rest)
		if end < 0 {
			return nil
		}
		rest = rest[end+1:]
	}
	if len(rest) < 4 {
		return nil
	}
	formats := int(binary.BigEndian.Uint16(rest[:2]))
	rest = rest[2+formats*2:]
	count := int(binary.BigEndian.Uint16(rest[:2]))
	rest = rest[2:]
	args := make([]string, 0, count)
	for i := 0; i < count; i++ {
		size := int32(binary.BigEndian.Uint32(rest[:4]))
		rest = rest[4:]
		if size < 0 {
			args = append(args, "<null>")
			continue
		}
		args = append(args, string(rest[:size]))
		rest = rest[size:]
	}
	return args
}

func (f *fakePostgres) respond(conn net.Conn) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.failure != "" {
		writePGMessage(conn, 'E', pgErrorPayload("42P01", f.failure))
		writePGMessage(conn, 'Z', []byte{'I'})
		return
	}

	writePGMessage(conn, '1', nil)
	writePGMessage(conn, '2', nil)
	if len(f.columns) > 0 {
		description := binary.BigEndian.AppendUint16(nil, uint16(len(f.columns)))
		for i, col := range f.columns {
			description = pgAppendString(description, col.name)
			description = binary.BigEndian.AppendUint32(description, 0)        // tabela
			description = binary.BigEndian.AppendUint16(description, uint16(i)) // coluna
			description = binary.BigEndian.AppendUint32(description, col.oid)
			description = binary.BigEndian.AppendUint16(description, 0xffff) // tamanho
			description = binary.BigEndian.AppendUint32(description, 0xffffffff)
			description = binary.BigEndian.AppendUint16(description, 0) // formato texto
		}
		writePGMessage(conn, 'T', description)
	}
	for _, row := range f.rows {
		data := binary.BigEndian.AppendUint16(nil, uint16(len(row)))
		for _, value := range row {
			if value == "<null>" {
				data = binary.BigEndian.AppendUint32(data, ^uint32(0))
				continue
			}
			data = binary.BigEndian.AppendUint32(data, uint32(len(value)))
			data = append(data, value...)
		}
		writePGMessage(conn, 'D', data)
	}
	writePGMessage(conn, 'C', append([]byte(f.tag), 0))
	writePGMessage(conn, 'Z', []byte{'I'})
}

func pgErrorPayload(code, message string) []byte {
	out := []byte{'S'}
	out = append(out, "ERROR\x00"...)
	out = append(out, 'C')
	out = append(out, append([]byte(code), 0)...)
	out = append(out, 'M')
	out = append(out, append([]byte(message), 0)...)
	return append(out, 0)
}

func writePGMessage(w io.Writer, kind byte, body []byte) {
	frame := append([]byte{kind}, binary.BigEndian.AppendUint32(nil, uint32(len(body)+4))...)
	w.Write(append(frame, body...))
}

func readPGMessage(r io.Reader) (byte, []byte, error) {
	header := make([]byte, 5)
	if _, err := io.ReadFull(r, header); err != nil {
		return 0, nil, err
	}
	body := make([]byte, int(binary.BigEndian.Uint32(header[1:5]))-4)
	if _, err := io.ReadFull(r, body); err != nil {
		return 0, nil, err
	}
	return header[0], body, nil
}

// --- Testes -----------------------------------------------------------------

func TestPostgresHandshakeAndTypedQuery(t *testing.T) {
	server := startFakePostgres(t)
	const oidText = 25
	server.column("id", oidInt4)
	server.column("name", oidText)
	server.column("score", oidFloat8)
	server.column("active", oidBool)
	server.column("note", oidText)
	server.rows = [][]string{{"42", "alice", "9.5", "t", "<null>"}}
	server.tag = "SELECT 1"

	conn, err := Open(DriverPostgres, server.dsn())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	rows, err := conn.Query("SELECT id, name, score, active, note FROM users WHERE id = $1", []Value{int64(42)})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows.Values) != 1 {
		t.Fatalf("esperado 1 linha, recebido %d", len(rows.Values))
	}
	got := rows.Values[0]
	// A conversao pelo OID e o que faz um campo int da struct LIAF receber 42
	// e nao "42"; sem ela o json.Unmarshal da linha falharia.
	if v, ok := got[0].(int64); !ok || v != 42 {
		t.Errorf("coluna int4: %#v", got[0])
	}
	if v, ok := got[1].(string); !ok || v != "alice" {
		t.Errorf("coluna text: %#v", got[1])
	}
	if v, ok := got[2].(float64); !ok || v != 9.5 {
		t.Errorf("coluna float8: %#v", got[2])
	}
	if v, ok := got[3].(bool); !ok || !v {
		t.Errorf("coluna bool: %#v", got[3])
	}
	if got[4] != nil {
		t.Errorf("coluna NULL: %#v", got[4])
	}
	if strings.Join(rows.Columns, ",") != "id,name,score,active,note" {
		t.Errorf("nomes de coluna: %v", rows.Columns)
	}
}

// TestPostgresSendsArgumentsOutOfBand e a prova do contrato de seguranca da
// issue #015: o valor nunca entra no texto do comando.
func TestPostgresSendsArgumentsOutOfBand(t *testing.T) {
	server := startFakePostgres(t)
	server.tag = "UPDATE 1"

	conn, err := Open(DriverPostgres, server.dsn())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	hostile := "x'; DROP TABLE users; --"
	affected, err := conn.Exec("UPDATE users SET name = $1 WHERE id = $2", []Value{hostile, int64(7)})
	if err != nil {
		t.Fatal(err)
	}
	if affected != 1 {
		t.Errorf("linhas afetadas: %d", affected)
	}

	server.mu.Lock()
	sql, args := server.lastSQL, server.lastArgs
	server.mu.Unlock()

	if sql != "UPDATE users SET name = $1 WHERE id = $2" {
		t.Fatalf("o SQL chegou alterado: %q", sql)
	}
	if strings.Contains(sql, "DROP TABLE") {
		t.Fatal("o valor hostil vazou para dentro do comando")
	}
	if len(args) != 2 || args[0] != hostile || args[1] != "7" {
		t.Fatalf("parametros recebidos: %q", args)
	}
}

func TestPostgresReportsServerError(t *testing.T) {
	server := startFakePostgres(t)
	server.failure = `relation "users" does not exist`

	conn, err := Open(DriverPostgres, server.dsn())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	_, err = conn.Query("SELECT 1 FROM users", nil)
	if err == nil {
		t.Fatal("erro do servidor foi engolido")
	}
	// A mensagem precisa carregar SQLSTATE e texto: e o que permite a um
	// modelo corrigir o programa sem adivinhar.
	if !strings.Contains(err.Error(), "42P01") || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("mensagem pobre: %v", err)
	}
}

func TestPostgresRejectsWrongPassword(t *testing.T) {
	server := startFakePostgres(t)
	dsn := strings.Replace(server.dsn(), "s3cr3t", "errada", 1)
	if _, err := Open(DriverPostgres, dsn); err == nil {
		t.Fatal("senha errada foi aceita")
	}
}

func TestPostgresSendsStartupParameters(t *testing.T) {
	server := startFakePostgres(t)
	conn, err := Open(DriverPostgres, server.dsn())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	server.mu.Lock()
	defer server.mu.Unlock()
	if server.startup["user"] != "app" {
		t.Errorf("user: %q", server.startup["user"])
	}
	if server.startup["database"] != "appdb" {
		t.Errorf("database: %q", server.startup["database"])
	}
}

func TestPostgresTransactionCommands(t *testing.T) {
	server := startFakePostgres(t)
	conn, err := Open(DriverPostgres, server.dsn())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	for _, tc := range []struct {
		run  func() error
		want string
	}{
		{conn.Begin, "BEGIN"},
		{conn.Commit, "COMMIT"},
		{conn.Rollback, "ROLLBACK"},
	} {
		if err := tc.run(); err != nil {
			t.Fatalf("%s: %v", tc.want, err)
		}
		server.mu.Lock()
		got := server.lastSQL
		server.mu.Unlock()
		if got != tc.want {
			t.Errorf("esperado %q, enviado %q", tc.want, got)
		}
	}
}

func TestPostgresRowsAffectedFromTag(t *testing.T) {
	for _, tc := range []struct {
		tag  string
		want int64
	}{
		{"INSERT 0 1", 1},
		{"UPDATE 3", 3},
		{"DELETE 12", 12},
		{"SELECT 7", 7},
		{"BEGIN", 0},
	} {
		if got := pgRowsAffected(append([]byte(tc.tag), 0)); got != tc.want {
			t.Errorf("%q: esperado %d, recebido %d", tc.tag, tc.want, got)
		}
	}
}
