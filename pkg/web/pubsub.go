package web

import "sync"

// WSPubSubBroker define a interface para propagacao de mensagens entre instancias (#017).
type WSPubSubBroker interface {
	Publish(topic, message string) error
	Close() error
}

// wsHub e o pub/sub em memoria com suporte a broker distribuido (#017).
type wsHub struct {
	mu     sync.RWMutex
	topics map[string]map[*WSConn]bool
	broker WSPubSubBroker
}

var defaultHub = &wsHub{topics: map[string]map[*WSConn]bool{}}

// SetBroker anexa um broker externo (ex: Redis PubSub) ao hub.
func (h *wsHub) SetBroker(b WSPubSubBroker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.broker != nil {
		_ = h.broker.Close()
	}
	h.broker = b
}

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

// broadcastLocal envia exclusivamente para os clientes conectados a este no.
func (h *wsHub) broadcastLocal(topic, message string) int {
	h.mu.RLock()
	members := make([]*WSConn, 0, len(h.topics[topic]))
	for c := range h.topics[topic] {
		members = append(members, c)
	}
	h.mu.RUnlock()

	if len(members) == 0 {
		return 0
	}

	frame := makeWSFrame(wsOpText, []byte(message))

	delivered := 0
	for _, c := range members {
		if err := c.writeRawFrame(frame); err != nil {
			_ = c.Close()
			continue
		}
		delivered++
	}
	return delivered
}

// broadcast envia a todos os inscritos locais e distribui no cluster se houver broker ativo.
func (h *wsHub) broadcast(topic, message string) int {
	delivered := h.broadcastLocal(topic, message)

	h.mu.RLock()
	b := h.broker
	h.mu.RUnlock()

	if b != nil {
		_ = b.Publish(topic, message)
	}
	return delivered
}

// WSJoin inscreve a conexao num topico.
func WSJoin(c *WSConn, topic string) { c.hub.join(c, topic) }

// WSLeave cancela a inscricao. O fechamento da conexao ja faz isso em todos
// os topicos; esta chamada serve para sair de um sem encerrar a conexao.
func WSLeave(c *WSConn, topic string) { c.hub.leave(c, topic) }

// WSBroadcast envia a mensagem a todos os inscritos no topico (local + cluster).
func WSBroadcast(topic, message string) int { return defaultHub.broadcast(topic, message) }

// WSTopicSize informa quantos clientes estao inscritos.
func WSTopicSize(topic string) int {
	defaultHub.mu.RLock()
	defer defaultHub.mu.RUnlock()
	return len(defaultHub.topics[topic])
}

// SetWSPubSubBroker define o broker distribuido para o hub padrao.
func SetWSPubSubBroker(b WSPubSubBroker) {
	defaultHub.SetBroker(b)
}
