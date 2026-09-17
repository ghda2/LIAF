package web

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// WebSocket do lado servidor, conforme a RFC 6455, mais o pub/sub em memoria
// que as rotas (ws-route ...) usam para broadcast.
//
// Nao ha dependencia externa aqui pelo mesmo motivo de pkg/dbdrv: a LIAF
// entrega um binario unico e o enquadramento da RFC 6455 cabe num arquivo.
// O que nao esta implementado: extensoes negociadas (permessage-deflate),
// fragmentacao na escrita e o papel de cliente.

const (
	wsOpContinuation = 0x0
	wsOpText         = 0x1
	wsOpBinary       = 0x2
	wsOpClose        = 0x8
	wsOpPing         = 0x9
	wsOpPong         = 0xa

	// GUID fixo da RFC 6455, secao 1.3: entra no calculo do
	// Sec-WebSocket-Accept para provar que o servidor entende o protocolo,
	// e nao e um servidor HTTP qualquer devolvendo 101 por acaso.
	wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

	// Mesmo teto do corpo HTTP em router.go: uma conexao aberta nao deve ser
	// um caminho mais generoso para exaurir a memoria do processo.
	wsMaxMessageSize = 1 << 20

	wsPingInterval = 30 * time.Second
	wsIdleTimeout  = 90 * time.Second
	wsWriteTimeout = 10 * time.Second
)

// Codigos de fechamento da RFC 6455, secao 7.4.1.
const (
	WSCloseNormal          = 1000
	WSCloseGoingAway       = 1001
	WSCloseProtocolError   = 1002
	WSCloseUnsupportedData = 1003
	WSCloseInvalidPayload  = 1007
	WSClosePolicyViolation = 1008
	WSCloseMessageTooBig   = 1009
	WSCloseInternalError   = 1011
)

// WSConn e uma conexao WebSocket aberta. Uma unica goroutine chama Next; as
// escritas podem vir de qualquer goroutine (um broadcast parte da conexao de
// outro cliente) e por isso passam por writeMu.
type WSConn struct {
	conn net.Conn
	r    *bufio.Reader

	writeMu sync.Mutex
	closeMu sync.Mutex
	closed  bool

	// Remote e Path ficam disponiveis para log; o handler gerado tambem
	// recebe o Request equivalente.
	Remote string
	Path   string

	topics map[string]bool
	hub    *wsHub
	done   chan struct{}
}

// --- Upgrade ----------------------------------------------------------------

// Upgrade troca a requisicao HTTP por uma conexao WebSocket. Devolve erro sem
// escrever nada quando a requisicao nao e um handshake valido, para o chamador
// poder responder 400.
func Upgrade(w http.ResponseWriter, r *http.Request) (*WSConn, error) {
	if r.Method != http.MethodGet {
		return nil, fmt.Errorf("websocket handshake requires GET")
	}
	if !headerContainsToken(r.Header, "Connection", "upgrade") {
		return nil, fmt.Errorf("missing Connection: Upgrade")
	}
	if !strings.EqualFold(strings.TrimSpace(r.Header.Get("Upgrade")), "websocket") {
		return nil, fmt.Errorf("missing Upgrade: websocket")
	}
	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		return nil, fmt.Errorf("unsupported Sec-WebSocket-Version (only 13 is implemented)")
	}
	key := strings.TrimSpace(r.Header.Get("Sec-WebSocket-Key"))
	// A chave e 16 bytes aleatorios em base64, o que da sempre 24 caracteres.
	if len(key) != 24 {
		return nil, fmt.Errorf("invalid Sec-WebSocket-Key")
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, fmt.Errorf("this server cannot hand over the connection (HTTP/2 does not support this upgrade)")
	}
	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, fmt.Errorf("could not take over the connection: %w", err)
	}

	// Depois do Hijack o servidor HTTP nao escreve mais nada, entao a
	// resposta 101 sai byte a byte daqui.
	accept := wsAcceptKey(key)
	response := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n\r\n"
	_ = conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
	if _, err := rw.WriteString(response); err != nil {
		conn.Close()
		return nil, err
	}
	if err := rw.Flush(); err != nil {
		conn.Close()
		return nil, err
	}
	_ = conn.SetWriteDeadline(time.Time{})

	c := &WSConn{
		conn:   conn,
		r:      rw.Reader,
		Remote: r.RemoteAddr,
		Path:   r.URL.Path,
		topics: map[string]bool{},
		hub:    defaultHub,
		done:   make(chan struct{}),
	}
	go c.keepalive()
	return c, nil
}

