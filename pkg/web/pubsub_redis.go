package web

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultRedisPubSubChannel = "liaf:ws:pubsub"
	redisDialTimeout          = 5 * time.Second
	redisIOTimeout            = 10 * time.Second
)

type wsRedisEnvelope struct {
	Origin string `json:"origin"`
	Topic  string `json:"topic"`
	Msg    string `json:"msg"`
}

// RedisPubSubBroker implementa WSPubSubBroker conectando-se a um servidor Redis
// via RESP2 puro sobre TCP, permitindo a sincronizacao de WebSockets entre multiplas instancias (#017).
type RedisPubSubBroker struct {
	dsn     string
	channel string
	nodeID  string
	hub     *wsHub

	mu      sync.Mutex
	pubConn net.Conn
	pubW    *bufio.Writer
	pubR    *bufio.Reader

	closed bool
	done   chan struct{}
}

// NewRedisPubSubBroker cria e inicializa o broker distribuido.
func NewRedisPubSubBroker(dsn string, hub *wsHub) (*RedisPubSubBroker, error) {
	if hub == nil {
		hub = defaultHub
	}
	idBytes := make([]byte, 8)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, fmt.Errorf("could not generate node id: %w", err)
	}
	nodeID := hex.EncodeToString(idBytes)

	broker := &RedisPubSubBroker{
		dsn:     dsn,
		channel: defaultRedisPubSubChannel,
		nodeID:  nodeID,
		hub:     hub,
		done:    make(chan struct{}),
	}

	// Testa a conexao de publicacao antes de concluir
	if err := broker.connectPublisher(); err != nil {
		return nil, err
	}

	// Inicia a goroutine de subscricao com reconexao automatica
	go broker.subscribeLoop()

	return broker, nil
}

func (b *RedisPubSubBroker) connectPublisher() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.pubConn != nil {
		_ = b.pubConn.Close()
	}

	conn, r, w, err := dialRedis(b.dsn)
	if err != nil {
		return err
	}
	b.pubConn = conn
	b.pubR = r
	b.pubW = w
	return nil
}

// Publish publica a mensagem de um topico para o cluster Redis.
func (b *RedisPubSubBroker) Publish(topic, message string) error {
	payload, err := json.Marshal(wsRedisEnvelope{
		Origin: b.nodeID,
		Topic:  topic,
		Msg:    message,
	})
	if err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return errors.New("redis pubsub broker is closed")
	}

	if b.pubConn == nil {
		if err := b.reconnectPublisherLocked(); err != nil {
			return err
		}
	}

	_ = b.pubConn.SetDeadline(time.Now().Add(redisIOTimeout))
	if err := sendRESPCommand(b.pubW, "PUBLISH", b.channel, string(payload)); err != nil {
		_ = b.pubConn.Close()
		b.pubConn = nil
		// Tenta reconectar 1 vez
		if recErr := b.reconnectPublisherLocked(); recErr == nil {
			_ = b.pubConn.SetDeadline(time.Now().Add(redisIOTimeout))
			if retryErr := sendRESPCommand(b.pubW, "PUBLISH", b.channel, string(payload)); retryErr == nil {
				_, _ = readRESPReply(b.pubR)
				return nil
			}
		}
		return fmt.Errorf("redis publish failed: %w", err)
	}

	_, err = readRESPReply(b.pubR)
	return err
}

func (b *RedisPubSubBroker) reconnectPublisherLocked() error {
	conn, r, w, err := dialRedis(b.dsn)
	if err != nil {
		return err
	}
	b.pubConn = conn
	b.pubR = r
	b.pubW = w
	return nil
}

// subscribeLoop mantem a conexao de escuta (SUBSCRIBE) ativa e processa envelopes.
func (b *RedisPubSubBroker) subscribeLoop() {
	backoff := 100 * time.Millisecond

	for {
		select {
		case <-b.done:
			return
		default:
		}

		conn, r, w, err := dialRedis(b.dsn)
		if err != nil {
			select {
			case <-b.done:
				return
			case <-time.After(backoff):
				if backoff < 3*time.Second {
					backoff *= 2
				}
				continue
			}
		}
		backoff = 100 * time.Millisecond

		// Envia comando SUBSCRIBE
		_ = conn.SetDeadline(time.Now().Add(redisIOTimeout))
		if err := sendRESPCommand(w, "SUBSCRIBE", b.channel); err != nil {
			_ = conn.Close()
			continue
		}
		// Le confirmacao de subscricao: ["subscribe", channel, 1]
		_, err = readRESPReply(r)
		if err != nil {
			_ = conn.Close()
			continue
		}
		_ = conn.SetDeadline(time.Time{}) // sem deadline para esperar mensagens

		for {
			select {
			case <-b.done:
				_ = conn.Close()
				return
			default:
			}

			reply, err := readRESPReply(r)
			if err != nil {
				_ = conn.Close()
				break
			}

			items, ok := reply.([]any)
			if !ok || len(items) < 3 {
				continue
			}

			kind, _ := items[0].(string)
			if kind != "message" {
				continue
			}

			payloadStr, _ := items[2].(string)
			var env wsRedisEnvelope
			if err := json.Unmarshal([]byte(payloadStr), &env); err != nil {
				continue
			}

			// Se a mensagem veio deste mesmo no, descarta para nao duplicar localmente
			if env.Origin == b.nodeID {
				continue
			}

			// Entrega localmente para os clientes conectados nesta instancia
			b.hub.broadcastLocal(env.Topic, env.Msg)
		}
	}
}

