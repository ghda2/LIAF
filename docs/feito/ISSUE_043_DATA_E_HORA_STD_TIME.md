# Issue #043 — Módulo de Data e Hora (`std/time`)

**Estado:** Concluída. **Criada em:** 17/10/2026. **Tipo:** linguagem/stdlib.  
**Componentes:** `pkg/runtime/time.go`, `pkg/builtins/table_system.go`, `std/time.liaf`.  
**Evidências:** `pkg/runtime/time_test.go`, caso de aceitação `conformance/basics/043_time_operations.liaf`, `TestStdEveryModuleChecks` (`pkg/loader`) e execução ponta a ponta com `liafc run`.

---

## 1. Contexto e Resolução
Implementado o suporte nativo e padronizado a manipulação de data, hora e timestamps Unix na linguagem LIAF.

---

## 2. O que foi entregue

1. **Funções no Runtime (`pkg/runtime/time.go`):**
   - `TimeNowUnix()`, `TimeNowMs()`, `TimeNowISO()` (efeito `clock`).
   - `TimeFormatISO()`, `TimeParseISO()` com fallback tolerante a ISO 8601, RFC 3339 e datas simples (`YYYY-MM-DD`, `DD/MM/YYYY`).
   - `TimeFormat()` com tradução automática de layouts amigáveis (`YYYY-MM-DD HH:mm:ss`, etc.) para o padrão Go.
   - Cálculos e extrações: `TimeAddSeconds()`, `TimeDiffSeconds()`, `TimeYear()`, `TimeMonth()`, `TimeDay()`, `TimeHour()`, `TimeMinute()`, `TimeSecond()`, `TimeWeekday()`.

2. **Builtins no Compilador (`pkg/builtins/table_system.go`):**
   - `time-now`, `time-now-unix`, `time-now-ms`, `time-format-iso`, `time-parse-iso`, `time-format`, `time-add-seconds`, `time-diff-seconds`, `time-year`, `time-month`, `time-day`, `time-hour`, `time-minute`, `time-second`, `time-weekday`.

3. **Módulo Standard Library (`std/time.liaf`):**
   - Utilitários: `time-now`, `time-now-unix`, `time-parse`, `time-format`, `time-add`, `time-add-minutes`, `time-add-hours`, `time-add-days`, `time-diff`, `time-is-past`, `time-is-future`, etc.
