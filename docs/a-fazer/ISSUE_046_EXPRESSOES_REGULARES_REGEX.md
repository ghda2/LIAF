# Issue #046 — Expressões Regulares (`std/regex`)

**Estado:** Aberta. **Criada em:** 17/10/2026. **Tipo:** stdlib/runtime.  
**Componentes:** `pkg/runtime`, `pkg/builtins`, `std/regex.liaf`.  

---

## 1. Contexto e Objetivo
Validação de formatos textuais (e-mails, CPFs, números de telefone, slugs de URLs) é onipresente em desenvolvimento web e APIs.
A biblioteca padrão deve prover motor RE2 seguro (linear sem risco de ReDoS) integrado.

---

## 2. Proposta Técnica

1. **Builtins no Runtime:**
   - `regex-match(pattern str, text str) (result bool str)`: testa correspondência total/parcial com compilação e cache LRU de padrões.
   - `regex-find(pattern str, text str) (result (option str) str)`: retorna a primeira ocorrência.
   - `regex-replace(pattern str, text str, repl str) (result str str)`: substitui todas as ocorrências.
2. **Módulo `std/regex.liaf`:**
   - Utilitários comuns: `is-email(str)`, `is-slug(str)`, `is-numeric(str)`, etc.
3. **Casos de Aceitação:**
   - Teste de conformidade em `conformance/basics/046_regex.liaf`.