func wsAcceptKey(key string) string {
	sum := sha1.Sum([]byte(key + wsGUID))
	return base64.StdEncoding.EncodeToString(sum[:])
}

// headerContainsToken procura um token numa lista separada por virgulas.
// Connection pode chegar como "keep-alive, Upgrade", entao comparar o campo
// inteiro perderia o caso.
func headerContainsToken(h http.Header, name, token string) bool {
	for _, value := range h.Values(name) {
		for _, part := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}

// --- Leitura ----------------------------------------------------------------

// Next devolve a proxima mensagem de texto do cliente. O segundo retorno e
// false quando a conexao terminou, por fechamento limpo ou por falha: para o
// codigo LIAF as duas coisas significam o mesmo, sair do laco e rodar
// (on-close ...).
//
// Ping, pong e frames binarios sao tratados aqui e nao chegam ao handler.
func (c *WSConn) Next() (string, bool) {
	var message []byte
	var fragmentOp byte

	for {
		_ = c.conn.SetReadDeadline(time.Now().Add(wsIdleTimeout))
		final, opcode, payload, err := c.readFrame()
		if err != nil {
			if errors.Is(err, errWSProtocol) {
				c.closeWith(WSCloseProtocolError, "protocol error")
			} else if errors.Is(err, errWSTooBig) {
				c.closeWith(WSCloseMessageTooBig, "message too big")
			}
			c.Close()
			return "", false
		}

		switch opcode {
		case wsOpPing:
			if err := c.writeFrame(wsOpPong, payload); err != nil {
				c.Close()
				return "", false
			}
		case wsOpPong:
			// A leitura ja renovou o prazo; nao ha mais nada a fazer.
		case wsOpClose:
			code, reason := parseClosePayload(payload)
			c.closeWith(code, reason)
			c.Close()
			return "", false
		case wsOpText, wsOpBinary:
			if fragmentOp != 0 {
				c.closeWith(WSCloseProtocolError, "interleaved data frame")
				c.Close()
				return "", false
			}
			if !final {
				fragmentOp = opcode
				message = append(message, payload...)
				continue
			}
			if opcode == wsOpBinary {
				continue // a LIAF so expoe mensagens de texto
			}
			if !utf8.Valid(payload) {
				c.closeWith(WSCloseInvalidPayload, "invalid UTF-8")
				c.Close()
				return "", false
			}
			return string(payload), true
		case wsOpContinuation:
			if fragmentOp == 0 {
				c.closeWith(WSCloseProtocolError, "continuation without a start frame")
				c.Close()
				return "", false
			}
			if len(message)+len(payload) > wsMaxMessageSize {
				c.closeWith(WSCloseMessageTooBig, "message too big")
				c.Close()
				return "", false
			}
			message = append(message, payload...)
			if !final {
				continue
			}
			complete := message
			wasText := fragmentOp == wsOpText
			message, fragmentOp = nil, 0
			if !wasText {
				continue
			}
			if !utf8.Valid(complete) {
				c.closeWith(WSCloseInvalidPayload, "invalid UTF-8")
				c.Close()
				return "", false
			}
			return string(complete), true
		default:
			c.closeWith(WSCloseProtocolError, "unknown opcode")
			c.Close()
			return "", false
		}
	}
}

var (
	errWSProtocol = errors.New("websocket protocol error")
	errWSTooBig   = errors.New("websocket message too big")
)

func (c *WSConn) readFrame() (final bool, opcode byte, payload []byte, err error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(c.r, header); err != nil {
		return false, 0, nil, err
	}
	final = header[0]&0x80 != 0
	if header[0]&0x70 != 0 {
		// RSV1-3 so podem estar ligados com uma extensao negociada, e nenhuma
		// e oferecida no handshake.
		return false, 0, nil, errWSProtocol
	}
	opcode = header[0] & 0x0f
	masked := header[1]&0x80 != 0
	size := uint64(header[1] & 0x7f)

	switch size {
	case 126:
		extended := make([]byte, 2)
		if _, err := io.ReadFull(c.r, extended); err != nil {
			return false, 0, nil, err
		}
		size = uint64(binary.BigEndian.Uint16(extended))
	case 127:
		extended := make([]byte, 8)
		if _, err := io.ReadFull(c.r, extended); err != nil {
			return false, 0, nil, err
		}
		size = binary.BigEndian.Uint64(extended)
	}

	isControl := opcode&0x8 != 0
	if isControl && (size > 125 || !final) {
		// Frames de controle nao podem ser fragmentados nem longos.
		return false, 0, nil, errWSProtocol
	}
	if size > wsMaxMessageSize {
		return false, 0, nil, errWSTooBig
	}
	// A RFC exige mascara em todo frame vindo do cliente; aceitar sem mascara
	// abriria o ataque de cache poisoning que a mascara existe para impedir.
	if !masked {
		return false, 0, nil, errWSProtocol
	}

	mask := make([]byte, 4)
	if _, err := io.ReadFull(c.r, mask); err != nil {
		return false, 0, nil, err
	}
	payload = make([]byte, size)
	if _, err := io.ReadFull(c.r, payload); err != nil {
		return false, 0, nil, err
	}
	for i := range payload {
		payload[i] ^= mask[i%4]
	}
	return final, opcode, payload, nil
}

func parseClosePayload(payload []byte) (int, string) {
	if len(payload) < 2 {
		return WSCloseNormal, ""
	}
	return int(binary.BigEndian.Uint16(payload[:2])), string(payload[2:])
}

// --- Escrita ----------------------------------------------------------------

// Send envia uma mensagem de texto.
func (c *WSConn) Send(message string) error {
	return c.writeFrame(wsOpText, []byte(message))
}

// SendJSON serializa e envia. Existe separado de Send porque a alternativa na
// LIAF seria (ws-send conn (try (json-encode v))), que obriga a tratar um
// erro de serializacao que quase nunca acontece.
func (c *WSConn) SendJSON(value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("could not serialize the message: %w", err)
	}
	return c.writeFrame(wsOpText, data)
}

