# Issue #024: Coleções Completas e `option`

**Status:** Concluída
**Componente:** `pkg/builtins`, `pkg/parser`, `pkg/checker`, `pkg/codegen`
**Data:** 25 de setembro de 2026
**Depende de:** #019
**Casos de aceitação:** `conformance/basics/024_*.liaf`

---

## 1. Contexto
Implementadas operações essenciais de coleções que faltavam na linguagem e o tipo `option`.

## 2. Decisão Adotada: Opção A (`option`)
- Literal de lista: `(list 1 2 3)`
- Operações de lista: `(list-remove l i)`, `(list-pop l)`, `(list-sort l)`, `(list-contains l x)`
- Operações de mapa: `(map-delete m k)`, `(map-keys m)` (chaves ordenadas deterministicamente)
- Tipo `option`: `(option T)` implementado com `(some x)`, `(none T)` e `(match opt (some v ...) (none ...))`

## 3. Critérios de conclusão
- [x] Decisão sobre `option` registrada; SPEC e AI_GUIDE coerentes com o checker.
- [x] Todos os casos `024_*` com `status: done`.
- [x] `map-keys` determinístico e ordenado.
