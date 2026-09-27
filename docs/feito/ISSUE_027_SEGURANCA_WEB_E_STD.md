# Issue #027 — Segurança web e biblioteca padrão (retroativa)

**Estado:** Concluída. **Criada em:** 26/09/2026, depois da entrega, para registrar o que foi feito
(item 0.1 da [#031](../a-fazer/ISSUE_031_GUIA_PROXIMAS_ISSUES.md)).
**Componentes:** `pkg/runtime`, `pkg/checker`, `pkg/builtins`, `pkg/loader`, `pkg/parser`, `std/`.
**Casos de aceitação:** `conformance/basics/027_*.liaf` (7), mais os testes ponta a ponta
`TestExecutePedidosAPI`, `TestExecuteCobrancaPix` e `TestExecuteStdWeb`.

Detalhes de cada passo, com as decisões e o que foi medido: `docs/logs/LOG_2026-09-26.md`, §6 a §11.

---

## O que foi entregue

| Frente | Entrega |
|---|---|
| HTTP | `request-header`, `request-query`, `response-set-header`; `Request` no handshake de `ws-route` |
| Crypto | `sha256`, `hmac-sha256`, `secure-eq`, `base64url-*`, `random-token`, `password-hash`/`-verify` (argon2id) |
| Módulos | `(import ...)`, `std/` embutida no `liafc`, `E_IMPORT_NOT_FOUND`, `E_DUPLICATE_DECL`, `E_DUPLICATE_ROUTE` |
| Std | `std/auth`, `std/jwt`, `std/cors`, escritos em LIAF |
| Parser | Palavras reservadas como nome de campo (`sub`, o claim do JWT) |
| Cliente HTTP | `http-fetch` e `reply-*` |

## Revisão da std em 26/09/2026 (noite)

- `jwt-sign` passou a devolver `(result str str)` e, junto com `jwt-verify`, recusa segredo com menos
  de 32 bytes (RFC 7518 §3.2). **Quebra de compatibilidade**; os casos foram ajustados.
- `auth-bearer-token` aceita o esquema em qualquer caixa (RFC 7235) e diz "empty bearer token" para
  `Bearer` sozinho (o HTTP apaga os espaços do fim do header).
- `cors-headers-credentials` e `cors-preflight-credentials` para API autenticada por cookie.
- `TestExecuteStdWeb` (`pkg/codegen/codegen_std_test.go`): primeira cobertura direta de `std/auth` e
  `std/cors`, pela rede, porque a LIAF não tem como montar um `Request` à mão.

## Renumeração

Os casos `028_jwt_*` e `029_campo_reservado` viraram `027_jwt_*` e `027_campo_reservado`, liberando
os prefixos da #028 e de uma futura #029.

## Atualização em 27/09/2026: Cookies e Headers Acumulativos Entregues

1. **`request-cookie`**: builtin adicionado para extração direta de cookies da requisição HTTP (`rt.RequestCookie`).
2. **`std/cookie`**: helpers canônicos `cookie-make`, `cookie-expire`, `response-set-cookie` e `response-delete-cookie` com suporte a `HttpOnly`, `Secure` e `SameSite`.
3. **`response-add-header` & `Vary` acumulado**: `Response.AddHeader` implementado no runtime web (`pkg/web/router.go`), garantindo que múltiplos `Set-Cookie` e cabeçalhos `Vary` acumulem sem sobrescrever. Testado em `TestExecuteStdWeb/cookies_e_vary`.

## O que permanece pendente (para issues futuras de Middleware/Sessão)

1. Sessão guardada no servidor com revogação e CSRF.
2. Limite de tentativas no login (rate limit).
3. `nbf` e tolerância de relógio no `jwt-verify`.

Autorização por papéis e a proteção de todas as rotas de uma vez dependem de **middleware**, que
depende de **closures** (seção B da #031).