// Close encerra o broker.
func (b *RedisPubSubBroker) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	close(b.done)
	if b.pubConn != nil {
		_ = b.pubConn.Close()
		b.pubConn = nil
	}
	b.mu.Unlock()
	return nil
}

// EnableRedisPubSub ativa a sincronizacao de WebSockets via Redis no hub padrao.
func EnableRedisPubSub(dsn string) error {
	broker, err := NewRedisPubSubBroker(dsn, defaultHub)
	if err != nil {
		return err
	}
	defaultHub.SetBroker(broker)
	return nil
}

// DisableRedisPubSub desativa a sincronizacao externa e volta a operar localmente.
func DisableRedisPubSub() {
	defaultHub.SetBroker(nil)
}

// --- Protocolo RESP2 Auxiliar -----------------------------------------------

func dialRedis(dsn string) (net.Conn, *bufio.Reader, *bufio.Writer, error) {
	dsn = strings.TrimSpace(dsn)
	host := "127.0.0.1"
	port := "6379"
	user := ""
	password := ""
	dbIndex := ""

	if strings.Contains(dsn, "://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("invalid redis dsn: %w", err)
		}
		if h := u.Hostname(); h != "" {
			host = h
		}
		if p := u.Port(); p != "" {
			port = p
		}
		if u.User != nil {
			user = u.User.Username()
			if pw, ok := u.User.Password(); ok {
				password = pw
			}
		}
		dbIndex = strings.TrimPrefix(u.Path, "/")
	} else if dsn != "" {
		if strings.Contains(dsn, ":") {
			h, p, err := net.SplitHostPort(dsn)
			if err == nil {
				host, port = h, p
			}
		} else {
			host = dsn
		}
	}

	addr := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", addr, redisDialTimeout)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("cannot connect to redis at %s: %w", addr, err)
	}

	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)

	// Autenticacao se especificada
	if password != "" {
		_ = conn.SetDeadline(time.Now().Add(redisIOTimeout))
		var authArgs []string
		if user != "" {
			authArgs = []string{"AUTH", user, password}
		} else {
			authArgs = []string{"AUTH", password}
		}
		if err := sendRESPCommand(w, authArgs...); err != nil {
			_ = conn.Close()
			return nil, nil, nil, err
		}
		if _, err := readRESPReply(r); err != nil {
			_ = conn.Close()
			return nil, nil, nil, fmt.Errorf("redis auth failed: %w", err)
		}
	}

	// Seleciona database se especificada
	if dbIndex != "" && dbIndex != "0" {
		_ = conn.SetDeadline(time.Now().Add(redisIOTimeout))
		if err := sendRESPCommand(w, "SELECT", dbIndex); err != nil {
			_ = conn.Close()
			return nil, nil, nil, err
		}
		if _, err := readRESPReply(r); err != nil {
			_ = conn.Close()
			return nil, nil, nil, fmt.Errorf("redis select %s failed: %w", dbIndex, err)
		}
	}

	_ = conn.SetDeadline(time.Time{})
	return conn, r, w, nil
}

func sendRESPCommand(w *bufio.Writer, args ...string) error {
	fmt.Fprintf(w, "*%d\r\n", len(args))
	for _, a := range args {
		fmt.Fprintf(w, "$%d\r\n%s\r\n", len(a), a)
	}
	return w.Flush()
}

func readRESPReply(r *bufio.Reader) (any, error) {
	prefix, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	line, err := readRESPLine(r)
	if err != nil {
		return nil, err
	}
	switch prefix {
	case '+':
		return line, nil
	case '-':
		return nil, fmt.Errorf("redis: %s", line)
	case ':':
		return strconv.ParseInt(line, 10, 64)
	case '$':
		n, err := strconv.Atoi(line)
		if err != nil {
			return nil, fmt.Errorf("redis: invalid bulk length %q", line)
		}
		if n < 0 {
			return nil, nil
		}
		buf := make([]byte, n+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		return string(buf[:n]), nil
	case '*':
		n, err := strconv.Atoi(line)
		if err != nil {
			return nil, fmt.Errorf("redis: invalid array length %q", line)
		}
		if n < 0 {
			return nil, nil
		}
		items := make([]any, 0, n)
		for i := 0; i < n; i++ {
			item, err := readRESPReply(r)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		return items, nil
	default:
		return nil, fmt.Errorf("redis: unsupported reply prefix %q", string(prefix))
	}
}

func readRESPLine(r *bufio.Reader) (string, error) {
	s, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(s, "\r\n"), nil
}
