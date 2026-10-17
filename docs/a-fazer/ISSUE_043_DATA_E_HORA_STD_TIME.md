# Issue #043 — Módulo de Data e Hora (`std/time`)

**Estado:** Aberta. **Criada em:** 17/10/2026. **Tipo:** linguagem/stdlib.  
**Componentes:** `pkg/runtime`, `pkg/builtins`, `pkg/checker`, `std/time.liaf`.  

---

## 1. Contexto e Objetivo
Aplicações de produção exigem manipulação confiável de datas: agendamento de eventos, expiração de tokens JWT, registros de auditoria e cálculo de períodos.
Atualmente a linguagem possui apenas `time-now-unix` (efeito `clock`), faltando formatação, parsing e operações com fusos horários.

---

## 2. Proposta Técnica

1. **Estrutura `DateTime` em `std/time`:**
   - Representação tipada com ano, mês, dia, hora, minuto, segundo, nanosegundo e offset de fuso.
2. **Builtins no Runtime:**
   - `time-now() DateTime` (efeito `clock`)
   - `time-parse-iso(str) (result DateTime str)`
   - `time-format-iso(DateTime) str`
   - `time-format(DateTime, fmt str) str`
   - `time-add-seconds(DateTime, int) DateTime`
   - `time-diff-seconds(DateTime, DateTime) int`
   - `time-unix(DateTime) int`
   - `time-from-unix(int) DateTime`
3. **Casos de Aceitação:**
   - `conformance/basics/043_time_operations.liaf`
