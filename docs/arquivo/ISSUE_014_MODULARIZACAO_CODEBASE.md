# Issue #014 — Modularização da codebase

**Estado:** Aberta. **Criada em:** 16/09/2026. **Tipo:** dívida técnica / manutenibilidade.

Relatório de leitura da árvore (7.677 linhas Go em 38 arquivos, mais 4 arquivos Python).
Nenhuma alteração de código foi feita ao levantar estes dados.

Critério de corte: arquivo com mais de uma responsabilidade, **ou** função acima de ~100 linhas,
**ou** lógica duplicada entre arquivos.

## Achado transversal: a tabela de builtins não tem fonte única

A superfície de builtins da linguagem está repetida em cinco lugares:

| Arquivo | O que define | Nº de `case` |
|---|---|---|
| `pkg/checker/checker.go` (`call`, L497-763) | assinatura de tipo e efeitos | 32 |
| `pkg/codegen/library.go` | emissão Go via runtime | 31 |
| `pkg/codegen/codegen.go` (`genCall`, L470-545) | emissão Go inline/especial | 34 |
| `pkg/codegen/c/codegen.go` (`call`, L219-312) | emissão C | 33 |
| `pkg/runtime/runtime.go` | implementação Go | 24 funções |

Adicionar um builtin exige editar cinco arquivos. As contagens divergentes (32/31/34/33) indicam
que as tabelas **já estão dessincronizadas** — não foi auditado quais entradas faltam em cada uma.

**Ação proposta:** extrair `pkg/builtins/` com uma tabela declarativa (nome, aridade, tipos de
parâmetro, tipo de retorno, efeitos, template Go, template C). Checker e ambos os backends passam a
consultar essa tabela em vez de manter `switch` paralelos. É a causa raiz de boa parte do tamanho de
`checker.go` e `codegen.go`.

## Tier 1 — compensa separar

### 1. `pkg/web/cache.go` — 784 linhas (prioridade máxima)

Duas funções ocupam 58% do arquivo e são variantes uma da outra:

- `LoadDirectory` (257 linhas, L126) — carga a partir do disco.
- `LoadFS` (196 linhas, L545) — carga a partir de `embed.FS`.
- `processHTML` (39 linhas) e `processHTMLFS` (42 linhas) — mesma lógica de layout/include,
  diferindo apenas no acesso ao arquivo.

Os tipos `rawHTMLItem` e `rawMDItem`, e os mapas `assetHashes` e `collections`, são redeclarados
identicamente nos dois caminhos. Hoje qualquer correção de SSG precisa ser aplicada duas vezes.

Separação proposta:

```
pkg/web/cache.go        AssetCache, Get, Count; LoadDirectory/LoadFS como wrappers finos
pkg/web/cache_load.go   pipeline único sobre fs.FS (os.DirFS unifica disco e embed)
pkg/web/template.go     processHTML, processTemplateLoops, processVariables, resolveIncludePath
pkg/web/fingerprint.go  fingerprintHTML, assetRefRegex, mapa de hashes
pkg/web/mime.go         defaultMimes, heavyMediaExts, detectContentType
```

### 2. `pkg/checker/checker.go` — 763 linhas

`call()` tem 266 linhas (L497): é a tabela de builtins escrita como função.

```
checker.go           Checker, Check, escopos (push/pop/define/lookup)
checker_types.go     primitive, applied, name, parts, IsResult, validType, scalar, require
checker_stmt.go      statements (110 linhas), returns, block, pending/restore/merge
checker_expr.go      expr, arity, typeArg
checker_builtins.go  call — ou, preferencialmente, consumo de pkg/builtins
```

### 3. `pkg/codegen/codegen.go` — 645 linhas

`genStmt` tem 170 linhas e 10 `case`. `genRoute` tem 113 linhas e mistura extração de parâmetros de
path, `json.Unmarshal` do corpo e serialização da resposta — lógica de web dentro do gerador genérico.

```
codegen.go        Generator, Generate, genStruct, genFunc
codegen_route.go  genRoute
codegen_stmt.go   genStmt, isStandAloneStmt
codegen_expr.go   genExpr, genCall, genBinaryOp
codegen_types.go  mapType, mapTypeName, sanitizeIdent, fieldIdent
```

