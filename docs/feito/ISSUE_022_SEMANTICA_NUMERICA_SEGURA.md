# Issue #022: Semântica Numérica Segura

**Status:** Concluída
**Componente:** `pkg/builtins`, `pkg/checker`, `pkg/runtime`
**Data:** 25 de setembro de 2026
**Depende de:** #019, #020
**Casos de aceitação:** `conformance/basics/022_*.liaf`

---

## 1. Contexto
Verificado em 24/09/2026:
- `(div 10 z)` com `z = 0` derrubava o processo com `panic: runtime error: integer divide by zero`.
- `(add 9223372036854775807 1)` imprimia `-9223372036854775808` sem aviso.
- `(div 1.0 0.0)` imprime `+Inf` (comportamento IEEE 754, aceitável).

## 2. Decisão Adotada: Opção A
- **Divisão e Resto (`div`/`mod` de `int`)**: Retornam `(result int str)`. Se o divisor for zero, produzem erro amigável (`div: divisao por zero` e `mod: divisao por zero`). Divisão de `float` continua retornando `float` (IEEE 754).
- **Overflow de `int`**: Aborta com erro LIAF e código de saída 1 (`liaf: overflow na adicao/subtracao/multiplicacao de inteiros`).

## 3. Critérios de conclusão
- [x] Decisões registradas nesta issue e na SPEC.
- [x] Casos `022_*` ajustados à decisão e com `status: done`.
- [x] Runtime Go e backend C implementados com proteção de overflow e divisão/resto segura.
