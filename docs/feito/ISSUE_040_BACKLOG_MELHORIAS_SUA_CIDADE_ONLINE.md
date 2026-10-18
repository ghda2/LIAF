# Issue 040 — Backlog de Melhorias e Correções da Linguagem LIAF (Sua Cidade Online)

**Status:** Em andamento  
**Prioridades:** P0 (Segurança/Correção), P1 (Ergonomia/Linguagem), P2 (Runtime/Ferramentas/Docs)  
**Versão Base:** `liafc v0.6.0` · **Data:** 2026-10-04  

---

## 🔴 P0 — Segurança e Correção (Concluídos)

- [x] **#1: `jwt-sign` / `jwt-verify`** — Ordem dos parâmetros corrigida e documentação alinhada em `stdlib.md` e `std/jwt.liaf`.
- [x] **#2: `serve-hybrid` sem `./public`** — Fallback gracioso no VFS cache e log informativo sem encerramento acidental do processo (`pkg/web/cache.go`).
- [x] **#3: Parâmetro de rota inválido vira `0` em silêncio** — Tratamento com resposta `400 Bad Request` automática em erro de conversão (`pkg/codegen/routes.go`, `websocket.go`).
- [x] **#4: JSON inválido em `on-err`** — Adicionado builtin `json-error(status, msg)` com escape seguro (`pkg/web/router.go`, `pkg/builtins/table_web.go`).
- [x] **#5: Tratamento de erro nos exemplos (`unwrap-or`)** — Substituído `unwrap-or` mascarado por `match` explícito no gerador e templates (`cmd/liafc/init.go`).

---

## 🟡 P1 — Ergonomia e Linguagem (Concluídos)

- [x] **#6: Placeholders numerados no SQL (`?1`, `:nome`)** — Checador aceita marcadores numerados SQLite `?NNN` (`pkg/checker/database.go`).
- [x] **#7: Fragmentos literais de SQL seguros** — Checador permite composição estática de literais via `concat` (`pkg/checker/database.go`).
- [x] **#8: Acesso a campo em resultado de chamada (`f().campo`)** — Adicionado suporte no parser linear para encadeamento pós-fixo `.campo` em qualquer expressão/chamada (`pkg/parser/linear.go`).
- [x] **#9: Rota `GET "/"` captura tudo** — Normalizada rota raiz exata `/{$}` e catch-all `/*` no router (`pkg/web/router.go`).
- [x] **#10: Serialização e escape de JSON** — Adicionado lint `W_MANUAL_JSON_FMT` contra montagem manual de JSON em `fmt` (`pkg/checker/hooks_core.go`).
- [x] **#11: Efeito `log` / desacoplamento de `effects(io)`** — Adicionados builtins `log-info` e `log-error` isentos do efeito `io` (`pkg/builtins/table_core.go`, `pkg/checker/hooks_core.go`).

---

## 🟢 P2 — Runtime, Ferramentas e Documentação (Concluídos)

- [x] **#12: Middleware / Guardas de Autenticação** — Adicionada guarda declarativa `auth-guard-jwt` em `std/auth.liaf` com 401 automático.
- [x] **#13: Rate Limit Embutido** — Adicionado rate limiter em memória via `rate-limit` e `req-client-ip` em `pkg/web/ratelimit.go`.
- [x] **#14: `mmap_size` conservador no SQLite** — Reduzido padrão de 30GB para 256MB com suporte a override via DSN em `pkg/dbdrv/sqlite.go`.
- [x] **#15: Logs Estruturados de Requisição** — Adicionado logging HTTP estruturado (JSON/texto) configurável em `pkg/web/router.go`.
- [x] **#16: Testes Nativos (`liafc test`)** — Implementado subcomando `liafc test` em `cmd/liafc/test.go`.
- [x] **#17: Mensagens informativas de inicialização** — Logs claros de inicialização, portas e caminhos no `ServeHybrid`.
- [x] **#18: Atualização de Documentação** — Documentadas transações `db-transaction`, `RETURNING`/`ON CONFLICT`, rate limit, guards e auth em `REFERENCIA.md` e `stdlib.md`.
