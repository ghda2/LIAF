# Issue #017: Canais em Tempo Real de Baixo Consumo e Resiliência (Estilo Phoenix/BEAM)

**Status:** Concluída  
**Componente:** `pkg/web`, `pkg/runtime`, `pkg/builtins`  
**Data:** 17 de outubro de 2026  

---

## 1. Contexto e Resolução

Inspirando-se no Phoenix (Elixir/BEAM) e na arquitetura de baixa alocação do uWebSockets, a camada de WebSockets da LIAF foi expandida com foco em resiliência extrema contra inundações (DoS), lentidão de rede e particionamento em clusters de múltiplas instâncias.

---

## 2. O que foi entregue

1. **Fila Limitada & Proteção contra Slow Clients (`sendQueue`):**
   - Buffer de saída desacoplado via goroutine `writePump` por conexão (default: 64 frames, configurável via `LIAF_WS_QUEUE_SIZE`).
   - Política configurável: `WSSlowClientDisconnect` (default, fecha com código RFC 6455 1008 `WSClosePolicyViolation`) e `WSSlowClientDropOldest`.
   - O broadcast não bloqueia: distribuição não-bloqueante para todos os clientes em $O(N)$ microssegundos.
   - Envio priorizado e síncrono do frame de fechamento com `writeDirect`.

2. **Rate Limiting Nativo por Conexão:**
   - Janela deslizante de 1 segundo rastreando contagem de mensagens recebidas (default: 100 msgs/s, configurável via `LIAF_WS_RATE_LIMIT` ou `.SetRateLimit(n)`).
   - Clientes que violam o limite são desconectados imediatamente com código 1008 `WSClosePolicyViolation` ("rate limit exceeded").

3. **Pub/Sub Escalável via Redis (Multi-Instância):**
   - Módulo `pkg/web/pubsub_redis.go` implementando `WSPubSubBroker` sobre RESP2 TCP nativo (sem dependências externas).
   - Cada nó gera um identificador único (`nodeID`) para evitar loops e replicação indevida (anti-eco).
   - Reconexão automática transparente com backoff exponencial.
   - Builtins expostas em LIAF: `ws-enable-redis(dsn)` e `ws-disable-redis()`.

---

## 3. Evidências de Testes

- `TestWebSocketRateLimitDisconnectsFlooder`: validação de desconexão RFC 6455 1008 ao ultrapassar limite de taxa.
- `TestWebSocketSlowClientDisconnect`: validação de detecção e desconexão de clientes lentos antes de vazamento de memória.
- `TestRedisPubSubMultiInstanceBroadcast`: validação de dois servidores LIAF independentes (`hubA` e `hubB`) sincronizando broadcasts instantaneamente através do broker Redis mock.
- Suíte completa `go test ./...` executada e aprovada com 100% de sucesso.
