package web

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockRedisServer simula um broker Redis minimo para testar pub/sub multi-instancia.
type mockRedisServer struct {
	listener    net.Listener
	mu          sync.Mutex
	subscribers []net.Conn
	closed      bool
}

func startMockRedis(t *testing.T) *mockRedisServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("falha ao iniciar mock redis: %v", err)
	}

	m := &mockRedisServer{listener: ln}
	go m.acceptLoop()
	return m
}

func (m *mockRedisServer) addr() string {
	return m.listener.Addr().String()
}

func (m *mockRedisServer) close() {
	m.mu.Lock()
	m.closed = true
	_ = m.listener.Close()
	for _, c := range m.subscribers {
		_ = c.Close()
	}
	m.mu.Unlock()
}

func (m *mockRedisServer) acceptLoop() {
	for {
		conn, err := m.listener.Accept()
		if err != nil {
			return
		}
		go m.handleConn(conn)
	}
}

func (m *mockRedisServer) handleConn(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)

	for {
		reply, err := readRESPReply(r)
		if err != nil {
			m.mu.Lock()
			for i, sub := range m.subscribers {
				if sub == conn {
					m.subscribers = append(m.subscribers[:i], m.subscribers[i+1:]...)
					break
				}
			}
			m.mu.Unlock()
			return
		}

		items, ok := reply.([]any)
		if !ok || len(items) == 0 {
			continue
		}

		cmd, _ := items[0].(string)
		cmd = strings.ToUpper(cmd)

		switch cmd {
		case "PING":
			w.WriteString("+PONG\r\n")
			w.Flush()
		case "SUBSCRIBE":
			m.mu.Lock()
			m.subscribers = append(m.subscribers, conn)
			m.mu.Unlock()

			// *3\r\n$9\r\nsubscribe\r\n$14\r\nliaf:ws:pubsub\r\n:1\r\n
			w.WriteString("*3\r\n$9\r\nsubscribe\r\n$14\r\nliaf:ws:pubsub\r\n:1\r\n")
			w.Flush()
		case "PUBLISH":
			// items[1] = channel, items[2] = message
			w.WriteString(":1\r\n")
			w.Flush()

			if len(items) >= 3 {
				ch, _ := items[1].(string)
				msg, _ := items[2].(string)
				m.broadcastToSubs(ch, msg)
			}
		default:
			w.WriteString("+OK\r\n")
			w.Flush()
		}
	}
}

func (m *mockRedisServer) broadcastToSubs(channel, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	frame := fmt.Sprintf("*3\r\n$7\r\nmessage\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n",
		len(channel), channel, len(message), message)

	for _, sub := range m.subscribers {
		_, _ = sub.Write([]byte(frame))
	}
}

func TestRedisPubSubMultiInstanceBroadcast(t *testing.T) {
	mock := startMockRedis(t)
	defer mock.close()

	dsn := "redis://" + mock.addr()

	// Simula duas instancias separadas da aplicacao com seus proprios hubs
	hubA := &wsHub{topics: map[string]map[*WSConn]bool{}}
	hubB := &wsHub{topics: map[string]map[*WSConn]bool{}}

	brokerA, err := NewRedisPubSubBroker(dsn, hubA)
	if err != nil {
		t.Fatalf("erro ao criar broker A: %v", err)
	}
	defer brokerA.Close()
	hubA.SetBroker(brokerA)

	brokerB, err := NewRedisPubSubBroker(dsn, hubB)
	if err != nil {
		t.Fatalf("erro ao criar broker B: %v", err)
	}
	defer brokerB.Close()
	hubB.SetBroker(brokerB)

	// Pequeno delay para a subscricao de B estabilizar
	time.Sleep(50 * time.Millisecond)

	// Cliente 1 conecta na Instancia A
	client1Pipe, client1Remote := net.Pipe()
	defer client1Pipe.Close()
	defer client1Remote.Close()

	conn1 := &WSConn{
		conn:       client1Pipe,
		topics:     map[string]bool{},
		hub:        hubA,
		done:       make(chan struct{}),
		sendQueue:  make(chan []byte, 32),
		slowPolicy: WSSlowClientDisconnect,
	}
	go conn1.writePump()
	hubA.join(conn1, "pedidos")

	// Cliente 2 conecta na Instancia B
	client2Pipe, client2Remote := net.Pipe()
	defer client2Pipe.Close()
	defer client2Remote.Close()

	conn2 := &WSConn{
		conn:       client2Pipe,
		topics:     map[string]bool{},
		hub:        hubB,
		done:       make(chan struct{}),
		sendQueue:  make(chan []byte, 32),
		slowPolicy: WSSlowClientDisconnect,
	}
	go conn2.writePump()
	hubB.join(conn2, "pedidos")

	// Instancia A emite um broadcast para o topico "pedidos"
	hubA.broadcast("pedidos", "novo_pedido_123")

	// Verifica se o Cliente 1 (conectado em A) recebeu
	r1 := bufio.NewReader(client1Remote)
	_ = client1Remote.SetReadDeadline(time.Now().Add(2 * time.Second))
	c1 := &wsClient{conn: client1Remote, r: r1, t: t}
	got1 := c1.expectText()
	if got1 != "novo_pedido_123" {
		t.Fatalf("cliente 1 em A esperava 'novo_pedido_123', obteve %q", got1)
	}

	// Verifica se o Cliente 2 (conectado em B!) recebeu via Redis
	r2 := bufio.NewReader(client2Remote)
	_ = client2Remote.SetReadDeadline(time.Now().Add(2 * time.Second))
	c2 := &wsClient{conn: client2Remote, r: r2, t: t}
	got2 := c2.expectText()
	if got2 != "novo_pedido_123" {
		t.Fatalf("cliente 2 em B esperava 'novo_pedido_123' sincronizado via Redis, obteve %q", got2)
	}
}
