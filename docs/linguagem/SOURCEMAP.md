# Mapa do código-fonte — LIAF

Guia de navegação do repositório: onde cada coisa mora e como as peças se ligam.
Para o *porquê* do design, veja [ARCHITECTURE.md](ARCHITECTURE.md); para a linguagem,
[SPEC.md](SPEC.md).

Módulo Go: `liaf` (Go 1.26). Única dependência direta relevante: `modernc.org/sqlite` (SQLite em Go puro).

---

## 1. Visão geral da raiz

**A linguagem em si** — o que define e implementa a LIAF:

| Caminho | O que é |
|---|---|
| [cmd/liafc/](../../cmd/liafc/) | CLI do compilador (`liafc`) |
| [pkg/](../../pkg/) | Compilador (lexer → codegen), runtime, drivers de banco e servidor web |
| [conformance/](../../conformance/basics/) | Suíte de aceitação: o comportamento que a linguagem promete |
| [std/](../../std/) | Biblioteca padrão escrita em LIAF, embutida no `liafc`: `auth`, `jwt`, `cors` |
| [docs/linguagem/SPEC.md](SPEC.md) | Especificação da linguagem |
| `go.mod`, `go.sum` | Módulo Go `liaf` |

**Em volta da linguagem** — usa a LIAF, mas não faz parte dela:

| Caminho | O que é |
|---|---|
| [docs/](../README.md) | Toda a documentação: `linguagem/` (guias, SPEC, este mapa), `a-fazer/` e `feito/` (issues por estado), `logs/`, `arquivo/`. `privado/` é ignorada pelo git |
| [.agents/skills/liaf/](../../.agents/skills/liaf/SKILL.md) | Skill para agentes de IA escreverem/compilarem LIAF |

`liafc.exe` na raiz é o compilador compilado localmente (ignorado pelo git).

---

## 2. Pipeline do compilador

```
arquivo.liaf
   │
   ▼
pkg/lexer ──► pkg/parser ──► pkg/ast ──► pkg/checker ──► pkg/codegen ──► go build ──► binário
 (tokens)     (S-exprs)      (árvore)    (tipos/efeitos)   (fonte Go)                   │
                                              │                                         │ linka
                                              ▼                                         ▼
                                       pkg/diagnostic                    pkg/runtime + pkg/web + pkg/dbdrv
                                       (erros em JSON)
```

`pkg/builtins` fica ao lado do checker e do codegen: é a **tabela única** de funções embutidas,
consultada pelos dois.

### Frontend

| Pacote | Arquivos | Responsabilidade |
|---|---|---|
| [pkg/token](../../pkg/token/token.go) | `token.go` | Tipos de token e palavras-chave (`LookupIdent`) |
| [pkg/lexer](../../pkg/lexer/lexer.go) | `lexer.go` | Tokenizador. Identificadores ASCII; NUL no meio da entrada é erro |
| [pkg/parser](../../pkg/parser/parser.go) | `parser.go` | `ParseModule`: exige `(module ...)`, monta declarações e expressões |
| | `control.go` | `loop`, `match`, `if` e demais formas de controle |
| | `database.go` | Formas de transação / acesso a banco (v0.4) |
| | `websocket.go` | Declaração `(ws-route ...)` com blocos em ordem fixa |
| [pkg/ast](../../pkg/ast/ast.go) | `ast.go` | Nós da árvore e tipos (`Type`, `AppliedType`, `CallExpr`, `LoopStmt`...) |
| | `printer.go` | Formatador canônico (`Format`) — usado por `liafc fmt` |
| | `typeexpr.go` | `TypeFromExpr`: converte expressão em posição de tipo para `ast.Type` |
| [pkg/loader](../../pkg/loader/loader.go) | `loader.go` | Resolve `(import ...)` (arquivos, diretórios e `std/`), ciclos, e colisões de nomes e rotas entre arquivos |

### Análise semântica

