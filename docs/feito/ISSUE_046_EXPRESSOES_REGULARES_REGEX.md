# Issue #046 — Expressões Regulares (`std/regex`)

**Estado:** Concluída. **Criada em:** 17/10/2026. **Tipo:** stdlib/runtime.  
**Componentes:** `pkg/runtime/regex.go`, `pkg/builtins/table_string.go`, `std/regex.liaf`.  
**Evidências:** `pkg/runtime/regex_test.go`, conformidade `conformance/basics/046_regex.liaf`, `TestStdEveryModuleChecks` (`pkg/loader`) e execução ponta a ponta com `liafc run`.

---

## 1. Contexto e Resolução
Implementado o suporte nativo e seguro a expressões regulares (motor RE2 imune a ataques de ReDoS) e validadores de formatos de dados comuns.

---

## 2. O que foi entregue

1. **Motor RE2 com Cache Compilado (`pkg/runtime/regex.go`):**
   - `RegexMatch`: correspondência booleana rápida com cache thread-safe (`sync.Map`).
   - `RegexFind`: busca da primeira ocorrência retornando `Result[Option[string], string]`.
   - `RegexFindAll`: busca de todas as ocorrências retornando `Result[*List[string], string]`.
   - `RegexReplace`: substituição de todas as ocorrências.

2. **Builtins no Compilador (`pkg/builtins/table_string.go`):**
   - `regex-match`, `regex-find`, `regex-find-all`, `regex-replace`.
   - Correção do parser de tipos em `pkg/builtins/builtins.go` (`splitTypeParts`) para suportar perfeitamente tipos aninhados com parênteses (ex: `(result (option str) str)`).

3. **Módulo Standard Library (`std/regex.liaf`):**
   - Helpers: `regex-is-match`, `regex-first`, `regex-extract-all`, `regex-sub`.
   - Validadores de produção: `is-email`, `is-cpf`, `is-cnpj`, `is-slug`, `is-numeric`, `is-uuid`.
