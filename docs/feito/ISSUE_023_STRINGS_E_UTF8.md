# Issue #023: Strings e UTF-8

**Status:** Concluída
**Componente:** `pkg/builtins`, `pkg/runtime`, `pkg/checker`
**Data:** 25 de setembro de 2026
**Depende de:** #019
**Casos de aceitação:** `conformance/basics/023_*.liaf`

---

## 1. Contexto
Verificado em 24/09/2026: `(str-len "ação")` retornava `6` (bytes) e `str-slice` cortava caracteres multibyte.

## 2. Decisão Adotada: Opção A (Runes / UTF-8)
- `str-len`, `str-slice`, `str-get` e `str-index` operam em caracteres Unicode (runes UTF-8).
- `str-byte-len` provê o tamanho em bytes quando necessário para protocolos binários/HTTP.
- Novas operações implementadas: `str-contains`, `str-starts-with`, `str-ends-with`, `str-index`, `str-split`, `str-join`, `str-trim`, `str-upper`, `str-lower`, `str-replace`, `str-get`.
- Operadores `lt`, `gt`, `lte`, `gte` aceitam strings e realizam comparação lexicográfica.

## 3. Critérios de conclusão
- [x] Decisão 2.1 registrada.
- [x] Todos os casos `023_*` com `status: done`.
- [x] `str-slice` fora dos limites continua retornando `err`, agora na unidade de runes.
- [x] SPEC, AI_GUIDE e SPEC_ENXUTA atualizados.
