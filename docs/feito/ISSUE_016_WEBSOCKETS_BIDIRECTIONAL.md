# Issue #016: WebSockets e Conexões Bidirecionais Contínuas

**Status:** Implementada (16/09/2026)  
**Componente:** `pkg/web`, `pkg/runtime`, `pkg/checker`, `pkg/codegen`, `pkg/parser`, `pkg/ast`  
**Data:** 16 de setembro de 2026  

---

## 1. Contexto e Motivação
Aplicações modernas interativas (chats, dashboards em tempo real, colaboração multi-usuário e streaming de eventos de IA) exigem conexões bidirecionais de baixa latência. O modelo HTTP puro request-response suportado na v0.3 não atende comunicações orientadas a eventos e push do servidor.

---

## 2. Proposta Técnica

### 2.1. Rotas Declarativas de WebSocket
Semelhante às rotas HTTP declarativas da v0.3:
```lisp
(ws-route "/ws/chat/{room}" (params (room str) (conn WSConn)) (effects net io)
  (on-message msg
    (ws-broadcast room (concat "Novo: " msg)))
  (on-close
    (println "Conexão encerrada")))
```

### 2.2. Primitivas de Conexão WebSocket
- Tipo `WSConn`.
- Builtins essenciais:
  - `(ws-send conn message)` -> `(result void str)`.
  - `(ws-send-json conn data)` -> `(result void str)`.
  - `(ws-close conn code reason)` -> `(result void str)`.
  - `(ws-broadcast topic message)` -> `(result int str)` (número de clientes que receberam).

### 2.3. Salas / Tópicos (Pub/Sub Integrado)
- Suporte a agregação por canais/tópicos na camada de web server sem necessidade de infraestrutura externa para casos locais/leves.

---

## 3. Tarefas de Implementação
- [x] Definir tipo `WSConn` na AST e checker.
- [x] Integrar upgrade HTTP para WebSocket no servidor híbrido (`pkg/web`).
- [x] Implementar loop de leitura/escrita assíncrona tolerante a desconexões súbitas.
- [x] Implementar pub/sub local seguro em memória para broadcast.
- [x] Criar exemplo prático em `examples/chat_ws.liaf` com mini frontend demonstrativo (`public/chat.html`).

---

## 4. Decisões tomadas na implementação

### 4.1 RFC 6455 implementada aqui, sem `gorilla/websocket`

Mesma razão da issue #015: o enquadramento cabe num arquivo (`pkg/web/websocket.go`) e `go.mod` continua só com `golang.org/x/*`. O binário LIAF segue único.

Implementado: handshake com `Sec-WebSocket-Accept`, frames de texto e binários, remontagem de fragmentos na leitura, ping/pong, fechamento com código e motivo, exigência de máscara em todo frame do cliente, teto de 1 MiB por mensagem (o mesmo do corpo HTTP), ping a cada 30 s e queda do ocioso em 90 s.

Não implementado: extensões negociadas (`permessage-deflate`), fragmentação na escrita e o papel de cliente.

### 4.2 `(on-open ...)` acrescentado

A §2.1 listava só `on-message` e `on-close`. Sem um gancho de abertura, `ws-join` não teria onde ser chamado e o exemplo de chat da própria issue não funcionaria — a conexão nunca se inscreveria na sala. `on-open` é opcional, como `on-close`.

### 4.3 `ws-join` e `ws-leave` acrescentados; inscrição é explícita

A §2.3 pedia "agregação por canais/tópicos", e o exemplo chamava `(ws-broadcast room ...)` sem dizer quem inscreve a conexão em `room`. Inferir a sala do caminho seria implícito demais: um caminho pode ter mais de um parâmetro, e nem toda `ws-route` é uma sala.

`ws-join` e `ws-leave` devolvem `void`, e não `result`: operam sobre um mapa em memória e não têm caminho de falha. Fechar a conexão desinscreve de todos os tópicos, então `on-close` não precisa de `ws-leave`.

`ws-broadcast` devolve `(result int str)` como a issue especifica, ainda que a implementação local nunca falhe — o tipo deixa espaço para um broker externo no futuro.

### 4.4 Parâmetro de rota restrito ao caminho

Um handshake de WebSocket não tem corpo JSON, ao contrário de uma `route` POST. Um parâmetro que não venha de `{nome}` no caminho não teria de onde ser preenchido, então é `E_WS_PARAM` em vez de virar o zero do tipo silenciosamente.

---

## 5. O que foi verificado

- `pkg/web`: eco (incluindo UTF-8 multibyte e payload de 70 KB, que exerce o tamanho estendido de 8 bytes), remontagem de fragmentos, ping respondido com pong carregando o payload, handshake de fechamento, recusa de frame sem máscara e de frame gigante com o código de fechamento correto, quatro formas de handshake inválido recusadas com 400.
- Pub/sub: broadcast alcança só os inscritos no tópico (confirmado com duas salas simultâneas), contagem correta de destinatários, e a conexão que cai sai de todos os tópicos — sem isso o broadcast seguiria escrevendo num socket morto.
- **Ponta a ponta:** `examples/chat_ws.liaf` é compilado e executado pelo teste, que abre três conexões WebSocket reais contra o binário. Verifica o evento de entrada, a mensagem chegando aos dois membros da sala, o isolamento contra uma terceira conexão noutra sala, e o evento de saída disparado por `on-close` quando um cliente cai. A contagem de clientes é conferida pela rota HTTP `/rooms/{room}/size`, servida na mesma porta.
- Servidor levantado à mão na porta 8099 servindo `public/chat.html` e respondendo `/rooms/geral/size`.

**Não verificado:** comportamento sob carga (número de conexões simultâneas, latência de broadcast com muitos inscritos) não foi medido. O pub/sub é local ao processo e não foi testado atrás de balanceador.

---

Contrato atual e especificações: [docs/SPEC.md](../linguagem/SPEC.md). Sintaxe: seção 19 de [docs/SPEC.md](../linguagem/SPEC.md). Evidências consolidadas: [STATUS.md](../STATUS.md).
