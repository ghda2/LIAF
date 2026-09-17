package web

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- Cliente de teste -------------------------------------------------------

// wsClient e um cliente WebSocket minimo. Escrever o cliente aqui, em vez de
// usar uma biblioteca, e o que faz o teste conferir os bytes que o servidor
// realmente produz — inclusive o Sec-WebSocket-Accept e a exigencia de mascara.
type wsClient struct {
	conn net.Conn
	r    *bufio.Reader
	t    *testing.T
}

func dialWS(t *testing.T, server *httptest.Server, path string) *wsClient {
	t.Helper()
	address := strings.TrimPrefix(server.URL, "http://")
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })

	raw := make([]byte, 16)
	rand.Read(raw)
	key := base64.StdEncoding.EncodeToString(raw)

	fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\n"+
		"Connection: keep-alive, Upgrade\r\nSec-WebSocket-Key: %s\r\n"+
		"Sec-WebSocket-Version: 13\r\n\r\n", path, address, key)

	r := bufio.NewReader(conn)
	response, err := http.ReadResponse(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("handshake devolveu %d", response.StatusCode)
	}
	// O Accept prova que o servidor entende o protocolo, e nao apenas que
	// devolveu 101 por acaso.
	sum := sha1.Sum([]byte(key + wsGUID))
	if want := base64.StdEncoding.EncodeToString(sum[:]); response.Header.Get("Sec-WebSocket-Accept") != want {
		t.Fatalf("Sec-WebSocket-Accept errado: %q", response.Header.Get("Sec-WebSocket-Accept"))
	}
	return &wsClient{conn: conn, r: r, t: t}
}

// write envia um frame mascarado, como a RFC exige do cliente.
func (c *wsClient) write(opcode byte, final bool, payload []byte) {
	c.t.Helper()
	header := []byte{opcode}
	if final {
		header[0] |= 0x80
	}
	switch {
	case len(payload) < 126:
		header = append(header, 0x80|byte(len(payload)))
	case len(payload) <= 0xffff:
		header = binary.BigEndian.AppendUint16(append(header, 0x80|126), uint16(len(payload)))
	default:
		header = binary.BigEndian.AppendUint64(append(header, 0x80|127), uint64(len(payload)))
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

func (c *wsClient) send(text string) { c.write(wsOpText, true, []byte(text)) }

// read devolve o proximo frame do servidor, ja sem mascara (o servidor nunca
// mascara).
func (c *wsClient) read() (byte, []byte) {
	c.t.Helper()
	c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	header := make([]byte, 2)
	if _, err := io.ReadFull(c.r, header); err != nil {
		c.t.Fatalf("leitura do frame: %v", err)
	}
	if header[1]&0x80 != 0 {
		c.t.Fatal("o servidor mascarou um frame, o que a RFC 6455 proibe")
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
		c.t.Fatalf("leitura do payload: %v", err)
	}
	return header[0] & 0x0f, payload
}

// expectText pula frames de controle ate achar a proxima mensagem de texto.
func (c *wsClient) expectText() string {
	c.t.Helper()
	for {
		opcode, payload := c.read()
		if opcode == wsOpText {
			return string(payload)
		}
		if opcode == wsOpClose {
			c.t.Fatalf("conexao fechada: %s", payload)
		}
	}
}

// --- Servidor de teste ------------------------------------------------------

// echoServer sobe um httptest que faz o upgrade e roda handler no laco.
func wsServer(t *testing.T, handler func(*WSConn)) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := Upgrade(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer conn.Close()
		handler(conn)
	}))
	t.Cleanup(server.Close)
	return server
}

func echoLoop(conn *WSConn) {
	for {
		message, alive := conn.Next()
		if !alive {
			return
		}
		if err := conn.Send("echo:" + message); err != nil {
			return
		}
	}
}

// --- Testes -----------------------------------------------------------------

func TestWebSocketEcho(t *testing.T) {
	server := wsServer(t, echoLoop)
	client := dialWS(t, server, "/ws")

	for _, message := range []string{"ola", "", "acentuação e emoji 🎉", strings.Repeat("x", 300)} {
		client.send(message)
		if got := client.expectText(); got != "echo:"+message {
			t.Fatalf("eco divergente para %q: %q", message, got)
		}
	}
}