### 4. `pkg/parser/parser.go` — 799 linhas

Menos urgente: as funções são coesas e só `parseExpression` passa de 100 linhas (121). O problema é
volume. O padrão correto já existe no repositório em `pkg/parser/control.go`; basta continuá-lo.

```
parser.go       Parser, New, helpers de token, erros, ParseModule
parser_decl.go  parseImport, parseStruct, parseField, parseFunc, parseRoute, parseParams, parseReturns, parseEffects
parser_stmt.go  parseBody, parseStatement, parseLet/Set/Return/If/Spawn/Send/Expr
parser_expr.go  parseExpression, parseType
control.go      loop/match (mantido)
```

## Tier 2 — vale separar, sem pressa

| Arquivo | Linhas | Problema | Proposta |
|---|---|---|---|
| `cmd/liafc/main.go` | 409 | `runBuild` tem 103 linhas: tmpdir, injeção de `//go:embed`, descoberta de `go.mod` e `exec go build`. `runPublish` tem 63. | `main.go` (dispatch e usage), `build.go`, `run.go`, `check.go`, `publish.go`, `fsutil.go` (`copyDirectory`, `findGoMod`) |
| `pkg/web/server.go` | 271 | `ServeHTTP` tem 142 linhas com cinco estratégias de resolução de rota mais ETag, gzip e Range no mesmo corpo | extrair `resolveAsset(path)`, `serveDisk` e `serveRAM`; mover `ServeSite` para `serve.go` |
| `pkg/ast/ast.go` | 344 | `LoopStmt` e `MatchStmt` foram inseridos no topo, fora da ordem lógica do arquivo | `ast.go` (interfaces e Module), `decl.go`, `stmt.go`, `expr.go`, `types.go` |
| `pkg/ast/printer.go` | 379 | `printFunc` e `printRoute` (66 linhas cada) diferem quase só no cabeçalho | fatorar `printSignature`; separar `printer_expr.go` |
| `benchmarks/run_suite.py` | 349 | `request_model` cobre três provedores (OpenAI, Anthropic, Ollama) no mesmo módulo que `evaluate` e a geração de SVG | `providers.py`, `evaluate.py`, `report.py`, `run_suite.py` (CLI) |
| `pipeline/insta_to_newspaper.py` | 371 | yt-dlp, Whisper, LLM, publish e bot de Telegram num único arquivo | `download.py`, `transcribe.py`, `article.py`, `publish.py`, `bot.py` |

## Tier 3 — manter como está

Coesos; separar adicionaria ruído sem ganho: `pkg/minifier/` (quatro arquivos de 26 a 40 linhas),
`pkg/web/admin.go`, `router.go`, `compress.go`, `files.go`, `autotls.go`, `pkg/deploy/`,
`pkg/markdown/`, `pkg/token/`, `pkg/lexer/`, `pkg/diagnostic/`, `pkg/runtime/` e
`pkg/codegen/c/` (312 linhas).

## Ordem recomendada

1. `pkg/web/cache.go` — elimina duplicação que já produz divergência entre os dois caminhos de carga.
2. `pkg/builtins/` — tabela única; desinfla checker e os dois backends de uma vez.
3. `pkg/checker` e `pkg/codegen` — separação por camada, já aliviados pelo passo 2.
4. `pkg/parser`, `cmd/liafc`, `pkg/web/server.go` — redução de volume.
5. Python (`run_suite.py`) — antes de rodar as medições da issue #009.

## Pré-condição

Há 751 linhas não commitadas em 11 arquivos (`pkg/codegen/codegen.go` +217, `pkg/parser/parser.go`
+124, `pkg/checker/checker.go` +71, entre outros). Commitar esse trabalho antes de qualquer divisão
de arquivo: reorganizar arquivos com alterações pendentes embaralha o diff e inviabiliza a revisão.

**Concluída quando:** nenhum arquivo do Tier 1 passa de ~400 linhas, nenhuma função passa de ~120
linhas, a tabela de builtins tem uma única definição consultada por checker e backends, `go test ./...`
continua verde e o comportamento observável do compilador e do servidor não muda (refatoração sem
mudança de semântica).
