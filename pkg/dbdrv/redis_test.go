package dbdrv

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakeRedis fala RESP de verdade contra o driver: e a unica forma de provar
// que o enquadramento do cliente esta certo sem um servidor instalado.
type fakeRedis struct {
	listener net.Listener
	mu       sync.Mutex
	values   map[string]string
	commands [][]string
	password string
}

func startFakeRedis(t *testing.T, password string) *fakeRedis {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeRedis{listener: listener, values: map[string]string{}, password: password}
	go f.accept()
	t.Cleanup(func() { listener.Close() })
	return f
}

func (f *fakeRedis) addr() string { return f.listener.Addr().String() }

func (f *fakeRedis) seen() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]string(nil), f.commands...)
}

func (f *fakeRedis) accept() {
	for {
		conn, err := f.listener.Accept()
		if err != nil {
			return
		}
		go f.serve(conn)
	}
}

func (f *fakeRedis) serve(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	authenticated := f.password == ""
	for {
		args, err := readRESPCommand(r)
		if err != nil {
			return
		}
		f.mu.Lock()
		f.commands = append(f.commands, args)
		f.mu.Unlock()

		switch strings.ToUpper(args[0]) {
		case "AUTH":
			if args[len(args)-1] == f.password {
				authenticated = true
				fmt.Fprint(conn, "+OK\r\n")
			} else {
				fmt.Fprint(conn, "-WRONGPASS invalid password\r\n")
			}
		case "SELECT", "PING":
			fmt.Fprint(conn, "+OK\r\n")
		case "GET":
			if !authenticated {
				fmt.Fprint(conn, "-NOAUTH Authentication required\r\n")
				continue
			}
			f.mu.Lock()
			v, ok := f.values[args[1]]
			f.mu.Unlock()
			if !ok {
				fmt.Fprint(conn, "$-1\r\n") // nil: chave ausente
				continue
			}
			fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(v), v)
		case "SET":
			f.mu.Lock()
			f.values[args[1]] = args[2]
			f.mu.Unlock()
			fmt.Fprint(conn, "+OK\r\n")
		default:
			fmt.Fprintf(conn, "-ERR unknown command '%s'\r\n", args[0])
		}
	}
}

func readRESPCommand(r *bufio.Reader) ([]string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(line, "*") {
		return nil, fmt.Errorf("expected array, got %q", line)
	}
	count, err := strconv.Atoi(strings.TrimRight(line[1:], "\r\n"))
	if err != nil {
		return nil, err
	}
	args := make([]string, 0, count)
	for i := 0; i < count; i++ {
		header, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		size, err := strconv.Atoi(strings.TrimRight(header[1:], "\r\n"))
		if err != nil {
			return nil, err
		}
		buf := make([]byte, size+2)
		if _, err := ioReadFull(r, buf); err != nil {
			return nil, err
		}
		args = append(args, string(buf[:size]))
	}
	return args, nil
}

func TestRedisGetSetRoundTrip(t *testing.T) {
	server := startFakeRedis(t, "")
	conn, err := Open(DriverRedis, "redis://"+server.addr()+"/0")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	kv := conn.(KV)
	if err := kv.Set("session:42", `{"user":"alice"}`, 60); err != nil {
		t.Fatal(err)
	}
	value, found, err := kv.Get("session:42")
	if err != nil || !found {
		t.Fatalf("Get: value=%q found=%v err=%v", value, found, err)
	}
	if value != `{"user":"alice"}` {
		t.Fatalf("valor devolvido: %q", value)
	}

	// Chave ausente e "nao encontrado", nao erro de protocolo.
	if _, found, err := kv.Get("session:inexistente"); err != nil || found {
		t.Fatalf("chave ausente: found=%v err=%v", found, err)
	}
}

func TestRedisSetSendsTTL(t *testing.T) {
	server := startFakeRedis(t, "")
	conn, err := Open(DriverRedis, "redis://"+server.addr())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	kv := conn.(KV)

	if err := kv.Set("a", "1", 90); err != nil {
		t.Fatal(err)
	}
	if err := kv.Set("b", "2", 0); err != nil {
		t.Fatal(err)
	}

	var withTTL, withoutTTL []string
	for _, cmd := range server.seen() {
		if len(cmd) >= 3 && cmd[0] == "SET" && cmd[1] == "a" {
			withTTL = cmd
		}
		if len(cmd) >= 3 && cmd[0] == "SET" && cmd[1] == "b" {
			withoutTTL = cmd
		}
	}
	if len(withTTL) != 5 || withTTL[3] != "EX" || withTTL[4] != "90" {
		t.Fatalf("TTL nao chegou como EX 90: %v", withTTL)
	}
	// TTL zero significa "sem expiracao": mandar EX 0 seria erro no Redis.
	if len(withoutTTL) != 3 {
		t.Fatalf("TTL zero nao devia virar EX: %v", withoutTTL)
	}
}

func TestRedisAuthAndSelect(t *testing.T) {
	server := startFakeRedis(t, "s3cr3t")
	conn, err := Open(DriverRedis, "redis://:s3cr3t@"+server.addr()+"/3")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	var sawAuth, sawSelect bool
	for _, cmd := range server.seen() {
		switch cmd[0] {
		case "AUTH":
			sawAuth = cmd[len(cmd)-1] == "s3cr3t"
		case "SELECT":
			sawSelect = cmd[1] == "3"
		}
	}
	if !sawAuth {
		t.Error("AUTH nao foi enviado com a senha do DSN")
	}
	if !sawSelect {
		t.Error("SELECT nao foi enviado com o banco do DSN")
	}
}

func TestRedisRejectsWrongPassword(t *testing.T) {
	server := startFakeRedis(t, "s3cr3t")
	if _, err := Open(DriverRedis, "redis://:errada@"+server.addr()); err == nil {
		t.Fatal("senha errada foi aceita")
	}
}

func TestRedisRejectsRelationalOperations(t *testing.T) {
	server := startFakeRedis(t, "")
	conn, err := Open(DriverRedis, "redis://"+server.addr())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Falhar explicitamente e melhor que devolver um resultado vazio, que
	// pareceria uma tabela sem linhas.
	if _, err := conn.Query("SELECT 1", nil); err == nil {
		t.Error("db-query devia falhar no driver redis")
	}
	if _, err := conn.Exec("DELETE FROM x", nil); err == nil {
		t.Error("db-exec devia falhar no driver redis")
	}
	if err := conn.Begin(); err == nil {
		t.Error("db-transaction devia falhar no driver redis")
	}
}