func TestWebSocketLargePayloadUsesExtendedLength(t *testing.T) {
	// Acima de 65535 bytes o tamanho vai em 8 bytes; e o caminho de codigo
	// que quase nunca roda e onde um off-by-one passa despercebido.
	big := strings.Repeat("a", 70000)
	server := wsServer(t, echoLoop)
	client := dialWS(t, server, "/ws")
	client.send(big)
	if got := client.expectText(); got != "echo:"+big {
		t.Fatalf("payload grande voltou com %d bytes", len(got))
	}
}

func TestWebSocketReassemblesFragments(t *testing.T) {
	server := wsServer(t, echoLoop)
	client := dialWS(t, server, "/ws")

	client.write(wsOpText, false, []byte("uma "))
	client.write(wsOpContinuation, false, []byte("mensagem "))
	client.write(wsOpContinuation, true, []byte("partida"))

	if got := client.expectText(); got != "echo:uma mensagem partida" {
		t.Fatalf("fragmentos remontados como %q", got)
	}
}

func TestWebSocketAnswersPingWithPong(t *testing.T) {
	server := wsServer(t, echoLoop)
	client := dialWS(t, server, "/ws")

	client.write(wsOpPing, true, []byte("bate"))
	opcode, payload := client.read()
	if opcode != wsOpPong {
		t.Fatalf("resposta ao ping foi opcode %d", opcode)
	}
	// O pong tem de devolver o payload do ping, senao o cliente nao consegue
	// casar a resposta com o que enviou.
	if string(payload) != "bate" {
		t.Fatalf("pong devolveu %q", payload)
	}
}

func TestWebSocketCloseHandshake(t *testing.T) {
	closed := make(chan struct{})
	server := wsServer(t, func(conn *WSConn) {
		echoLoop(conn)
		close(closed)
	})
	client := dialWS(t, server, "/ws")

	client.write(wsOpClose, true, binary.BigEndian.AppendUint16(nil, WSCloseNormal))
	if opcode, _ := client.read(); opcode != wsOpClose {
		t.Fatalf("servidor nao devolveu o frame de fechamento: opcode %d", opcode)
	}
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("Next nao retornou depois do fechamento")
	}
}

func TestWebSocketRejectsUnmaskedClientFrame(t *testing.T) {
	server := wsServer(t, echoLoop)
	client := dialWS(t, server, "/ws")

	// Frame sem mascara: a RFC obriga o cliente a mascarar, e aceitar sem
	// mascara reabriria o ataque de envenenamento de cache que a mascara evita.
	client.conn.Write([]byte{0x81, 0x03, 'a', 'b', 'c'})
	opcode, payload := client.read()
	if opcode != wsOpClose {
		t.Fatalf("frame sem mascara foi aceito (opcode %d)", opcode)
	}
	if code := binary.BigEndian.Uint16(payload[:2]); code != WSCloseProtocolError {
		t.Fatalf("codigo de fechamento %d, esperado %d", code, WSCloseProtocolError)
	}
}

func TestWebSocketRejectsOversizedFrame(t *testing.T) {
	server := wsServer(t, echoLoop)
	client := dialWS(t, server, "/ws")

	// Anuncia 64 MiB sem enviar nada: o servidor nao pode alocar o buffer
	// antes de checar o limite.
	header := binary.BigEndian.AppendUint64([]byte{0x81, 0x80 | 127}, 64<<20)
	client.conn.Write(append(header, 0, 0, 0, 0))
	opcode, payload := client.read()
	if opcode != wsOpClose {
		t.Fatalf("frame gigante foi aceito (opcode %d)", opcode)
	}
	if code := binary.BigEndian.Uint16(payload[:2]); code != WSCloseMessageTooBig {
		t.Fatalf("codigo de fechamento %d, esperado %d", code, WSCloseMessageTooBig)
	}
}

func TestUpgradeRejectsBadHandshakes(t *testing.T) {
	server := wsServer(t, echoLoop)

	for _, tc := range []struct {
		name    string
		method  string
		headers map[string]string
	}{
		{"sem cabecalho de upgrade", "GET", nil},
		{"versao errada", "GET", map[string]string{
			"Upgrade": "websocket", "Connection": "Upgrade",
			"Sec-WebSocket-Key": "dGhlIHNhbXBsZSBub25jZQ==", "Sec-WebSocket-Version": "8"}},
		{"chave invalida", "GET", map[string]string{
			"Upgrade": "websocket", "Connection": "Upgrade",
			"Sec-WebSocket-Key": "curta", "Sec-WebSocket-Version": "13"}},
		{"metodo errado", "POST", map[string]string{
			"Upgrade": "websocket", "Connection": "Upgrade",
			"Sec-WebSocket-Key": "dGhlIHNhbXBsZSBub25jZQ==", "Sec-WebSocket-Version": "13"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequest(tc.method, server.URL+"/ws", nil)
			if err != nil {
				t.Fatal(err)
			}
			for k, v := range tc.headers {
				request.Header.Set(k, v)
			}
			response, err := server.Client().Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode == http.StatusSwitchingProtocols {
				t.Fatal("handshake invalido foi aceito")
			}
		})
	}
}

