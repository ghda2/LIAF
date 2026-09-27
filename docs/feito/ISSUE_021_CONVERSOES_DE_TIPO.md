# Issue #021: Conversões entre Tipos Primitivos

**Status:** Concluída e validada
**Componente:** `pkg/builtins`, `pkg/runtime`
**Data:** 24 de setembro de 2026
**Depende de:** #019
**Casos de aceitação:** `conformance/basics/021_*.liaf`

---

## 1. Contexto
A SPEC diz que "não existe coerção implícita; conversões devem ser expressas por funções do núcleo",
mas só existem `str-from-int`, `float-from-int` e `int-from-str`. Não há como converter `float` para
nenhum outro tipo. `int-from-str` devolve a mensagem crua do Go
(`strconv.ParseInt: parsing "12x": invalid syntax`).

## 2. Proposta
| Forma | Tipo |
|---|---|
| `(int-from-float x)` | `float -> int`, trunca em direção ao zero |
| `(str-from-float x)` | `float -> str`, menor representação exata (`strconv.FormatFloat(x, 'g', -1, 64)`) |
| `(float-from-str s)` | `str -> (result float str)` |
| `(str-from-bool b)` | `bool -> str` |
| `(bool-from-str s)` | `str -> (result bool str)`, aceita só `true` e `false` |

Mensagens de erro próprias da LIAF, no formato `<builtin>: "<entrada>" nao e um <tipo> valido`.
`int-from-float` com `NaN`, `Inf` ou valor fora do intervalo de `int`: ver #022.

## 3. Critérios de conclusão
- [x] Todos os casos `021_*` com `status: done`.
- [x] Nenhuma mensagem de erro do runtime Go vaza para o programa LIAF nos builtins de conversão.
- [x] SPEC, AI_GUIDE e SPEC_ENXUTA atualizados.
