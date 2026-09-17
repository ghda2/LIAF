package codegen

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"liaf/pkg/ast"
	"liaf/pkg/lexer"
	"liaf/pkg/parser"
)

// Cobertura da v0.4: drivers de banco (issue #015) e WebSocket (issue #016).
// Os testes de geracao conferem a forma do codigo emitido; os de execucao
// compilam o exemplo de verdade e falam com ele pela rede.

// --- Geracao ----------------------------------------------------------------

func TestGenerateV04Transaction(t *testing.T) {
	code := generateExample(t, "db_postgres")

	for _, want := range []string{
		"rt.DBBegin(db)",
		"defer rt.DBRollback(db)",
		"rt.DBCommit(db)",
	} {
		if !strings.Contains(code, want) {
			t.Errorf("codigo gerado nao contem %q", want)
		}
	}
	// O defer e o que torna a transacao correta na presenca de try: um try
	// que falha faz return na funcao inteira, e so o defer alcanca essa saida.
	begin := strings.Index(code, "rt.DBBegin(db)")
	rollback := strings.Index(code, "defer rt.DBRollback(db)")
	commit := strings.Index(code, "rt.DBCommit(db)")
	if !(begin < rollback && rollback < commit) {
		t.Errorf("ordem errada: begin=%d rollback=%d commit=%d", begin, rollback, commit)
	}
	// Dentro do bloco o nome da conexao designa a transacao.
	if !strings.Contains(code, "db := _liaf_tx_") {
		t.Error("a conexao deve ser re-declarada como a transacao dentro do bloco")
	}
}

func TestGenerateV04QueryKeepsArgumentsOutOfSQL(t *testing.T) {
	code := generateExample(t, "db_postgres")

	// O SQL e um literal e os valores sao argumentos separados da chamada.
	want := `rt.DBQuery[User](db, "SELECT id, name, email FROM users WHERE id >= $1 ORDER BY id", minimum)`
	if !strings.Contains(code, want) {
		t.Errorf("db-query nao gerou a chamada esperada\nesperado: %s", want)
	}
	if !strings.Contains(code, `rt.DBExec(db, "INSERT INTO users (id, name, email) VALUES ($1, $2, $3)", 1, "alice", "alice@example.com")`) {
		t.Error("db-exec nao passou os parametros fora do SQL")
	}
	// Nenhuma concatenacao pode ter entrado no caminho da consulta.
	if strings.Contains(code, `_liaf_concat("SELECT`) || strings.Contains(code, `_liaf_concat("INSERT`) {
		t.Error("o SQL foi montado por concatenacao no codigo gerado")
	}
}

func TestGenerateV04WSRoute(t *testing.T) {
	code := generateExample(t, "chat_ws")

	for _, want := range []string{
		`web.RegisterWS("/ws/chat/{room}"`,
		"_liaf_ws_open_",
		"_liaf_ws_message_",
		"_liaf_ws_close_",
		"_liaf_conn.Next()",
	} {
		if !strings.Contains(code, want) {
			t.Errorf("codigo gerado nao contem %q", want)
		}
	}
	// {room} esta no indice 2 de /ws/chat/{room}.
	if !strings.Contains(code, "room = parts[2]") {
		t.Error("o parametro de path deve sair do indice do padrao")
	}
	// on-close precisa rodar depois do laco, e nao dentro dele, para valer
	// tanto no fechamento limpo quanto na queda da conexao.
	loop := strings.Index(code, "for {\n\t\t\t_liaf_text, _liaf_alive")
	closeCall := strings.Index(code, "_liaf_ws_close_")
	if loop < 0 || closeCall < 0 {
		t.Fatal("laco de leitura ou chamada de fechamento ausentes")
	}
	if strings.LastIndex(code, "_liaf_ws_close_") < loop {
		t.Error("on-close deve ser chamado depois do laco de leitura")
	}
}