func (c *WSConn) writeFrame(opcode byte, payload []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.isClosed() {
		return errors.New("connection is closed")
	}

	header := []byte{0x80 | opcode} // FIN ligado: nunca fragmentamos na saida
	size := len(payload)
	switch {
	case size < 126:
		header = append(header, byte(size))
	case size <= 0xffff:
		header = binary.BigEndian.AppendUint16(append(header, 126), uint16(size))
	default:
		header = binary.BigEndian.AppendUint64(append(header, 127), uint64(size))
	}
	// Frames do servidor nunca sao mascarados (RFC 6455, secao 5.1).

	_ = c.conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
	if _, err := c.conn.Write(append(header, payload...)); err != nil {
		return err
	}
	return nil
}

// CloseWith envia o frame de fechamento e encerra a conexao.
func (c *WSConn) CloseWith(code int, reason string) error {
	c.closeWith(code, reason)
	return c.Close()
}

func (c *WSConn) closeWith(code int, reason string) {
	if c.isClosed() {
		return
	}
	if len(reason) > 123 { // 125 do frame de controle menos os 2 do codigo
		reason = reason[:123]
	}
	payload := binary.BigEndian.AppendUint16(nil, uint16(code))
	_ = c.writeFrame(wsOpClose, append(payload, reason...))
}

