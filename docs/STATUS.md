# Status das issues — LIAF

**Conferido em 26/09/2026**, contra o código e os testes da branch `feat/liaf-v03` (commit `64cf5d4` e
alterações posteriores). `go test ./...` e `go vet ./...` passam; 47 casos em `conformance/basics/`.

O que fazer a seguir: [PROXIMOS_PASSOS.md](a-fazer/PROXIMOS_PASSOS.md). Como criar uma issue nova: seção 2 da
[#031](a-fazer/ISSUE_031_GUIA_PROXIMAS_ISSUES.md).

Como as pastas estão organizadas: [README.md](README.md).

## A fazer — [a-fazer/](a-fazer/)

| Issue | Estado | O que falta de fato |
|---|---|---|
| [#012 Ergonomia](a-fazer/ISSUE_012_SYNTAX_SUGAR_AND_TOKEN_EFFICIENCY.md) | Parcial | `s.campo` não existe (dá `E_UNEXPECTED_TOKEN`); `(eq x false)` ainda é aceito |
| [#030 Modularização](a-fazer/ISSUE_030_MODULARIZACAO_MONOLITOS.md) | Aberta, nada feito | `web/cache.go` (784, duplicação de 88%), `builtins/table.go` (1.511), `parser.go` (1.062), `checker.go` (745), `codegen.go` (747) |
| [#032 Compilar fora do repositório](a-fazer/ISSUE_032_COMPILADOR_AUTONOMO_RUNTIME_EMBUTIDO.md) | Aberta | `liafc build` fora da árvore falha: o código gerado importa `liaf/pkg/runtime` |
| [#010 Backend C](a-fazer/ISSUE_010_GO_INDEPENDENCE_NATIVE_BACKEND.md) | Parcial | Testes do emissor C passam; sem `--backend` na CLI, sem JSON/web, sem GC, nunca rodado sem Go |
| [#015 Bancos](a-fazer/ISSUE_015_DATABASE_DRIVERS_NETWORK.md) | Implementada | Nunca testada contra PostgreSQL, MySQL ou Redis de verdade |
| [#007 Deploy](a-fazer/ISSUE_007_AUTONOMOUS_AI_DEPLOY_PIPELINE.md) | Implementada | Esta revisão não foi instalada em servidor |
| [#001 SSG](a-fazer/ISSUE_001_SSG_EVOLUTION.md) | Implementada | Metas de desempenho (RAM <5 MB, reload <5 ms) nunca medidas sob carga definida |
| [#006 Result](a-fazer/ISSUE_006_STRUCTURED_ERROR_HANDLING.md) | Implementada | Revisão de fluxo entre ramos e loops, sem registro de andamento desde 16/09 |
| [#017 Tempo real](a-fazer/ISSUE_017_REALTIME_CHANNELS_RESILIENCY.md) | Proposta | Nada implementado |
| [#033 Sintaxe linear](a-fazer/ISSUE_033_LIAF_V06_SINTAXE_LINEAR_SEM_PARENTESES.md) | Proposta | Nada implementado. Concorre com a #028 pelo mesmo objetivo; decidir depois do benchmark |
| [#011 Self-hosting](a-fazer/ISSUE_011_SELF_HOSTING_COMPILER.md) | Pendente | Depende de linguagem e backend estáveis |
| [#031 Guia](a-fazer/ISSUE_031_GUIA_PROXIMAS_ISSUES.md) | Aberta | Itens 0.1 e 0.2 feitos; faltam transformar as seções B e C em issues |
| **#029** `std/storage` | Concluída / Pendente commit | Upload multipart/raw, compressão WebP sem perdas e serviço de imagens |

## Feito — [feito/](feito/)

| Issue | Evidência |
|---|---|
| [#028 Sintaxe compacta v0.5](feito/ISSUE_028_SINTAXE_COMPACTA_E_EXPRESSOES.md) | Parser, checker, `liafc fmt` canônico compacto, documentação na `SPEC.md` e `SKILL.md` |
| [#009 Benchmark AI-First](feito/ISSUE_009_AI_FIRST_BENCHMARK_AND_SHOWCASE.md) | Medição e comparação LIAF vs Python em consumo de tokens realizada |
| [#027 Segurança web e std](feito/ISSUE_027_SEGURANCA_WEB_E_STD.md) | 7 casos `027_*`, `TestExecutePedidosAPI`, `TestExecuteCobrancaPix`, `TestExecuteStdWeb`. Pendências de auth listadas na issue |
| [#026 Sistema](feito/ISSUE_026_INTERACAO_COM_O_SISTEMA.md) | Casos `026_*` |
| [#025 `if` como expressão](feito/ISSUE_025_IF_COMO_EXPRESSAO.md) | Caso `025_if_valor` |
| [#024 Coleções e `option`](feito/ISSUE_024_COLECOES_COMPLETAS.md) | Casos `024_*` |
| [#023 Strings e UTF-8](feito/ISSUE_023_STRINGS_E_UTF8.md) | Casos `023_*` |
| [#022 Semântica numérica](feito/ISSUE_022_SEMANTICA_NUMERICA_SEGURA.md) | Casos `022_*` |
| [#021 Conversões](feito/ISSUE_021_CONVERSOES_DE_TIPO.md) | Casos `021_*` |
| [#020 Aritmética](feito/ISSUE_020_ARITMETICA_BASICA.md) | Casos `020_*` |
| [#019 Tabela de builtins](feito/ISSUE_019_TABELA_UNICA_DE_BUILTINS.md) | `pkg/builtins/table.go` (virou monólito: ver #030) |
| [#018 Diagnósticos](feito/ISSUE_018_DIAGNOSTICOS_PRECIOSOS_AUTO_CURA.md) | `let` sem tipo; o cabeçalho dizia "Aberta" por engano |
| [#016 WebSockets](feito/ISSUE_016_WEBSOCKETS_BIDIRECTIONAL.md) | Teste ponta a ponta com `chat_ws`. Carga nunca medida |
| [#013 I/O atômico e JSON](feito/ISSUE_013_STDLIB_ATOMIC_FS_AND_CLEAN_JSON.md) | Teste de execução |
| [#008 Streaming](feito/ISSUE_008_HYBRID_STORAGE_AND_STREAMING.md) | Range 206/416 testados |
| [#002–#005](feito/) | Loops, coleções, rotas HTTP, FS/JSON |
| [#014 Modularização](arquivo/ISSUE_014_MODULARIZACAO_CODEBASE.md) | **Substituída** pela #030; está em `arquivo/` |