func TestFormatV04Examples(t *testing.T) {
	// O liafc fmt precisa reimprimir as formas novas sem perder informacao e
	// sem oscilar entre duas saidas.
	for _, name := range []string{"db_postgres", "db_redis", "chat_ws"} {
		t.Run(name, func(t *testing.T) {
			root, err := filepath.Abs("../..")
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(root, "examples", name+".liaf"))
			if err != nil {
				t.Fatal(err)
			}
			p := parser.New(lexer.New(string(data)), name)
			mod := p.ParseModule()
			if len(p.Diagnostics) > 0 {
				t.Fatalf("parse: %+v", p.Diagnostics)
			}
			formatted := ast.Format(mod)

			p = parser.New(lexer.New(formatted), name)
			round := p.ParseModule()
			if len(p.Diagnostics) > 0 {
				t.Fatalf("a saida do formatter nao reanalisa:\n%s\n%+v", formatted, p.Diagnostics)
			}
			if again := ast.Format(round); again != formatted {
				t.Fatalf("formatter nao e idempotente\n--- primeira ---\n%s\n--- segunda ---\n%s", formatted, again)
			}
		})
	}
}

// --- Execucao ---------------------------------------------------------------

// buildExample compila um exemplo e devolve o caminho do binario.
func buildExample(t *testing.T, ctx context.Context, name, dir string) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(dir, "main.go")
	if err := os.WriteFile(source, []byte(generateExample(t, name)), 0600); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, name+".exe")
	build := exec.CommandContext(ctx, "go", "build", "-o", bin, source)
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build de %s: %v\n%s", name, err, out)
	}
	return bin
}

// --- Chat WebSocket ---------------------------------------------------------

// testWSClient e um cliente RFC 6455 minimo, para falar com o binario gerado.
type testWSClient struct {
	conn net.Conn
	r    *bufio.Reader
	t    *testing.T
}

func dialChat(t *testing.T, address, path string) *testWSClient {
	t.Helper()
	var conn net.Conn
	var err error
	for i := 0; i < 100; i++ {
		conn, err = net.DialTimeout("tcp", address, time.Second)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("nao conectou em %s: %v", address, err)
	}
	t.Cleanup(func() { conn.Close() })

	raw := make([]byte, 16)
	rand.Read(raw)
	key := base64.StdEncoding.EncodeToString(raw)
	fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\n"+
		"Connection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n",
		path, address, key)

	r := bufio.NewReader(conn)
	response, err := http.ReadResponse(r, nil)
	if err != nil {
		t.Fatalf("handshake: %v", err)
	}
	if response.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("handshake devolveu %d", response.StatusCode)
	}
	return &testWSClient{conn: conn, r: r, t: t}
}

func (c *testWSClient) send(text string) {
	c.t.Helper()
	payload := []byte(text)
	header := []byte{0x81}
	if len(payload) < 126 {
		header = append(header, 0x80|byte(len(payload)))
	} else {
		header = binary.BigEndian.AppendUint16(append(header, 0x80|126), uint16(len(payload)))
	}
	mask := make([]byte, 4)
	rand.Read(mask)
	header = append(header, mask...)
	masked := make([]byte, len(payload))
	for i := range payload {
		masked[i] = payload[i] ^ mask[i%4]
	}
	if _, err := c.conn.Write(append(header, masked...)); err != nil {
		c.t.Fatal(err)
	}
}

// nextEvent devolve o proximo ChatEvent, pulando frames de controle.
func (c *testWSClient) nextEvent() map[string]string {
	c.t.Helper()
	for {
		c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		header := make([]byte, 2)
		if _, err := io.ReadFull(c.r, header); err != nil {
			c.t.Fatalf("leitura: %v", err)
		}
		size := uint64(header[1] & 0x7f)
		switch size {
		case 126:
			extended := make([]byte, 2)
			io.ReadFull(c.r, extended)
			size = uint64(binary.BigEndian.Uint16(extended))
		case 127:
			extended := make([]byte, 8)
			io.ReadFull(c.r, extended)
			size = binary.BigEndian.Uint64(extended)
		}
		payload := make([]byte, size)
		if _, err := io.ReadFull(c.r, payload); err != nil {
			c.t.Fatalf("payload: %v", err)
		}
		if header[0]&0x0f != 0x1 { // ping, pong ou close
			continue
		}
		var event map[string]string
		if err := json.Unmarshal(payload, &event); err != nil {
			c.t.Fatalf("evento nao e JSON: %s", payload)
		}
		return event
	}
}