// Close encerra a conexao e tira o cliente de todos os topicos. E idempotente:
// o laco de leitura e o handler podem chama-la em qualquer ordem.
func (c *WSConn) Close() error {
	c.closeMu.Lock()
	if c.closed {
		c.closeMu.Unlock()
		return nil
	}
	c.closed = true
	close(c.done)
	c.closeMu.Unlock()

	c.hub.leaveAll(c)
	return c.conn.Close()
}

func (c *WSConn) isClosed() bool {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	return c.closed
}

// keepalive envia pings periodicos. Sem isso, um cliente que sumiu sem
// fechar (queda de rede, notebook fechado) so seria detectado no proximo
// envio, e a entrada dele no topico seguiria recebendo broadcast.
func (c *WSConn) keepalive() {
	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			if err := c.writeFrame(wsOpPing, nil); err != nil {
				_ = c.Close()
				return
			}
		}
	}
}

// --- Topicos ----------------------------------------------------------------

// wsHub e o pub/sub em memoria. Atende o caso local e leve descrito na issue
// #016; nao substitui um broker quando ha mais de um processo, porque cada
// processo tem o seu hub.
type wsHub struct {
	mu     sync.RWMutex
	topics map[string]map[*WSConn]bool
}

var defaultHub = &wsHub{topics: map[string]map[*WSConn]bool{}}

func (h *wsHub) join(c *WSConn, topic string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.topics[topic] == nil {
		h.topics[topic] = map[*WSConn]bool{}
	}
	h.topics[topic][c] = true
	c.topics[topic] = true
}

func (h *wsHub) leave(c *WSConn, topic string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.removeLocked(c, topic)
}

func (h *wsHub) leaveAll(c *WSConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for topic := range c.topics {
		h.removeLocked(c, topic)
	}
}

func (h *wsHub) removeLocked(c *WSConn, topic string) {
	if members := h.topics[topic]; members != nil {
		delete(members, c)
		if len(members) == 0 {
			delete(h.topics, topic)
		}
	}
	delete(c.topics, topic)
}

// broadcast envia a todos os inscritos e devolve quantos receberam. A copia
// da lista sai debaixo do lock antes do envio: escrever com o lock preso
// deixaria um cliente lento bloqueando todo o topico.
func (h *wsHub) broadcast(topic, message string) int {
	h.mu.RLock()
	members := make([]*WSConn, 0, len(h.topics[topic]))
	for c := range h.topics[topic] {
		members = append(members, c)
	}
	h.mu.RUnlock()

	delivered := 0
	for _, c := range members {
		if err := c.Send(message); err != nil {
			// Um envio que falha significa conexao morta; fechar aqui tira
			// o cliente do topico e acorda o laco de leitura dele.
			_ = c.Close()
			continue
		}
		delivered++
	}
	return delivered
}

// WSJoin inscreve a conexao num topico.
func WSJoin(c *WSConn, topic string) { c.hub.join(c, topic) }

// WSLeave cancela a inscricao. O fechamento da conexao ja faz isso em todos
// os topicos; esta chamada serve para sair de um sem encerrar a conexao.
func WSLeave(c *WSConn, topic string) { c.hub.leave(c, topic) }

// WSBroadcast envia a mensagem a todos os inscritos no topico.
func WSBroadcast(topic, message string) int { return defaultHub.broadcast(topic, message) }

// WSTopicSize informa quantos clientes estao inscritos.
func WSTopicSize(topic string) int {
	defaultHub.mu.RLock()
	defer defaultHub.mu.RUnlock()
	return len(defaultHub.topics[topic])
}
