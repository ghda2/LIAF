package web

import "sync"

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

	if len(members) == 0 {
		return 0
	}

	frame := makeWSFrame(wsOpText, []byte(message))

	delivered := 0
	for _, c := range members {
		if err := c.writeRawFrame(frame); err != nil {
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