// TestExecuteV04ChatWS compila examples/chat_ws.liaf e conversa com ele por
// WebSocket de verdade: e a unica forma de provar que a declaracao
// (ws-route ...) vira um socket bidirecional funcionando.
func TestExecuteV04ChatWS(t *testing.T) {
	if testing.Short() {
		t.Skip("compila um binario; pulado em -short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "public"), 0755); err != nil {
		t.Fatal(err)
	}
	bin := buildExample(t, ctx, "chat_ws", dir)

	port := freePort(t)
	server := exec.CommandContext(ctx, bin, port)
	server.Dir = dir
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = server.Process.Kill()
		_, _ = server.Process.Wait()
	}()

	address := "127.0.0.1:" + port
	client := &http.Client{Timeout: 5 * time.Second}
	base := "http://" + address

	// Espera a porta responder antes de abrir o socket.
	ready := false
	for i := 0; i < 100; i++ {
		if res, err := client.Get(base + "/rooms/geral/size"); err == nil {
			res.Body.Close()
			ready = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !ready {
		t.Fatal("servidor nao subiu")
	}

	alice := dialChat(t, address, "/ws/chat/geral")
	// alice recebe o proprio evento de entrada: o join acontece em on-open,
	// antes do broadcast.
	if event := alice.nextEvent(); event["kind"] != "join" {
		t.Fatalf("primeiro evento de alice: %v", event)
	}

	bruno := dialChat(t, address, "/ws/chat/geral")
	if event := alice.nextEvent(); event["kind"] != "join" {
		t.Fatalf("alice devia ver a entrada de bruno: %v", event)
	}
	if event := bruno.nextEvent(); event["kind"] != "join" {
		t.Fatalf("bruno nao viu o proprio join: %v", event)
	}

	// A rota HTTP comum, na mesma porta, enxerga o mesmo topico.
	waitForClients(t, client, base+"/rooms/geral/size", 2)

	// Mensagem de um chega aos dois.
	alice.send("ola sala")
	for name, c := range map[string]*testWSClient{"alice": alice, "bruno": bruno} {
		event := c.nextEvent()
		if event["kind"] != "message" || event["text"] != "ola sala" {
			t.Fatalf("%s recebeu %v", name, event)
		}
		if event["room"] != "geral" {
			t.Errorf("%s: sala errada no evento: %v", name, event)
		}
	}

	// Um cliente noutra sala nao pode receber nada da sala geral.
	outra := dialChat(t, address, "/ws/chat/outra")
	if event := outra.nextEvent(); event["room"] != "outra" {
		t.Fatalf("cliente da outra sala recebeu %v", event)
	}
	waitForClients(t, client, base+"/rooms/outra/size", 1)

	// Ao cair a conexao, on-close roda e a sala e avisada.
	bruno.conn.Close()
	if event := alice.nextEvent(); event["kind"] != "leave" {
		t.Fatalf("alice devia ver a saida de bruno: %v", event)
	}
	waitForClients(t, client, base+"/rooms/geral/size", 1)
}

// waitForClients consulta a rota HTTP ate o topico ter o tamanho esperado.
func waitForClients(t *testing.T, client *http.Client, url string, want int) {
	t.Helper()
	var last int
	for i := 0; i < 100; i++ {
		res, err := client.Get(url)
		if err != nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		var payload struct {
			Room    string `json:"room"`
			Clients int    `json:"clients"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("%s devolveu %s", url, body)
		}
		last = payload.Clients
		if last == want {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("%s: esperado %d clientes, ultimo valor %d", url, want, last)
}

// --- Redis ------------------------------------------------------------------

// miniRedis atende o suficiente de RESP para o exemplo rodar: e o que permite
// exercitar db-connect, redis-set e redis-get no binario gerado sem exigir um
// Redis instalado.
type miniRedis struct {
	listener net.Listener
	mu       sync.Mutex
	values   map[string]string
}

func startMiniRedis(t *testing.T) *miniRedis {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	m := &miniRedis{listener: listener, values: map[string]string{}}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go m.serve(conn)
		}
	}()
	t.Cleanup(func() { listener.Close() })
	return m
}

func (m *miniRedis) serve(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		if !strings.HasPrefix(line, "*") {
			return
		}
		count, err := strconv.Atoi(strings.TrimRight(line[1:], "\r\n"))
		if err != nil {
			return
		}
		args := make([]string, 0, count)
		for i := 0; i < count; i++ {
			header, err := r.ReadString('\n')
			if err != nil {
				return
			}
			size, err := strconv.Atoi(strings.TrimRight(header[1:], "\r\n"))
			if err != nil {
				return
			}
			buf := make([]byte, size+2)
			if _, err := io.ReadFull(r, buf); err != nil {
				return
			}
			args = append(args, string(buf[:size]))
		}

		switch strings.ToUpper(args[0]) {
		case "SET":
			m.mu.Lock()
			m.values[args[1]] = args[2]
			m.mu.Unlock()
			fmt.Fprint(conn, "+OK\r\n")
		case "GET":
			m.mu.Lock()
			value, ok := m.values[args[1]]
			m.mu.Unlock()
			if !ok {
				fmt.Fprint(conn, "$-1\r\n")
				continue
			}
			fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(value), value)
		default:
			fmt.Fprint(conn, "+OK\r\n")
		}
	}
}

// TestExecuteV04RedisExample compila examples/db_redis.liaf e o roda contra o
// servidor acima, cobrindo o caminho inteiro: db-connect, redis-set,
// redis-get, json-decode e o erro de chave ausente.
func TestExecuteV04RedisExample(t *testing.T) {
	if testing.Short() {
		t.Skip("compila um binario; pulado em -short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	dir := t.TempDir()
	bin := buildExample(t, ctx, "db_redis", dir)

	server := startMiniRedis(t)
	run := exec.CommandContext(ctx, bin, "redis://"+server.listener.Addr().String()+"/0")
	run.Dir = dir
	output, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("execucao: %v\n%s", err, output)
	}
	got := strings.ReplaceAll(string(output), "\r\n", "\n")

	for _, want := range []string{
		"alice / admin",             // roundtrip completo pelo json-encode/decode
		"esperado: redis key not found: session:inexistente", // chave ausente vira err
		"pronto",
		"conexao encerrada",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("saida nao contem %q\n--- saida ---\n%s", want, got)
		}
	}
}

// TestExecuteV04PostgresExampleReportsConnectionFailure confirma que o
// exemplo de banco relacional compila e que a falha de conexao chega ao
// usuario como mensagem, e nao como panic.
func TestExecuteV04PostgresExampleFailsCleanly(t *testing.T) {
	if testing.Short() {
		t.Skip("compila um binario; pulado em -short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	dir := t.TempDir()
	bin := buildExample(t, ctx, "db_postgres", dir)

	// Porta fechada: o driver tem de falhar em db-connect, nao na consulta.
	run := exec.CommandContext(ctx, bin, "postgres://app:senha@127.0.0.1:"+freePort(t)+"/appdb?sslmode=disable")
	run.Dir = dir
	output, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("o programa devia terminar com codigo zero: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "conexao falhou") {
		t.Fatalf("falha de conexao nao chegou ao usuario:\n%s", output)
	}

	// Sem argumento nenhum, a mensagem de uso.
	run = exec.CommandContext(ctx, bin)
	run.Dir = dir
	output, _ = run.CombinedOutput()
	if !strings.Contains(string(output), "Uso: db_postgres") {
		t.Fatalf("mensagem de uso ausente:\n%s", output)
	}
}
