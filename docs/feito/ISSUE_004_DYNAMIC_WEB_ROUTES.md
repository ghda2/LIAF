# Issue #004: Rotas HTTP Dinâmicas e Handlers de API no Web Engine

**Status:** Concluida e validada localmente
**Componente:** `pkg/web`, `pkg/ast`, `pkg/codegen`  
**Data:** 16 de setembro de 2026  

---

## 1. Contexto e Objetivo
O motor web (`pkg/web`) hoje é focado em entrega ultra-rápida de assets estáticos e páginas compiladas em RAM com `(serve-site ...)`. Para que a LIAF crie serviços de backend e APIs REST/JSON completos, precisamos de declaração de rotas dinâmicas acopladas a funções de callback.

---

## 2. Especificações de Sintaxe

### 2.1. Handlers de Rota
Funções receptoras de requisição e construtoras de resposta:
```liaf
(fn handle-users
  (params (req Request))
  (returns Response)
  (effects io)
  (body
    (return (call json-response 200 "{\"status\":\"ok\"}"))))
```

### 2.2. Registro de Rotas e Servidor Híbrido
```liaf
(do (call http-get "/api/users" handle-users))
(do (call http-post "/api/submit" handle-submit))
(do (call serve-hybrid "./public" "8080"))
```

---

## 3. Tarefas de Implementação
- [x] Implementar tipos nativos `Request` e `Response` no runtime.
- [x] Adicionar roteador dinâmico baseado em `http.ServeMux` em `pkg/web`.
- [x] Expor built-ins: `http-get`, `http-post`, `http-put`, `http-delete`, `json-response`.
- [x] Exemplo em `examples/api_server.liaf`.


Contrato atual e ajustes de sintaxe: [docs/SPEC.md](../linguagem/SPEC.md). Evidencias consolidadas: [STATUS.md](../STATUS.md).