// --- Pub/sub ----------------------------------------------------------------

func TestBroadcastReachesTopicMembersOnly(t *testing.T) {
	var ready sync.WaitGroup
	ready.Add(3)
	server := wsServer(t, func(conn *WSConn) {
		topic := strings.TrimPrefix(conn.Path, "/ws/")
		WSJoin(conn, topic)
		ready.Done()
		for {
			if _, alive := conn.Next(); !alive {
				return
			}
		}
	})

	salaA1 := dialWS(t, server, "/ws/sala-a")
	salaA2 := dialWS(t, server, "/ws/sala-a")
	salaB := dialWS(t, server, "/ws/sala-b")
	ready.Wait()

	if got := WSBroadcast("sala-a", "so para a sala A"); got != 2 {
		t.Fatalf("broadcast alcancou %d clientes, esperado 2", got)
	}
	if got := salaA1.expectText(); got != "so para a sala A" {
		t.Errorf("cliente 1: %q", got)
	}
	if got := salaA2.expectText(); got != "so para a sala A" {
		t.Errorf("cliente 2: %q", got)
	}

	// O cliente da outra sala nao pode ter recebido nada.
	if got := WSBroadcast("sala-b", "so para a sala B"); got != 1 {
		t.Fatalf("sala-b tem %d clientes, esperado 1", got)
	}
	if got := salaB.expectText(); got != "so para a sala B" {
		t.Errorf("vazamento entre salas: %q", got)
	}

	if WSTopicSize("sala-a") != 2 || WSTopicSize("sala-b") != 1 {
		t.Errorf("tamanhos: a=%d b=%d", WSTopicSize("sala-a"), WSTopicSize("sala-b"))
	}
	if WSTopicSize("sala-inexistente") != 0 {
		t.Error("topico inexistente devia ter tamanho zero")
	}
}

func TestClosingConnectionLeavesEveryTopic(t *testing.T) {
	gone := make(chan struct{})
	server := wsServer(t, func(conn *WSConn) {
		WSJoin(conn, "efemera")
		WSJoin(conn, "efemera-2")
		for {
			if _, alive := conn.Next(); !alive {
				close(gone)
				return
			}
		}
	})
	client := dialWS(t, server, "/ws")

	deadline := time.Now().Add(5 * time.Second)
	for WSTopicSize("efemera") == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if WSTopicSize("efemera") != 1 {
		t.Fatal("o cliente nao entrou no topico")
	}

	client.conn.Close()
	<-gone

	// Sem a saida automatica, o broadcast seguiria escrevendo num socket morto
	// e o topico cresceria para sempre.
	deadline = time.Now().Add(5 * time.Second)
	for (WSTopicSize("efemera") != 0 || WSTopicSize("efemera-2") != 0) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if WSTopicSize("efemera") != 0 || WSTopicSize("efemera-2") != 0 {
		t.Fatalf("o cliente ficou pendurado: efemera=%d efemera-2=%d",
			WSTopicSize("efemera"), WSTopicSize("efemera-2"))
	}
}

func TestSendJSONSerializes(t *testing.T) {
	type event struct {
		Kind string `json:"kind"`
		Text string `json:"text"`
	}
	server := wsServer(t, func(conn *WSConn) {
		conn.SendJSON(event{Kind: "join", Text: "alguem entrou"})
		conn.Next()
	})
	client := dialWS(t, server, "/ws")
	if got := client.expectText(); got != `{"kind":"join","text":"alguem entrou"}` {
		t.Fatalf("json enviado: %s", got)
	}
}

func TestSendAfterCloseFails(t *testing.T) {
	done := make(chan *WSConn, 1)
	server := wsServer(t, func(conn *WSConn) {
		done <- conn
		conn.Next()
	})
	dialWS(t, server, "/ws")
	conn := <-done
	conn.Close()
	if err := conn.Send("tarde demais"); err == nil {
		t.Fatal("envio numa conexao fechada foi aceito")
	}
}
