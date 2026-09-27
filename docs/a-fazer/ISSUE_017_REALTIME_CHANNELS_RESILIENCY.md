# Issue #017: Canais em Tempo Real de Baixo Consumo e Resiliência (Estilo Phoenix/BEAM)

**Status:** Aberta  
**Componente:** `std/realtime`, `pkg/checker`, `pkg/codegen`, `pkg/web`, `runtime/`  
**Data:** 16 de setembro de 2026  

---

## 1. Contexto, Inspiração e Filosofia LIAF

Frameworks como **Phoenix (Elixir/BEAM)** tornaram-se padrão da indústria em tempo real por oferecerem:
1. Abstração unificada de tópicos/canais (*Channels*).
2. Estado isolado por conexão (`socket.assigns`).
3. Ganchos declarativos de ciclo de vida (`join`, `handle_in`, `terminate`).

Entretanto, pilares fundamentais do **LIAF** são a **eficiência máxima para IA, consumo irrisório de CPU e RAM (<5 MB de baseline)** e binários nativos autônomos. Inspirando-se no Phoenix, mas extraindo a performance de baixo nível de implementações como **uWebSockets (C++)** e **epoll/goroutines (Go/Rust)**, esta issue define a camada de alta resiliência e escala em tempo real do LIAF.

---

## 2. Proposta Técnica

### 2.1. Canais Declarativos com Estado Tipado
Em vez de lidar com sockets crus, a aplicação define canais estruturados onde cada conexão possui seu estado tipado em memória:
```lisp
(channel "chat:{room_id}" 
  (state UserState (user_id str) (role str))
  (effects net io)

  (on-join (params (room_id str) (conn WSConn))
    (let ((user (try (auth-token (ws-header conn "Authorization")))))
      (ws-assign conn (UserState user.id user.role))
      (ws-join conn (concat "chat:" room_id))
      (ok "joined")))

  (on-message (params (conn WSConn) (msg str))
    (let ((st (ws-state conn UserState)))
      (ws-broadcast (concat "chat:" room_id) (json-encode (ChatMessage st.user_id msg)))))

  (on-leave (params (conn WSConn) (reason str))
    (println "Cliente desconectou")))
```

### 2.2. Zero-Copy Broadcast (CPU & Alocação Mínima)
- Ao fazer `ws-broadcast` para 10.000 clientes em uma sala:
  - O payload/frame WebSocket é **serializado e formatado exatamente uma vez** na memória.
  - O mesmo buffer imutável é compartilhado via ponteiro entre todos os canais de escrita dos clientes.
  - Elimina pressão de Garbage Collector e reduz o uso de CPU para $O(N)$ escritas puras de rede.

### 2.3. Backpressure e Bounded Ring Buffers (Proteção de RAM)
- Cada conexão possui uma fila de envio com capacidade estrita e limitada (ex: 64 frames pré-alocados).
- Prevenção contra *Slow Clients* (clientes lentos ou maliciosos acumulando RAM do servidor):
  - Modo configurável: `drop-oldest` ou `disconnect-slow`.
  - Garante que mesmo com milhares de conexões, a pegada de RAM permaneça estritamente previsível e delimitada.

### 2.4. Heartbeat Ativo e Limpeza de Sockets Zumbis
- Ciclo de Ping/Pong integrado ao loop de I/O de rede de baixíssimo custo de CPU:
  - Detecção automática de desconexões abruptas (quedas de Wi-Fi/4G sem TCP FIN).
  - Remoção imediata do socket das salas de broadcast (`wsHub`) sem vazamento de ponteiros ou conexões fantasmas.

### 2.5. Proteção DoS e Limites Rígidos por Padrão
- Limite máximo de tamanho de payload (default 64 KB, ajustável).
- Rate-limiting nativo por conexão (máximo de mensagens/segundo) para proteger o backend de inundações.

---

## 3. Metas de Desempenho e Recursos
- **Baseline de RAM:** Manter o servidor com 1.000 conexões ativas ociosas em **menos de 15 MB de RAM**.
- **CPU em repouso:** 0.0% de CPU sem conexões ativas ou em keepalive.
- **Zero Brokers Externos:** Broadcast local e canais 100% autossuficientes sem necessidade de Redis ou dependências pesadas.

---

## 4. Tarefas de Implementação
- [ ] Implementar buffer de envio limitado (*bounded ring-buffer*) com política de descarte/desconexão lenta em `pkg/web/websocket.go`.
- [ ] Implementar `ws-broadcast` otimizado com frame pré-construído (Zero-Copy single-encode).
- [ ] Definir construção sintática `(channel ...)` no parser e AST (`pkg/ast`).
- [ ] Adicionar checagem de tipos para estado de canal (`ws-assign` e `ws-state`) no `pkg/checker`.
- [ ] Implementar timers eficientes de Ping/Pong heartbeat com remoção automática de conexões ociosas/mortas.
- [ ] Adicionar suporte a rate-limit de mensagens por conexão na camada de rede.
- [ ] Escrever benchmark comparativo de consumo de memória e latência sob 1.000 e 5.000 conexões simultâneas.

---

Contrato atual e especificações: [docs/linguagem/SPEC.md](../linguagem/SPEC.md). Evidências consolidadas: [STATUS.md](../STATUS.md).
