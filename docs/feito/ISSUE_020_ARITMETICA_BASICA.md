# Issue #020: Aritmética Básica

**Status:** Concluída e validada
**Componente:** `pkg/builtins`, `pkg/lexer`, `pkg/parser`, `pkg/checker`, `pkg/codegen`
**Data:** 24 de setembro de 2026
**Depende de:** #019
**Casos de aceitação:** `conformance/basics/020_*.liaf`

---

## 1. Contexto
Teste empírico em 24/09/2026: `mod`, `neg`, `abs`, `min`, `max`, `pow`, `sqrt`, `floor`, `ceil` e
`round` retornam `E_UNDEFINED_SYMBOL`. `add`, `mul`, `and` e `or` aceitam só dois argumentos.
Os literais `1e3` e `0xff` dão erro de sintaxe. `mod` falta em praticamente qualquer programa real.

## 2. Proposta
| Forma | Tipos | Observação |
|---|---|---|
| `(mod a b)` | `int int -> int` | Resto truncado, como em Go: `(mod -7 3)` = `-1`. Divisor zero: ver #022. |
| `(neg x)` | `int -> int`, `float -> float` | |
| `(abs x)` | `int -> int`, `float -> float` | |
| `(min a b)`, `(max a b)` | mesmo tipo numérico | |
| `(pow a b)` | `int int -> int`, `float float -> float` | Expoente `int` negativo: erro de checagem se literal; em runtime, ver #022. |
| `(sqrt x)` | `float -> float` | |
| `(floor x)`, `(ceil x)`, `(round x)` | `float -> float` | `round` arredonda a metade para longe do zero (`math.Round`). Para obter `int`, usar `int-from-float` (#021). |
| `add`, `mul`, `and`, `or` | variádicos, mínimo 2 | `sub` e `div` continuam binários, para evitar ambiguidade de associatividade. |

Literais: notação científica (`1e3`, `2.5e-3`) é `float`; hexadecimal (`0xff`) é `int`.

## 3. Critérios de conclusão
- [x] Todos os casos `020_*` com `status: done`.
- [x] Mistura de `int` e `float` continua proibida, com diagnóstico que sugere `float-from-int`.
- [x] SPEC, `docs/AI_GUIDE.md` e `teste_ia/SPEC_ENXUTA.md` atualizados.
- [x] Backend C: implementado ou marcado como não suportado na tabela.