| Pacote | Arquivos | Responsabilidade |
|---|---|---|
| [pkg/checker](../../pkg/checker/checker.go) | `checker.go` | Tipos, escopos, efeitos transitivos, `Result` obrigatório |
| | `builtins_hooks.go` | Implementa `builtins.CheckContext` e registra checagens especiais (numéricos polimórficos etc.) |
| | `database.go` | Regras de tipo dos builtins de banco (issue #015) |
| | `websocket.go` | Regras de tipo de `ws-*` e `(ws-route ...)` (issue #016) |
| | `crypto.go` | Codificação literal (`"hex"`/`"base64url"`) de `sha256` e `hmac-sha256` |
| | `httpclient.go` | Método literal de `http-fetch` (`E_UNKNOWN_METHOD`) |
| [pkg/diagnostic](../../pkg/diagnostic/diagnostic.go) | `diagnostic.go` | Formato dos diagnósticos, incluindo a saída JSON para agentes |
| [pkg/builtins](../../pkg/builtins/builtins.go) | `builtins.go` | `Builtin`, `Arity`, registro e interfaces `CheckContext`/`GoContext` |
| | `table.go` | A tabela: nome, categoria, aridade, assinatura e emissão Go de cada builtin (math #020, conversões #021, strings #023, coleções #024...) |

### Geração de código

| Pacote | Arquivos | Responsabilidade |
|---|---|---|
| [pkg/codegen](../../pkg/codegen/codegen.go) | `codegen.go` | Emite Go a partir da AST checada |
| | `library.go` | `libraryCall`: consulta `pkg/builtins` para gerar a chamada Go |
| | `database.go` | Transações → padrão Go begin / defer rollback / commit |
| | `websocket.go` | Cada bloco de `ws-route` vira uma função própria |
| [pkg/codegen/c](../../pkg/codegen/c/codegen.go) | `codegen.go`, `runtime.h` | Backend C99 alternativo, experimental, **não ligado à CLI** |

---

## 3. Runtime (o que o programa gerado importa)

| Pacote | Arquivos | Responsabilidade |
|---|---|---|
| [pkg/runtime](../../pkg/runtime/runtime.go) | `runtime.go` | Biblioteca padrão tipada: `Result`, listas, mapas, FS, JSON |
| | `math.go` | Aritmética checada (`ModInt`, overflow, divisão por zero...) |
| | `strings.go` | Operações de string (UTF-8, busca, split/join) |
| | `db.go` | `DBConn` — valor por trás do tipo opaco `DBConnection` |
| | `ws.go` | Adaptadores dos builtins `ws-*` sobre `pkg/web` |
| | `http.go` | `request-header`, `request-query`, `response-set-header` |
| | `crypto.go` | SHA-256, HMAC, base64url, `random-token`, senha argon2id, `secure-eq` |
| | `httpclient.go` | `http-fetch` e `reply-*`: timeout, limite de corpo, erros sem a URL |
| [pkg/dbdrv](../../pkg/dbdrv/dbdrv.go) | `dbdrv.go` | Interface comum de drivers, protocolos em TCP puro |
| | `postgres.go`, `scram.go` | Protocolo PostgreSQL v3 + autenticação SCRAM-SHA-256 |
| | `mysql.go` | Protocolo cliente/servidor MySQL 4.1 |
| | `redis.go` | RESP; operações relacionais falham com erro claro |
| | `sqlite.go` | SQLite via `modernc.org/sqlite` |

## 4. Servidor web e deploy

| Pacote | Arquivos | Responsabilidade |
|---|---|---|
| [pkg/web](../../pkg/web/server.go) | `server.go` | Servidor HTTP; `Reload` atômico sem downtime |
| | `router.go` | Rotas dinâmicas; `Request` com headers e query, `Response` com headers validados; `HandleRaw` para WebSocket |
| | `cache.go` | Assets em RAM com fallback em disco; `SetEmbeddedFS` para binário único |
| | `files.go` | Acesso a arquivos restrito por `os.Root` |
| | `admin.go` | Endpoint autenticado de publish/reload (`LIAF_DEPLOY_TOKEN`) |
| | `compress.go`, `autotls.go` | Gzip e TLS automático (Let's Encrypt) |
| | `websocket.go` | WebSocket RFC 6455 + pub/sub em memória |
| [pkg/markdown](../../pkg/markdown/markdown.go) | `markdown.go` | Markdown com front matter → HTML (SSG) |
| [pkg/minifier](../../pkg/minifier/minifier.go) | `html.go`, `css.go`, `js.go` | Minificação dos assets servidos (desativável com `LIAF_MINIFY=false`) |
| [pkg/deploy](../../pkg/deploy/deploy.go) | `deploy.go` | Unidade systemd + rota via Caddy Admin API |

---

## 5. CLI — [cmd/liafc](../../cmd/liafc/main.go)

| Arquivo | Comandos |
|---|---|
| `main.go` | `check` (com `--json`), `emit`, `build` (`-o`, `--embed[=dir]`), `run`, `publish`, `service` |
| `fmt.go` | `fmt` — atenção: a AST não guarda comentários, então formatar apaga comentários |
| `deploy.go` | `deploy` |

Fluxo do `build`: `parseAndCheck` → codegen gera `main.go` num diretório temporário `liaf_build_*`
→ (opcional) copia a pasta de assets para `public/` e injeta um `//go:embed` → `go build` usando o
`go.mod` deste repositório (`findGoMod`). Por isso compilar exige Go e o repo; o binário final não.

---

## 6. Testes

| Onde | Cobre |
|---|---|
| `pkg/*/..._test.go` | Unitários por pacote. Sufixos `_v03`/`_v04` agrupam os recursos de cada versão |
| [pkg/codegen/integration_test.go](../../pkg/codegen/integration_test.go) | Compila e executa programas de ponta a ponta |
| [pkg/codegen/conformance_test.go](../../pkg/codegen/conformance_test.go) | Roda tudo em `conformance/basics/` |
| [conformance/basics/](../../conformance/basics/) | 47 casos. Prefixo = issue (`000_` e `020_` aritmética, `021_` conversões, `022_` erros aritméticos, `023_` strings, `024_` coleções, `025_` if-valor, `026_` ambiente/IO, `027_` crypto, std/jwt e nomes de campo, `028_` sintaxe compacta, `fmt`, `unwrap-or` e `match` como valor) |

Formato de um caso de conformance (cabeçalho em comentários):

```liaf
;; issue: #022
;; status: done          ; pending = pulado enquanto falha; quebra se passar
;; out: div: divisao por zero
(let z int 0)
(match (div 10 z) (ok v (println v)) (err e (println e)))
```

Diretivas opcionais: `effects:`, `stdin:`, `exit:`. Corpo sem `(module` é embrulhado num `main`.

Rodar tudo: `go test ./...`

---

## 7. Programas de teste — [pkg/codegen/testdata/](../../pkg/codegen/testdata/)

Programas LIAF completos que os testes de `pkg/codegen` compilam e executam:

| Tema | Arquivos |
|---|---|
| Núcleo | `loops`, `collections`, `math`, `result`, `concurrency`, `fs_json` |
| HTTP / APIs | `task_api_v03`, `pedidos_api` (Bearer, query, CORS, cookie, WebSocket autenticado), `cobranca_pix` (cliente HTTP) |
| Bancos | `db_postgres`, `db_redis` |
| WebSocket | `chat_ws` |
| Biblioteca padrão | `std_web` (`std/auth` e `std/cors` testados pela rede) |

Os servidores servem `"./public"` **relativo ao diretório atual**; os testes criam essa pasta vazia ao
lado do binário.

---

## 8. Onde mexer para...

| Tarefa | Arquivos |
|---|---|
| Adicionar um builtin simples | `pkg/builtins/table.go` (+ função em `pkg/runtime/*.go` se precisar) + caso em `conformance/basics/` |
| Builtin com regra de tipo especial | acima + `RegisterSpecialCheck` em `pkg/checker/builtins_hooks.go` |
| Nova forma sintática | `pkg/token` → `pkg/parser` → `pkg/ast` (nó + `printer.go`) → `pkg/checker` → `pkg/codegen` |
| Novo driver de banco | `pkg/dbdrv/<driver>.go` + registro em `dbdrv.go` |
| Comportamento do servidor | `pkg/web` |
| Novo subcomando da CLI | `cmd/liafc/main.go` (switch em `main`) |
| Documentar para modelos | `docs/linguagem/AI_GUIDE.md`, `.agents/skills/liaf/SKILL.md` |

---

## Observações

- Os maiores arquivos (`pkg/builtins/table.go`, `pkg/dbdrv/mysql.go`, `pkg/parser/parser.go`,
  `pkg/web/cache.go`, `pkg/codegen/codegen.go`, `pkg/checker/checker.go`) concentram mais
  responsabilidade do que o ideal; ARCHITECTURE.md já os lista como candidatos a divisão.
