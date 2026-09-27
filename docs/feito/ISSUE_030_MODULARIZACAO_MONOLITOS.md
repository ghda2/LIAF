# Issue #030 — Modularização dos monólitos (continuação da #014)

**Estado:** Concluída (100%). **Criada em:** 26/09/2026. **Tipo:** dívida técnica / manutenibilidade.
**Substitui como plano de execução:** [#014](../arquivo/ISSUE_014_MODULARIZACAO_CODEBASE.md), cujo relatório é de
16/09/2026 e ficou defasado.

Levantamento refeito sobre a árvore atual: **14.214 linhas Go em 56 arquivos** (sem testes), mais
6.811 linhas de testes. Nenhuma alteração de código foi feita ao levantar estes dados. O tamanho de
cada função foi medido com `go/parser`: **21 de 549 funções** têm 80 linhas ou mais.

Critério de corte (o mesmo da #014): arquivo com mais de uma responsabilidade, **ou** função acima de
~100 linhas, **ou** lógica duplicada.

## O que mudou desde a #014

| Item da #014 | Estado em 26/09 |
|---|---|
| Tabela de builtins em 5 lugares | **Resolvido em parte:** existe `pkg/builtins/` com tabela única. Mas o `table.go` virou um monólito novo (1.339 linhas, um único `init`) e as checagens especiais foram para outro (`builtins_hooks.go`, 403 linhas) |
| `pkg/web/cache.go` duplicado | **Inalterado:** 784 linhas, mesma duplicação |
| `pkg/checker/checker.go` | 763 → 753 linhas; o `call()` gigante saiu, mas `expr`, `Check` e `statements` passam de 150 linhas cada |
| `pkg/codegen/codegen.go` | 645 → 678 linhas |
| `pkg/parser/parser.go` | **799 → 1.047 linhas** |
| `cmd/cmd/liafc/main.go` | 409 → 392 linhas; `runBuild` continua com 103 |

## Prioridade alta — 6 arquivos [TODOS CONCLUÍDOS]

### 1. `pkg/web/cache.go` — [CONCLUÍDO - Commit 5db40f4]

Duplicação medida linha a linha (sem comentários e linhas vazias):

- `LoadDirectory` (197 linhas) × `LoadFS` (169): **150 linhas idênticas, 88% de `LoadFS`**.
- `processHTML` (33) × `processHTMLFS` (40): **26 linhas idênticas, 78% do menor**.

Qualquer correção de SSG precisa ser feita duas vezes, e nada garante que foi. É o único item desta
issue que é risco, não só desconforto.

```
cache.go        AssetCache, Get, Count; LoadDirectory = LoadFS(os.DirFS(dir))
cache_load.go   pipeline único sobre fs.FS
template.go     processHTML (versão única), processTemplateLoops, processVariables, resolveIncludePath
fingerprint.go  fingerprintHTML e mapa de hashes
mime.go         detectContentType e tabelas de extensão
```

Cuidado: o caminho de disco hoje passa pela restrição de `os.Root` (`files.go`). Ao unificar sobre
`fs.FS`, confirmar que o sandbox continua valendo (`TestSandboxedVFS`).

### 2. `pkg/builtins/table.go` — [CONCLUÍDO - Commit 7196eb2]

Uma única função `init` de **1.311 linhas** registra todos os builtins, de cerca de 20 categorias:

| Categoria | Builtins |
|---|---|
| string | 17 |
| collection | 17 |
| math | 10 |
| web | 10 |
| conversion | 8 |
| crypto | 8 |
| ws | 7 |
| system | 7 |
| fs | 6 |
| db | 6 |
| http-client | 4 |
| result, option, object, json, io, core, concurrency | 1–2 cada |

Divisão mecânica, de baixo risco: um arquivo por grupo, cada um com seu `init`.

```
table_core.go        core, io, result, option, object, json, concurrency
table_math.go        math
table_conversion.go  conversion
table_string.go      string
table_collection.go  collection
table_system.go      system, fs
table_web.go         web, ws, http-client
table_db.go          db
table_crypto.go      crypto
```

Verificar se a ordem de `orderedBuiltins` importa para alguma saída (listagem, documentação): com
vários `init`, a ordem passa a seguir o nome dos arquivos.

### 3. `pkg/checker/builtins_hooks.go` — [CONCLUÍDO - Commit 586bd2e]

`init` de 383 linhas com as 15 checagens especiais. Seguir os mesmos grupos do item 2, ao lado dos
arquivos que já existem (`crypto.go`, `httpclient.go`, `database.go`, `websocket.go`).

### 4. `pkg/parser/parser.go` — [CONCLUÍDO - Commit a507a60]

Declarações, comandos, expressões e tipos no mesmo arquivo; `parseExpression` tem 126 linhas. O
padrão certo já existe (`control.go`, `database.go`, `websocket.go`).

```
parser.go       Parser, New, helpers de token, erros, ParseModule
decl.go         parseTopLevel, parseImport, parseStruct, parseField, parseFunc, parseRoute,
                parseParams, parseReturns, parseEffects, parseBody
stmt.go         parseStatement, parseLet/Set/Return/If/Spawn/Send/ExprStmt
expr.go         parseExpression, parseIfExpr, parseCallArgs
types.go        parseType
```

### 5. `pkg/checker/checker.go` — [CONCLUÍDO - Commit 48af565]

| Função | Linhas | `case` |
|---|---|---|
| `expr` | 164 | 14 |
| `Check` | 154 | 5 |
| `statements` | 152 | 14 |

```
checker.go   Checker, Check (entrada e registro de declarações)
scope.go     push, pop, define, lookup, consume, pending, restore, merge
stmt.go      statements, returns, block
expr.go      expr, call, arity, typeArg
types.go     primitive, applied, name, parts, IsResult, validType, scalar, require
```

`Check` e `statements` também precisam ser quebradas por dentro, não só movidas.

### 6. `pkg/codegen/codegen.go` — [CONCLUÍDO - Commit 7bd4eea]

`genStmt` (159 linhas, 14 `case`) e `genRoute` (136 linhas, que mistura extração de parâmetro de
caminho, `json.Unmarshal` do corpo e serialização da resposta).

```
codegen.go   Generator, New, Generate, genStruct, genFunc
routes.go    genRoute (e extrair os três passos dele em funções)
stmt.go      genStmt, isStandAloneStmt
expr.go      genExpr, genCall, genBinaryOp
types.go     mapType, mapTypeName, opaqueGoTypes, sanitizeIdent, fieldIdent
```

## Prioridade média — 4 arquivos [TODOS CONCLUÍDOS]

Grandes, mas coesos. Dividir ajuda a navegar.

| Arquivo | Linhas originais | Divisão realizada | Estado |
|---|---|---|---|
| `pkg/dbdrv/mysql.go` | 899 (33 funções) | `mysql.go` (Conn: Query, Exec, Begin...), `mysql_auth.go` (handshake, authResponse, finishAuth), `mysql_rows.go` (readColumns, decodificação binária de valores e datas), `mysql_wire.go` (write, read, lenenc, myError) | [CONCLUÍDO - Commit 1af74b7] |
| `pkg/dbdrv/postgres.go` | 546 | `postgres.go`, `postgres_auth.go`, `postgres_rows.go`, `postgres_wire.go` | [CONCLUÍDO - Commit c43741b] |
| `pkg/web/websocket.go` | 516 | RFC 6455 (`Upgrade`, frames, `WSConn`) em `websocket.go` e pub/sub (`wsHub`, `WSJoin`, `WSBroadcast`...) em `pubsub.go` | [CONCLUÍDO - Commit 881e96c] |
| `cmd/liafc/main.go` | 392 | `main.go` (dispatch e usage), `build.go` (`runBuild`), `run.go`, `check.go`, `publish.go`, `fsutil.go` (`copyDirectory`, `findGoMod`) | [CONCLUÍDO - Commit 97c745b] |

## Funções longas em arquivos que estão bem [TODAS CONCLUÍDAS]

Refatoradas por dentro, sem mover arquivo:

| Função | Arquivo | Linhas originais | Resolução | Estado |
|---|---|---|---|---|
| `Server.ServeHTTP` | `pkg/web/server.go` | 142 | Extraídos `findAsset`, `serveDiskAsset` e `serveRAMAsset` | [CONCLUÍDO - Commit 867c401] |
| `RenderHTML` | `pkg/markdown/markdown.go` | 134 | Encapsulado em `markdownRenderer` com handlers por bloco | [CONCLUÍDO - Commit 868631b] |
| `Parser.parseMatch` | `pkg/parser/control.go` | 123 | Decomposto em `parseMatchSome` e `parseMatchOk` | [CONCLUÍDO - Commit 1b96c32] |
| `Server.serveAdmin` | `pkg/web/admin.go` | 119 | Decomposto em `authenticateAdmin`, `handleReload`, `handlePublish` | [CONCLUÍDO - Commit 867c401] |
| `Checker.wsRoute` | `pkg/checker/websocket.go` | 109 | Decomposto em `validateWSEffects`, `validateWSParams` e `checkWSBodies` | [CONCLUÍDO - Commit 342fbdb] |
| `Generator.genWSRoute` | `pkg/codegen/websocket.go` | 96 | Extraídos `genWSPathParams` e `genWSLoop` | [CONCLUÍDO - Commit 33ed5f8] |

## Manter como está

`pkg/ast/printer.go` (487, coeso), `pkg/ast/ast.go` (66 funções pequenas), `pkg/runtime/*`,
`pkg/loader/loader.go`, `pkg/minifier/`, `pkg/deploy/`, `pkg/lexer/`, `pkg/token/`, `pkg/diagnostic/`,
`pkg/codegen/c/`.

## Ordem recomendada

1. `pkg/web/cache.go` — elimina a duplicação.
2. `pkg/builtins/table.go` + `pkg/checker/builtins_hooks.go` — mecânico, e os dois crescem a cada
   builtin novo.
3. `pkg/parser`, `pkg/checker`, `pkg/codegen` juntos — seguem os mesmos cortes (declaração, comando,
   expressão, tipo).
4. Prioridade média e funções longas, quando alguém for mexer nesses arquivos.

## Pré-condição

A branch `feat/liaf-v03` tem muito trabalho não commitado desde `6b5bd73` (ver
`docs/logs/LOG_2026-09-26.md`). **Commitar antes de qualquer divisão:** mover arquivos com alterações
pendentes embaralha o diff e impede a revisão. Cada item acima deve ser um commit próprio, só com
movimentação e quebra de funções, sem mudança de comportamento.

## Rede de segurança

`go test ./...`, os 43 casos de `conformance/basics/` (compilados e executados) e os testes ponta a
ponta de `pkg/codegen` (`TestExecutePedidosAPI`, `TestExecuteCobrancaPix`, `TestExecuteV04ChatWS`...)
cobrem os pacotes afetados. Se tudo continuar verde, o comportamento não mudou.

**Concluída quando:** nenhum arquivo de prioridade alta passa de ~400 linhas, nenhuma função passa de
~120 linhas, `cache.go` não tem mais caminho duplicado, `go test ./...` continua verde (também com
`-race` em `pkg/runtime` e `pkg/web`) e o comportamento observável do compilador e do servidor não
muda.
